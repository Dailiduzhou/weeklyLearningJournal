package indexer

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"gorag/internal/document"
	"gorag/internal/embedding"
	"gorag/internal/repository"
)

func TestSyncIsRepeatableAndReindexesChangedDocument(t *testing.T) {
	ctx := context.Background()
	loader := &fakeLoader{documents: []document.Document{testDocument("api/auth.md", "hash-v1")}}
	store := newMemoryStore()
	indexer := newTestIndexer(t, loader, store, nil)

	first, err := indexer.Sync(ctx)
	if err != nil || first.Added != 1 || first.Chunks != 1 {
		t.Fatalf("first Sync() = %#v, error %v", first, err)
	}
	firstVersion := store.byPath["api/auth.md"].CurrentVersion
	if firstVersion == nil || store.insertCalls != 1 || store.runs[len(store.runs)-1].Status != repository.IndexRunStatusCompleted {
		t.Fatalf("first sync state = %#v, insert calls %d", store.byPath["api/auth.md"], store.insertCalls)
	}

	second, err := indexer.Sync(ctx)
	if err != nil || second.Skipped != 1 || second.Chunks != 0 || store.insertCalls != 1 {
		t.Fatalf("second Sync() = %#v, error %v, inserts %d", second, err, store.insertCalls)
	}

	loader.documents[0] = testDocument("api/auth.md", "hash-v2")
	third, err := indexer.Sync(ctx)
	if err != nil || third.Updated != 1 || third.Chunks != 1 {
		t.Fatalf("changed Sync() = %#v, error %v", third, err)
	}
	current := store.byPath["api/auth.md"].CurrentVersion
	if current == nil || *current == *firstVersion || store.byPath["api/auth.md"].ContentHash != "hash-v2" {
		t.Fatalf("changed document was not activated: %#v", store.byPath["api/auth.md"])
	}
}

func TestReindexFailurePreservesActiveVersion(t *testing.T) {
	ctx := context.Background()
	loader := &fakeLoader{documents: []document.Document{testDocument("guide.md", "hash-v1")}}
	store := newMemoryStore()
	indexer := newTestIndexer(t, loader, store, nil)
	if _, err := indexer.Sync(ctx); err != nil {
		t.Fatalf("initial Sync() error = %v", err)
	}
	oldVersion := *store.byPath["guide.md"].CurrentVersion

	loader.documents[0] = testDocument("guide.md", "hash-v2")
	embedder := &fakeEmbedder{failPath: "guide.md"}
	indexer = newTestIndexer(t, loader, store, embedder)
	result, err := indexer.ReindexFile(ctx, "guide.md")
	if err == nil || result.Failed != 1 {
		t.Fatalf("ReindexFile() = %#v, error %v", result, err)
	}
	record := store.byPath["guide.md"]
	if record.CurrentVersion == nil || *record.CurrentVersion != oldVersion || record.ContentHash != "hash-v1" {
		t.Fatalf("failed reindex changed active metadata: %#v", record)
	}
	if store.runs[len(store.runs)-1].Status != repository.IndexRunStatusFailed {
		t.Fatalf("failed reindex run = %#v", store.runs[len(store.runs)-1])
	}
}

func TestStorageFailuresPreserveActiveVersion(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		configure func(*memoryStore)
	}{
		{name: "insert", configure: func(store *memoryStore) { store.insertErr = errors.New("write failed") }},
		{name: "activate", configure: func(store *memoryStore) { store.activateErr = errors.New("activation failed") }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			loader := &fakeLoader{documents: []document.Document{testDocument("guide.md", "hash-v1")}}
			store := newMemoryStore()
			indexer := newTestIndexer(t, loader, store, nil)
			if _, err := indexer.Sync(ctx); err != nil {
				t.Fatalf("initial Sync() error = %v", err)
			}
			oldVersion := *store.byPath["guide.md"].CurrentVersion
			testCase.configure(store)
			loader.documents[0] = testDocument("guide.md", "hash-v2")

			if _, err := indexer.ReindexFile(ctx, "guide.md"); err == nil {
				t.Fatal("ReindexFile() error = nil")
			}
			record := store.byPath["guide.md"]
			if record.CurrentVersion == nil || *record.CurrentVersion != oldVersion || record.ContentHash != "hash-v1" {
				t.Fatalf("failed %s changed active metadata: %#v", testCase.name, record)
			}
		})
	}
}

