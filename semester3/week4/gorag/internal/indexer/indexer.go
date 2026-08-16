// Package indexer coordinates document processing, embedding, and versioned
// storage. It owns lifecycle decisions but contains no database or HTTP SDK
// details.
package indexer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"gorag/internal/document"
	"gorag/internal/document/cleaner"
	"gorag/internal/embedding"
	"gorag/internal/repository"
)

const DefaultFinalizeTimeout = 5 * time.Second

// Loader is the stable filesystem boundary supplied by document/loader.
type Loader interface {
	Scan(context.Context, string) ([]document.Document, error)
	Load(context.Context, string, string) (document.Document, error)
}

// Splitter is the stable structural chunking boundary.
type Splitter interface {
	Split(document.Document) ([]document.Chunk, error)
}

// Embedder is the document-only portion of embedding.Client used here.
type Embedder interface {
	EmbedDocuments(context.Context, []embedding.DocumentInput) ([][]float32, error)
	Model() string
	Dimension() int
}

// Store exposes only atomic repository operations. Embedding is completed
// before InsertVersion is called, so it is never held inside a DB transaction.
type Store interface {
	GetOrCreateDocument(context.Context, repository.DocumentCreate) (repository.Document, bool, error)
	GetDocumentByPath(context.Context, string) (repository.Document, error)
	ListDocuments(context.Context) ([]repository.Document, error)
	InsertVersion(context.Context, int64, string, []repository.VersionChunk) error
	ActivateVersion(context.Context, repository.Activation) error
	MarkDocumentDeleted(context.Context, int64) error
	DeleteInactiveVersions(context.Context, int64) (int64, error)
	StartIndexRun(context.Context, string) (repository.IndexRun, error)
	CompleteIndexRun(context.Context, int64, int, int) error
	FailIndexRun(context.Context, int64, int, int, error) error
}

type Config struct {
	DocsRoot        string
	FinalizeTimeout time.Duration
}

// Result summarizes one recorded lifecycle operation.
type Result struct {
	RunID        int64
	Operation    string
	Documents    int
	Chunks       int
	Added        int
	Updated      int
	Deleted      int
	Skipped      int
	Failed       int
	FailurePaths []string
}

type Indexer struct {
	config   Config
	loader   Loader
	splitter Splitter
	embedder Embedder
	store    Store
	version  func() (string, error)
}

type Option func(*Indexer)

// WithVersionGenerator makes version allocation deterministic in tests.
func WithVersionGenerator(generator func() (string, error)) Option {
	return func(indexer *Indexer) {
		if generator != nil {
			indexer.version = generator
		}
	}
}

func New(config Config, loader Loader, splitter Splitter, embedder Embedder, store Store, options ...Option) (*Indexer, error) {
	if strings.TrimSpace(config.DocsRoot) == "" {
		return nil, errors.New("indexer: docs root is empty")
	}
	if config.FinalizeTimeout == 0 {
		config.FinalizeTimeout = DefaultFinalizeTimeout
	}
	if config.FinalizeTimeout < 0 {
		return nil, errors.New("indexer: finalize timeout must be positive")
	}
	if loader == nil || splitter == nil || embedder == nil || store == nil {
		return nil, errors.New("indexer: loader, splitter, embedder, and store are required")
	}
	if embedder.Model() != embedding.DefaultModel {
		return nil, fmt.Errorf("indexer: embedding model %q, want %q", embedder.Model(), embedding.DefaultModel)
	}
	if embedder.Dimension() != embedding.VectorDimension {
		return nil, fmt.Errorf("indexer: embedding dimension %d, want %d", embedder.Dimension(), embedding.VectorDimension)
	}

	indexer := &Indexer{
		config: config, loader: loader, splitter: splitter, embedder: embedder,
		store: store, version: newVersion,
	}
	for _, option := range options {
		option(indexer)
	}
	return indexer, nil
}

// Sync indexes new or changed files, skips unchanged active versions, and
// marks database documents missing from disk as deleted.
func (i *Indexer) Sync(ctx context.Context) (Result, error) {
	return i.withRun(ctx, "sync", func(ctx context.Context, result *Result) error {
		documents, err := i.loader.Scan(ctx, i.config.DocsRoot)
		if err != nil {
			return fmt.Errorf("scan docs root: %w", err)
		}
		stored, err := i.store.ListDocuments(ctx)
		if err != nil {
			return fmt.Errorf("list stored documents: %w", err)
		}

		onDisk := make(map[string]struct{}, len(documents))
		var failures []error
		for _, doc := range documents {
			if err := ctx.Err(); err != nil {
				return errors.Join(append(failures, err)...)
			}
			onDisk[doc.SourcePath] = struct{}{}
			outcome, chunks, err := i.indexDocument(ctx, doc, false)
			result.Documents++
			result.Chunks += chunks
			if err != nil {
				result.Failed++
				result.FailurePaths = append(result.FailurePaths, doc.SourcePath)
				failures = append(failures, fmt.Errorf("index %q: %w", doc.SourcePath, err))
				continue
			}
			result.record(outcome)
		}

		for _, record := range stored {
			if _, exists := onDisk[record.SourcePath]; exists || record.Status == repository.DocumentStatusDeleted {
				continue
			}
			if err := ctx.Err(); err != nil {
				return errors.Join(append(failures, err)...)
			}
			result.Documents++
			if err := i.store.MarkDocumentDeleted(ctx, record.ID); err != nil {
				result.Failed++
				result.FailurePaths = append(result.FailurePaths, record.SourcePath)
				failures = append(failures, fmt.Errorf("delete missing %q: %w", record.SourcePath, err))
				continue
			}
			result.Deleted++
		}
		return errors.Join(failures...)
	})
}

