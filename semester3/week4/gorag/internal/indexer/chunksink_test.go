package indexer

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"gorag/internal/document"
)

type fakeChunkSink struct {
	mu             sync.Mutex
	indexCalls     []sinkCall
	deleteCalls    []string
	failIndexCall  int
	failDeleteCall int
}

type sinkCall struct {
	documentID string
	version    string
	chunkCount int
}

func (s *fakeChunkSink) IndexChunks(ctx context.Context, documentID, version string, chunks []document.Chunk) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.indexCalls)+1 == s.failIndexCall {
		return errors.New("sink index failed")
	}
	s.indexCalls = append(s.indexCalls, sinkCall{documentID: documentID, version: version, chunkCount: len(chunks)})
	return nil
}

func (s *fakeChunkSink) DeleteDocument(ctx context.Context, documentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.deleteCalls)+1 == s.failDeleteCall {
		return errors.New("sink delete failed")
	}
	s.deleteCalls = append(s.deleteCalls, documentID)
	return nil
}

func TestChunkSinkMirrorsActivatedVersions(t *testing.T) {
	ctx := context.Background()
	loader := &fakeLoader{documents: []document.Document{testDocument("api/auth.md", "hash-v1")}}
	store := newMemoryStore()
	sink := &fakeChunkSink{}
	sequence := 0
	indexer, err := New(Config{DocsRoot: "docs"}, loader, fakeSplitter{}, &fakeEmbedder{}, store,
		WithVersionGenerator(func() (string, error) {
			sequence++
			return "version-" + string(rune('0'+sequence)), nil
		}), WithChunkSink(sink))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if _, err := indexer.Sync(ctx); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if len(sink.indexCalls) != 1 {
		t.Fatalf("sink index calls = %d, want 1", len(sink.indexCalls))
	}
	first := sink.indexCalls[0]
	if first.documentID != "1" || first.version != "version-1" || first.chunkCount != 1 {
		t.Fatalf("first sink call = %#v, want document 1, version-1, one chunk", first)
	}

	// Unchanged documents must not be re-mirrored.
	if _, err := indexer.Sync(ctx); err != nil {
		t.Fatalf("second Sync() error = %v", err)
	}
	if len(sink.indexCalls) != 1 {
		t.Fatalf("sink index calls after skip = %d, want 1", len(sink.indexCalls))
	}

	// Reindex replaces the version in the sink.
	if _, err := indexer.ReindexFile(ctx, "api/auth.md"); err != nil {
		t.Fatalf("ReindexFile() error = %v", err)
	}
	if len(sink.indexCalls) != 2 || sink.indexCalls[1].version != "version-2" {
		t.Fatalf("sink index calls after reindex = %#v, want version-2", sink.indexCalls)
	}

	if _, err := indexer.DeleteFile(ctx, "api/auth.md"); err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}
	if len(sink.deleteCalls) != 1 || sink.deleteCalls[0] != "1" {
		t.Fatalf("sink delete calls = %#v, want [1]", sink.deleteCalls)
	}
}

func TestChunkSinkFailureFailsOperation(t *testing.T) {
	ctx := context.Background()
	loader := &fakeLoader{documents: []document.Document{testDocument("api/auth.md", "hash-v1")}}
	store := newMemoryStore()
	sink := &fakeChunkSink{failIndexCall: 1}
	sequence := 0
	indexer, err := New(Config{DocsRoot: "docs"}, loader, fakeSplitter{}, &fakeEmbedder{}, store,
		WithVersionGenerator(func() (string, error) {
			sequence++
			return "version-1", nil
		}), WithChunkSink(sink))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := indexer.Sync(ctx)
	if err == nil || !strings.Contains(err.Error(), "chunk sink") {
		t.Fatalf("Sync() error = %v, want chunk sink failure", err)
	}
	if result.Failed != 1 || len(result.FailurePaths) != 1 {
		t.Fatalf("result = %#v, want one failure", result)
	}
}

func TestChunkSinkDeleteFailureFailsOperation(t *testing.T) {
	ctx := context.Background()
	loader := &fakeLoader{documents: []document.Document{testDocument("api/auth.md", "hash-v1")}}
	store := newMemoryStore()
	sink := &fakeChunkSink{failDeleteCall: 1}
	indexer, err := New(Config{DocsRoot: "docs"}, loader, fakeSplitter{}, &fakeEmbedder{}, store, WithChunkSink(sink))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := indexer.Sync(ctx); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	sink.mu.Lock()
	sink.failDeleteCall = 1
	sink.mu.Unlock()
	result, err := indexer.DeleteFile(ctx, "api/auth.md")
	if err == nil || !strings.Contains(err.Error(), "chunk sink") {
		t.Fatalf("DeleteFile() error = %v, want chunk sink failure", err)
	}
	if result.Deleted != 0 || result.Failed != 1 {
		t.Fatalf("result = %#v, want failed delete", result)
	}
}