func TestSyncMarksMissingDocumentDeletedAndReportsPartialFailure(t *testing.T) {
	ctx := context.Background()
	loader := &fakeLoader{documents: []document.Document{
		testDocument("bad.md", "bad-hash"),
		testDocument("good.md", "good-hash"),
	}}
	store := newMemoryStore()
	missing := repository.Document{ID: 99, SourcePath: "removed.md", Title: "removed", ContentHash: "old", Status: repository.DocumentStatusActive}
	store.byPath[missing.SourcePath] = missing
	store.nextID = 100
	embedder := &fakeEmbedder{failPath: "bad.md"}
	indexer := newTestIndexer(t, loader, store, embedder)

	result, err := indexer.Sync(ctx)
	if err == nil || result.Failed != 1 || result.Added != 1 || result.Deleted != 1 {
		t.Fatalf("Sync() = %#v, error %v", result, err)
	}
	if store.byPath["removed.md"].Status != repository.DocumentStatusDeleted {
		t.Fatalf("missing document status = %q", store.byPath["removed.md"].Status)
	}
	if store.byPath["good.md"].CurrentVersion == nil {
		t.Fatal("sync stopped before processing a healthy document")
	}
	if store.runs[len(store.runs)-1].Status != repository.IndexRunStatusFailed {
		t.Fatalf("partial run status = %q", store.runs[len(store.runs)-1].Status)
	}
}

func TestContextCancellationStopsWorkAndFinalizesRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	loader := &fakeLoader{documents: []document.Document{
		testDocument("a.md", "a"), testDocument("b.md", "b"),
	}}
	store := newMemoryStore()
	embedder := &fakeEmbedder{cancel: cancel}
	indexer := newTestIndexer(t, loader, store, embedder)

	result, err := indexer.Sync(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Sync() error = %v, want context.Canceled", err)
	}
	if result.Documents != 1 || embedder.callCount() != 1 {
		t.Fatalf("cancellation processed documents=%d embedding calls=%d", result.Documents, embedder.callCount())
	}
	last := store.runs[len(store.runs)-1]
	if last.Status != repository.IndexRunStatusFailed || store.finalizeSawCancelled {
		t.Fatalf("cancelled run = %#v, finalizeSawCancelled=%v", last, store.finalizeSawCancelled)
	}
}

func TestSyncLimitsDocumentConcurrency(t *testing.T) {
	documents := make([]document.Document, 6)
	for index := range documents {
		documents[index] = testDocument(fmt.Sprintf("doc-%d.md", index), fmt.Sprintf("hash-%d", index))
	}
	release := make(chan struct{})
	entered := make(chan string, len(documents))
	embedder := &fakeEmbedder{entered: entered, release: release}
	indexer := newTestIndexerWithConfig(t, Config{DocsRoot: "docs", DocumentConcurrency: 3}, &fakeLoader{documents: documents}, newMemoryStore(), embedder)

	type syncResult struct {
		result Result
		err    error
	}
	done := make(chan syncResult, 1)
	go func() {
		result, err := indexer.Sync(context.Background())
		done <- syncResult{result: result, err: err}
	}()
	for range 3 {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("three document workers did not start")
		}
	}
	select {
	case path := <-entered:
		t.Fatalf("document %q exceeded configured concurrency before a slot was released", path)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	completed := <-done
	if completed.err != nil || completed.result.Added != len(documents) {
		t.Fatalf("Sync() = %#v, error %v", completed.result, completed.err)
	}
	if embedder.maximumActive() != 3 {
		t.Fatalf("maximum active documents = %d, want 3", embedder.maximumActive())
	}
}