// IndexFile indexes a target when it is new or changed and otherwise skips it.
func (i *Indexer) IndexFile(ctx context.Context, target string) (Result, error) {
	return i.fileRun(ctx, "add", target, false)
}

// ReindexFile always builds and activates a fresh version of the target.
func (i *Indexer) ReindexFile(ctx context.Context, target string) (Result, error) {
	return i.fileRun(ctx, "reindex", target, true)
}

// ReindexAll rebuilds every valid file currently present on disk.
func (i *Indexer) ReindexAll(ctx context.Context) (Result, error) {
	return i.withRun(ctx, "reindex", func(ctx context.Context, result *Result) error {
		documents, err := i.loader.Scan(ctx, i.config.DocsRoot)
		if err != nil {
			return fmt.Errorf("scan docs root: %w", err)
		}
		var failures []error
		for _, doc := range documents {
			if err := ctx.Err(); err != nil {
				return errors.Join(append(failures, err)...)
			}
			outcome, chunks, err := i.indexDocument(ctx, doc, true)
			result.Documents++
			result.Chunks += chunks
			if err != nil {
				result.Failed++
				result.FailurePaths = append(result.FailurePaths, doc.SourcePath)
				failures = append(failures, fmt.Errorf("reindex %q: %w", doc.SourcePath, err))
				continue
			}
			result.record(outcome)
		}
		return errors.Join(failures...)
	})
}

// DeleteFile immediately removes the target from retrieval by marking it.
func (i *Indexer) DeleteFile(ctx context.Context, target string) (Result, error) {
	return i.withRun(ctx, "delete", func(ctx context.Context, result *Result) error {
		sourcePath, err := normalizeTarget(target)
		if err != nil {
			return err
		}
		record, err := i.store.GetDocumentByPath(ctx, sourcePath)
		result.Documents = 1
		if err != nil {
			result.Failed = 1
			result.FailurePaths = []string{sourcePath}
			return fmt.Errorf("get document %q: %w", sourcePath, err)
		}
		if record.Status == repository.DocumentStatusDeleted {
			result.Skipped = 1
			return nil
		}
		if err := i.store.MarkDocumentDeleted(ctx, record.ID); err != nil {
			result.Failed = 1
			result.FailurePaths = []string{sourcePath}
			return fmt.Errorf("mark document %q deleted: %w", sourcePath, err)
		}
		result.Deleted = 1
		return nil
	})
}

func (i *Indexer) fileRun(ctx context.Context, runType, target string, force bool) (Result, error) {
	return i.withRun(ctx, runType, func(ctx context.Context, result *Result) error {
		sourcePath, err := normalizeTarget(target)
		if err != nil {
			return err
		}
		doc, err := i.loader.Load(ctx, i.config.DocsRoot, sourcePath)
		result.Documents = 1
		if err != nil {
			result.Failed = 1
			result.FailurePaths = []string{sourcePath}
			return fmt.Errorf("load %q: %w", sourcePath, err)
		}
		outcome, chunks, err := i.indexDocument(ctx, doc, force)
		result.Chunks = chunks
		if err != nil {
			result.Failed = 1
			result.FailurePaths = []string{doc.SourcePath}
			return fmt.Errorf("index %q: %w", doc.SourcePath, err)
		}
		result.record(outcome)
		return nil
	})
}

type outcome int

const (
	outcomeAdded outcome = iota
	outcomeUpdated
	outcomeSkipped
)

func (r *Result) record(value outcome) {
	switch value {
	case outcomeAdded:
		r.Added++
	case outcomeUpdated:
		r.Updated++
	case outcomeSkipped:
		r.Skipped++
	}
}

