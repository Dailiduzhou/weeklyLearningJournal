// Package bm25 implements a local lexical (BM25) chunk index backed by
// bluge. It runs in parallel with the pgvector store: the indexing pipeline
// mirrors every activated chunk version into this index and the query side
// searches it as an independent retriever.
package bm25

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/index"
	"github.com/blugelabs/bluge/search"

	"gorag/internal/document"
)

// Field names used inside the bluge index.
const (
	FieldContent     = "content"
	FieldTitle       = "document_title"
	FieldDocumentID  = "document_id"
	FieldSourcePath  = "source_path"
	FieldVersion     = "document_version"
	FieldContentHash = "content_hash"
	FieldHeadingPath = "heading_path_json"
	FieldChunkIndex  = "chunk_index"
	FieldStartLine   = "start_line"
	FieldEndLine     = "end_line"
)

// TitleBoost biases title matches above ordinary content matches.
const TitleBoost = 2.0

// SearchResult mirrors repository.SearchResult so the retriever layer can
// adapt either backend with the same code shape. Score is the raw BM25 score,
// which is unbounded and not comparable to cosine similarity.
type SearchResult struct {
	DocumentID int64
	document.Chunk
	Score float64
}

var (
	// ErrInvalidConfig identifies construction or call-time options that
	// cannot produce a bounded BM25 search.
	ErrInvalidConfig = errors.New("bm25: invalid configuration")
	// ErrSearch identifies index query failures.
	ErrSearch = errors.New("bm25: index search failed")
)

// Index owns the bluge writer for one on-disk index directory. It is safe for
// use by a single indexing process; bluge serializes writes internally.
type Index struct {
	writer *bluge.Writer
}

// Open opens (creating if necessary) the index directory at path.
func Open(path string) (*Index, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("%w: index path is empty", ErrInvalidConfig)
	}
	writer, err := bluge.OpenWriter(bluge.DefaultConfig(path))
	if err != nil {
		return nil, fmt.Errorf("bm25: open index at %q: %w", path, err)
	}
	return &Index{writer: writer}, nil
}

// Close flushes and releases the underlying writer.
func (i *Index) Close() error {
	if i == nil || i.writer == nil {
		return nil
	}
	if err := i.writer.Close(); err != nil {
		return fmt.Errorf("bm25: close index: %w", err)
	}
	return nil
}

// IndexChunks atomically replaces all stored chunks of one document with the
// supplied version's chunks. Chunks must already carry their document ID and
// version, matching the repository contract.
func (i *Index) IndexChunks(ctx context.Context, docID, version string, chunks []document.Chunk) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if i == nil || i.writer == nil {
		return fmt.Errorf("%w: index is closed", ErrInvalidConfig)
	}
	if strings.TrimSpace(docID) == "" || strings.TrimSpace(version) == "" {
		return fmt.Errorf("%w: document ID and version are required", ErrInvalidConfig)
	}
	if len(chunks) == 0 {
		return fmt.Errorf("%w: no chunks supplied for document %q", ErrInvalidConfig, docID)
	}

	batch := bluge.NewBatch()
	// Old versions of the same document must not survive a new activation.
	if err := appendDocumentDeletes(ctx, i.writer, batch, docID); err != nil {
		return err
	}
	for _, chunk := range chunks {
		if chunk.DocumentID != docID || chunk.DocumentVersion != version {
			return fmt.Errorf("%w: chunk %d has ID/version %q/%q, want %q/%q",
				ErrInvalidConfig, chunk.Index, chunk.DocumentID, chunk.DocumentVersion, docID, version)
		}
		batch.Insert(chunkDocument(chunk))
	}
	if err := i.writer.Batch(batch); err != nil {
		return fmt.Errorf("bm25: index %d chunks for document %q: %w", len(chunks), docID, err)
	}
	return nil
}