func TestSyncConcurrentFailuresRemainOrderedAndIsolated(t *testing.T) {
	loader := &fakeLoader{documents: []document.Document{
		testDocument("a-bad.md", "a"),
		testDocument("b-good.md", "b"),
		testDocument("c-bad.md", "c"),
	}}
	store := newMemoryStore()
	embedder := &fakeEmbedder{failPaths: map[string]bool{"a-bad.md": true, "c-bad.md": true}}
	indexer := newTestIndexerWithConfig(t, Config{DocsRoot: "docs", DocumentConcurrency: 3}, loader, store, embedder)

	result, err := indexer.Sync(context.Background())
	if err == nil || result.Added != 1 || result.Failed != 2 {
		t.Fatalf("Sync() = %#v, error %v", result, err)
	}
	if len(result.FailurePaths) != 2 || result.FailurePaths[0] != "a-bad.md" || result.FailurePaths[1] != "c-bad.md" {
		t.Fatalf("FailurePaths = %q, want stable input order", result.FailurePaths)
	}
	if strings.Index(err.Error(), "a-bad.md") > strings.Index(err.Error(), "c-bad.md") {
		t.Fatalf("joined errors are not in input order: %v", err)
	}
	if store.byPath["b-good.md"].CurrentVersion == nil {
		t.Fatal("healthy document was not committed")
	}
}

func TestSyncCancellationStopsDispatchingDocuments(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	documents := make([]document.Document, 5)
	for index := range documents {
		documents[index] = testDocument(fmt.Sprintf("doc-%d.md", index), fmt.Sprintf("hash-%d", index))
	}
	entered := make(chan string, len(documents))
	embedder := &fakeEmbedder{entered: entered, release: make(chan struct{})}
	indexer := newTestIndexerWithConfig(t, Config{DocsRoot: "docs", DocumentConcurrency: 2}, &fakeLoader{documents: documents}, newMemoryStore(), embedder)

	type syncResult struct {
		result Result
		err    error
	}
	done := make(chan syncResult, 1)
	go func() {
		result, err := indexer.Sync(ctx)
		done <- syncResult{result: result, err: err}
	}()
	for range 2 {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("two document workers did not start")
		}
	}
	cancel()
	completed := <-done
	if !errors.Is(completed.err, context.Canceled) {
		t.Fatalf("Sync() error = %v, want context.Canceled", completed.err)
	}
	if embedder.callCount() != 2 || completed.result.Documents != 2 {
		t.Fatalf("cancellation processed documents=%d embedding calls=%d, want 2 in-flight documents only", completed.result.Documents, embedder.callCount())
	}
}

func TestAllLifecycleOperations(t *testing.T) {
	ctx := context.Background()
	loader := &fakeLoader{documents: []document.Document{testDocument("one.md", "h1")}}
	store := newMemoryStore()
	indexer := newTestIndexer(t, loader, store, nil)

	if result, err := indexer.IndexFile(ctx, "one.md"); err != nil || result.Added != 1 || result.Operation != "add" {
		t.Fatalf("IndexFile() = %#v, error %v", result, err)
	}
	if result, err := indexer.ReindexFile(ctx, "one.md"); err != nil || result.Updated != 1 || result.Operation != "reindex" {
		t.Fatalf("ReindexFile() = %#v, error %v", result, err)
	}
	if result, err := indexer.ReindexAll(ctx); err != nil || result.Updated != 1 || result.Operation != "reindex" {
		t.Fatalf("ReindexAll() = %#v, error %v", result, err)
	}
	if result, err := indexer.DeleteFile(ctx, "one.md"); err != nil || result.Deleted != 1 || result.Operation != "delete" {
		t.Fatalf("DeleteFile() = %#v, error %v", result, err)
	}
	if result, err := indexer.Sync(ctx); err != nil || result.Updated != 1 || result.Operation != "sync" {
		t.Fatalf("Sync() after delete = %#v, error %v", result, err)
	}
}