func (i *Indexer) indexDocument(ctx context.Context, doc document.Document, force bool) (outcome, int, error) {
	if err := ctx.Err(); err != nil {
		return outcomeSkipped, 0, err
	}
	record, created, err := i.store.GetOrCreateDocument(ctx, repository.DocumentCreate{
		SourcePath: doc.SourcePath, Title: doc.Title, ContentHash: doc.ContentHash,
	})
	if err != nil {
		return outcomeSkipped, 0, fmt.Errorf("get or create document: %w", err)
	}
	if !force && !created && record.Status == repository.DocumentStatusActive &&
		record.CurrentVersion != nil && record.ContentHash == doc.ContentHash {
		return outcomeSkipped, 0, nil
	}

	version, err := i.version()
	if err != nil {
		return outcomeSkipped, 0, fmt.Errorf("allocate document version: %w", err)
	}
	doc.ID = strconv.FormatInt(record.ID, 10)
	doc.Version = version
	cleaned := cleaner.Clean(doc.Content, cleaner.Options{ParseFrontMatter: true})
	doc.Content = cleaned.Content
	doc.FrontMatter = cleaned.FrontMatter
	doc.LineNumbers = append([]int(nil), cleaned.LineNumbers...)
	chunks, err := i.splitter.Split(doc)
	if err != nil {
		return outcomeSkipped, 0, fmt.Errorf("split document: %w", err)
	}
	if len(chunks) == 0 {
		return outcomeSkipped, 0, errors.New("split document: no chunks produced")
	}

	inputs := make([]embedding.DocumentInput, len(chunks))
	for index, chunk := range chunks {
		inputs[index] = embedding.DocumentInput{SourcePath: doc.SourcePath, ChunkIndex: chunk.Index, Text: chunk.Content}
	}
	vectors, err := i.embedder.EmbedDocuments(ctx, inputs)
	if err != nil {
		return outcomeSkipped, 0, fmt.Errorf("embed document: %w", err)
	}
	if len(vectors) != len(chunks) {
		return outcomeSkipped, 0, fmt.Errorf("embed document: got %d vectors for %d chunks", len(vectors), len(chunks))
	}

	storedChunks := make([]repository.VersionChunk, len(chunks))
	for index := range chunks {
		if err := ctx.Err(); err != nil {
			return outcomeSkipped, 0, err
		}
		if len(vectors[index]) != i.embedder.Dimension() {
			return outcomeSkipped, 0, fmt.Errorf("embed document: chunk %d vector dimension %d, want %d", chunks[index].Index, len(vectors[index]), i.embedder.Dimension())
		}
		chunks[index].DocumentID = strconv.FormatInt(record.ID, 10)
		chunks[index].DocumentVersion = version
		chunks[index].EmbeddingModel = i.embedder.Model()
		chunks[index].EmbeddingDimension = i.embedder.Dimension()
		storedChunks[index] = repository.VersionChunk{Chunk: chunks[index], Embedding: vectors[index]}
	}
	if err := i.store.InsertVersion(ctx, record.ID, version, storedChunks); err != nil {
		return outcomeSkipped, 0, fmt.Errorf("insert version %q: %w", version, err)
	}
	if err := i.store.ActivateVersion(ctx, repository.Activation{
		DocumentID: record.ID, Version: version, ExpectedChunkCount: len(storedChunks),
		Title: doc.Title, ContentHash: doc.ContentHash,
	}); err != nil {
		return outcomeSkipped, 0, fmt.Errorf("activate version %q: %w", version, err)
	}
	if _, err := i.store.DeleteInactiveVersions(ctx, record.ID); err != nil {
		return outcomeSkipped, len(storedChunks), fmt.Errorf("clean inactive versions: %w", err)
	}
	if created || record.CurrentVersion == nil {
		return outcomeAdded, len(storedChunks), nil
	}
	return outcomeUpdated, len(storedChunks), nil
}

func (i *Indexer) withRun(ctx context.Context, runType string, work func(context.Context, *Result) error) (Result, error) {
	result := Result{Operation: runType}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	run, err := i.store.StartIndexRun(ctx, runType)
	if err != nil {
		return result, fmt.Errorf("start %s index run: %w", runType, err)
	}
	result.RunID = run.ID

	workErr := work(ctx, &result)
	if workErr == nil {
		workErr = ctx.Err()
	}
	finalizeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), i.config.FinalizeTimeout)
	defer cancel()
	if workErr != nil {
		if err := i.store.FailIndexRun(finalizeCtx, run.ID, result.Documents, result.Chunks, workErr); err != nil {
			return result, errors.Join(workErr, fmt.Errorf("fail index run %d: %w", run.ID, err))
		}
		return result, workErr
	}
	if err := i.store.CompleteIndexRun(finalizeCtx, run.ID, result.Documents, result.Chunks); err != nil {
		return result, fmt.Errorf("complete index run %d: %w", run.ID, err)
	}
	return result, nil
}

func normalizeTarget(target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", errors.New("indexer: target path is empty")
	}
	if path.IsAbs(strings.ReplaceAll(target, "\\", "/")) || (len(target) >= 2 && target[1] == ':') {
		return "", fmt.Errorf("indexer: target %q must be relative to docs root", target)
	}
	normalized := path.Clean(strings.ReplaceAll(target, "\\", "/"))
	if normalized == "." || normalized == ".." || strings.HasPrefix(normalized, "../") {
		return "", fmt.Errorf("indexer: target %q is outside docs root", target)
	}
	return normalized, nil
}

func newVersion() (string, error) {
	var suffix [12]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(suffix[:]), nil
}