// DeleteDocument removes every stored chunk of the document.
func (i *Index) DeleteDocument(ctx context.Context, docID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if i == nil || i.writer == nil {
		return fmt.Errorf("%w: index is closed", ErrInvalidConfig)
	}
	if strings.TrimSpace(docID) == "" {
		return fmt.Errorf("%w: document ID is required", ErrInvalidConfig)
	}
	batch := bluge.NewBatch()
	if err := appendDocumentDeletes(ctx, i.writer, batch, docID); err != nil {
		return err
	}
	if err := i.writer.Batch(batch); err != nil {
		return fmt.Errorf("bm25: delete document %q: %w", docID, err)
	}
	return nil
}

// Search returns the topK lexical matches for the query, ranked by BM25
// score. Both content and title fields are searched; title hits are boosted.
func (i *Index) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if i == nil || i.writer == nil {
		return nil, fmt.Errorf("%w: index is closed", ErrInvalidConfig)
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("%w: query is empty", ErrInvalidConfig)
	}
	if topK <= 0 || topK > 100 {
		return nil, fmt.Errorf("%w: topK must be between 1 and 100, got %d", ErrInvalidConfig, topK)
	}

	contentQuery := bluge.NewMatchQuery(query).SetField(FieldContent)
	titleQuery := bluge.NewMatchQuery(query).SetField(FieldTitle).SetBoost(TitleBoost)
	booleanQuery := bluge.NewBooleanQuery().AddShould(contentQuery, titleQuery)

	reader, err := i.writer.Reader()
	if err != nil {
		return nil, fmt.Errorf("%w: open index reader: %w", ErrSearch, err)
	}
	defer reader.Close()

	request := bluge.NewTopNSearch(topK, booleanQuery).WithStandardAggregations()
	matches, err := reader.Search(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("%w: execute search: %w", ErrSearch, err)
	}

	results := make([]SearchResult, 0, topK)
	for {
		match, err := matches.Next()
		if err != nil {
			return nil, fmt.Errorf("%w: iterate matches: %w", ErrSearch, err)
		}
		if match == nil {
			break
		}
		result, err := resultFromMatch(match)
		if err != nil {
			return nil, err
		}
		result.Score = match.Score
		results = append(results, result)
	}
	return results, nil
}

func chunkIdentifier(docID, version string, chunkIndex int) string {
	return docID + ":" + version + ":" + strconv.Itoa(chunkIndex)
}

func chunkDocument(chunk document.Chunk) *bluge.Document {
	return bluge.NewDocument(chunkIdentifier(chunk.DocumentID, chunk.DocumentVersion, chunk.Index)).
		AddField(bluge.NewTextField(FieldContent, chunk.Content).StoreValue()).
		AddField(bluge.NewKeywordField(FieldTitle, chunk.DocumentTitle).StoreValue()).
		AddField(bluge.NewKeywordField(FieldDocumentID, chunk.DocumentID).StoreValue()).
		AddField(bluge.NewKeywordField(FieldSourcePath, chunk.SourcePath).StoreValue()).
		AddField(bluge.NewKeywordField(FieldVersion, chunk.DocumentVersion).StoreValue()).
		AddField(bluge.NewKeywordField(FieldContentHash, chunk.ContentHash).StoreValue()).
		AddField(bluge.NewKeywordField(FieldHeadingPath, encodeHeadingPath(chunk.HeadingPath)).StoreValue()).
		AddField(bluge.NewNumericField(FieldChunkIndex, float64(chunk.Index)).StoreValue()).
		AddField(bluge.NewNumericField(FieldStartLine, float64(chunk.StartLine)).StoreValue()).
		AddField(bluge.NewNumericField(FieldEndLine, float64(chunk.EndLine)).StoreValue())
}

func encodeHeadingPath(headingPath []string) string {
	if len(headingPath) == 0 {
		headingPath = []string{}
	}
	encoded, err := json.Marshal(headingPath)
	if err != nil {
		// json.Marshal on []string cannot fail; fall back defensively.
		return "[]"
	}
	return string(encoded)
}