func newTestIndexer(t *testing.T, loader *fakeLoader, store *memoryStore, embedder *fakeEmbedder) *Indexer {
	return newTestIndexerWithConfig(t, Config{DocsRoot: "docs"}, loader, store, embedder)
}

func newTestIndexerWithConfig(t *testing.T, config Config, loader *fakeLoader, store *memoryStore, embedder *fakeEmbedder) *Indexer {
	t.Helper()
	if embedder == nil {
		embedder = &fakeEmbedder{}
	}
	sequence := 0
	indexer, err := New(config, loader, fakeSplitter{}, embedder, store,
		WithVersionGenerator(func() (string, error) {
			sequence++
			return fmt.Sprintf("version-%d", sequence), nil
		}))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return indexer
}

func testDocument(path, hash string) document.Document {
	return document.Document{
		SourcePath: path, Title: path, Kind: document.KindMarkdown,
		Content: "# Title\n\nBody", ContentHash: hash,
	}
}

type fakeLoader struct {
	documents []document.Document
	scanErr   error
	loadErr   error
}

func (l *fakeLoader) Scan(ctx context.Context, _ string) ([]document.Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]document.Document(nil), l.documents...), l.scanErr
}

func (l *fakeLoader) Load(ctx context.Context, _ string, target string) (document.Document, error) {
	if err := ctx.Err(); err != nil {
		return document.Document{}, err
	}
	if l.loadErr != nil {
		return document.Document{}, l.loadErr
	}
	for _, doc := range l.documents {
		if doc.SourcePath == target {
			return doc, nil
		}
	}
	return document.Document{}, errors.New("not found")
}

type fakeSplitter struct{}

func (fakeSplitter) Split(doc document.Document) ([]document.Chunk, error) {
	return []document.Chunk{{
		DocumentID: doc.ID, SourcePath: doc.SourcePath, DocumentTitle: doc.Title,
		Index: 0, Content: doc.Content, StartLine: 1, EndLine: 3,
		ContentHash: doc.ContentHash, DocumentVersion: doc.Version,
	}}, nil
}

type fakeEmbedder struct {
	mu        sync.Mutex
	failPath  string
	failPaths map[string]bool
	cancel    context.CancelFunc
	calls     int
	active    int
	maximum   int
	entered   chan<- string
	release   <-chan struct{}
}

func (e *fakeEmbedder) EmbedDocuments(ctx context.Context, inputs []embedding.DocumentInput) ([][]float32, error) {
	e.mu.Lock()
	e.calls++
	e.active++
	if e.active > e.maximum {
		e.maximum = e.active
	}
	path := ""
	if len(inputs) > 0 {
		path = inputs[0].SourcePath
	}
	fail := path == e.failPath || e.failPaths[path]
	cancel := e.cancel
	entered := e.entered
	release := e.release
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		e.active--
		e.mu.Unlock()
	}()

	if entered != nil {
		select {
		case entered <- path:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if release != nil {
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if fail {
		return nil, errors.New("embedding unavailable")
	}
	if cancel != nil {
		cancel()
		return nil, ctx.Err()
	}
	vectors := make([][]float32, len(inputs))
	for index := range vectors {
		vectors[index] = make([]float32, embedding.VectorDimension)
	}
	return vectors, nil
}

func (e *fakeEmbedder) callCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.calls
}

func (e *fakeEmbedder) maximumActive() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.maximum
}

func (*fakeEmbedder) Model() string  { return embedding.DefaultModel }
func (*fakeEmbedder) Dimension() int { return embedding.VectorDimension }

type memoryStore struct {
	mu                   sync.Mutex
	byPath               map[string]repository.Document
	chunks               map[int64]map[string][]repository.VersionChunk
	runs                 []repository.IndexRun
	nextID               int64
	insertCalls          int
	insertErr            error
	activateErr          error
	finalizeSawCancelled bool
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		byPath: make(map[string]repository.Document), chunks: make(map[int64]map[string][]repository.VersionChunk), nextID: 1,
	}
}