// appendDocumentDeletes adds stored-IDs deletion entries for every chunk of
// the document currently in the index.
func appendDocumentDeletes(ctx context.Context, writer *bluge.Writer, batch *index.Batch, docID string) error {
	termQuery := bluge.NewTermQuery(docID).SetField(FieldDocumentID)
	request := bluge.NewAllMatches(termQuery)
	reader, err := writer.Reader()
	if err != nil {
		return fmt.Errorf("%w: open index reader: %w", ErrSearch, err)
	}
	defer reader.Close()
	matches, err := reader.Search(ctx, request)
	if err != nil {
		return fmt.Errorf("%w: find stored chunks of document %q: %w", ErrSearch, docID, err)
	}
	for {
		match, err := matches.Next()
		if err != nil {
			return fmt.Errorf("%w: iterate stored chunks of document %q: %w", ErrSearch, docID, err)
		}
		if match == nil {
			return nil
		}
		id, ok := storedValue(match, "_id")
		if !ok {
			return fmt.Errorf("%w: stored chunk of document %q has no ID", ErrSearch, docID)
		}
		batch.Delete(bluge.Identifier(id))
	}
}

func resultFromMatch(match *search.DocumentMatch) (SearchResult, error) {
	var result SearchResult
	var headingEncoded string
	var chunkIndex, startLine, endLine float64

	fields := map[string]string{}
	if err := match.VisitStoredFields(func(field string, value []byte) bool {
		fields[field] = string(value)
		return true
	}); err != nil {
		return SearchResult{}, fmt.Errorf("%w: visit stored fields: %w", ErrSearch, err)
	}

	var missing []string
	for _, field := range []string{FieldDocumentID, FieldSourcePath, FieldTitle, FieldVersion, FieldContentHash} {
		if strings.TrimSpace(fields[field]) == "" {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		return SearchResult{}, fmt.Errorf("%w: match is missing stored fields %s", ErrSearch, strings.Join(missing, ", "))
	}

	documentID, err := strconv.ParseInt(fields[FieldDocumentID], 10, 64)
	if err != nil {
		return SearchResult{}, fmt.Errorf("%w: document ID %q is not an integer", ErrSearch, fields[FieldDocumentID])
	}
	if headingEncoded = fields[FieldHeadingPath]; headingEncoded == "" {
		headingEncoded = "[]"
	}
	for field, target := range map[string]*float64{
		FieldChunkIndex: &chunkIndex, FieldStartLine: &startLine, FieldEndLine: &endLine,
	} {
		value, ok := fields[field]
		if !ok {
			return SearchResult{}, fmt.Errorf("%w: match is missing stored field %q", ErrSearch, field)
		}
		decoded, err := bluge.DecodeNumericFloat64([]byte(value))
		if err != nil {
			return SearchResult{}, fmt.Errorf("%w: decode stored field %q: %w", ErrSearch, field, err)
		}
		*target = decoded
	}

	var headingPath []string
	if err := json.Unmarshal([]byte(headingEncoded), &headingPath); err != nil {
		return SearchResult{}, fmt.Errorf("%w: decode stored field %q: %w", ErrSearch, FieldHeadingPath, err)
	}
	if headingPath == nil {
		headingPath = []string{}
	}

	result.DocumentID = documentID
	result.Chunk = document.Chunk{
		DocumentID:      fields[FieldDocumentID],
		SourcePath:      fields[FieldSourcePath],
		DocumentTitle:   fields[FieldTitle],
		HeadingPath:     headingPath,
		Index:           int(chunkIndex),
		Content:         fields[FieldContent],
		StartLine:       int(startLine),
		EndLine:         int(endLine),
		ContentHash:     fields[FieldContentHash],
		DocumentVersion: fields[FieldVersion],
	}
	return result, nil
}

func storedValue(match *search.DocumentMatch, field string) (string, bool) {
	var value string
	found := false
	_ = match.VisitStoredFields(func(name string, raw []byte) bool {
		if name == field && !found {
			value = string(raw)
			found = true
		}
		return true
	})
	return value, found
}