func (s *memoryStore) GetOrCreateDocument(ctx context.Context, input repository.DocumentCreate) (repository.Document, bool, error) {
	if err := ctx.Err(); err != nil {
		return repository.Document{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if record, exists := s.byPath[input.SourcePath]; exists {
		return record, false, nil
	}
	record := repository.Document{ID: s.nextID, SourcePath: input.SourcePath, Title: input.Title, ContentHash: input.ContentHash, Status: repository.DocumentStatusActive}
	s.nextID++
	s.byPath[input.SourcePath] = record
	return record, true, nil
}

func (s *memoryStore) GetDocumentByPath(ctx context.Context, path string) (repository.Document, error) {
	if err := ctx.Err(); err != nil {
		return repository.Document{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, exists := s.byPath[path]
	if !exists {
		return repository.Document{}, errors.New("not found")
	}
	return record, nil
}

func (s *memoryStore) ListDocuments(ctx context.Context) ([]repository.Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	documents := make([]repository.Document, 0, len(s.byPath))
	for _, record := range s.byPath {
		documents = append(documents, record)
	}
	sort.Slice(documents, func(a, b int) bool { return documents[a].SourcePath < documents[b].SourcePath })
	return documents, nil
}

func (s *memoryStore) InsertVersion(ctx context.Context, documentID int64, version string, chunks []repository.VersionChunk) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.insertErr != nil {
		return s.insertErr
	}
	s.insertCalls++
	if s.chunks[documentID] == nil {
		s.chunks[documentID] = make(map[string][]repository.VersionChunk)
	}
	s.chunks[documentID][version] = chunks
	return nil
}

func (s *memoryStore) ActivateVersion(ctx context.Context, activation repository.Activation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activateErr != nil {
		return s.activateErr
	}
	if len(s.chunks[activation.DocumentID][activation.Version]) != activation.ExpectedChunkCount {
		return repository.ErrIncompleteVersion
	}
	for path, record := range s.byPath {
		if record.ID == activation.DocumentID {
			version := activation.Version
			record.CurrentVersion = &version
			record.Title = activation.Title
			record.ContentHash = activation.ContentHash
			record.Status = repository.DocumentStatusActive
			s.byPath[path] = record
			return nil
		}
	}
	return errors.New("not found")
}

func (s *memoryStore) MarkDocumentDeleted(ctx context.Context, documentID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for path, record := range s.byPath {
		if record.ID == documentID {
			record.Status = repository.DocumentStatusDeleted
			s.byPath[path] = record
			return nil
		}
	}
	return errors.New("not found")
}

func (s *memoryStore) DeleteInactiveVersions(context.Context, int64) (int64, error) { return 0, nil }

func (s *memoryStore) StartIndexRun(ctx context.Context, runType string) (repository.IndexRun, error) {
	if err := ctx.Err(); err != nil {
		return repository.IndexRun{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	run := repository.IndexRun{ID: int64(len(s.runs) + 1), RunType: runType, Status: repository.IndexRunStatusRunning}
	s.runs = append(s.runs, run)
	return run, nil
}

func (s *memoryStore) CompleteIndexRun(ctx context.Context, runID int64, documents, chunks int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.finalizeSawCancelled = s.finalizeSawCancelled || ctx.Err() != nil
	if err := ctx.Err(); err != nil {
		return err
	}
	s.runs[runID-1].Status = repository.IndexRunStatusCompleted
	s.runs[runID-1].DocumentCount = documents
	s.runs[runID-1].ChunkCount = chunks
	return nil
}

func (s *memoryStore) FailIndexRun(ctx context.Context, runID int64, documents, chunks int, runErr error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.finalizeSawCancelled = s.finalizeSawCancelled || ctx.Err() != nil
	if err := ctx.Err(); err != nil {
		return err
	}
	s.runs[runID-1].Status = repository.IndexRunStatusFailed
	s.runs[runID-1].DocumentCount = documents
	s.runs[runID-1].ChunkCount = chunks
	message := runErr.Error()
	s.runs[runID-1].ErrorMessage = &message
	return nil
}
