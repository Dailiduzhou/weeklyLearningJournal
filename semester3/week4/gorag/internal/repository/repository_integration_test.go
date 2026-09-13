package repository

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"gorag/internal/document"
	"gorag/internal/embedding"
)

func TestPgvectorRepositoryIntegration(t *testing.T) {
	if os.Getenv("GORAG_INTEGRATION") != "1" {
		t.Skip("set GORAG_INTEGRATION=1 to run the pgvector container test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, dsn := startPgvector(t, ctx)
	defer func() {
		terminateCtx, terminateCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer terminateCancel()
		if err := container.Terminate(terminateCtx); err != nil {
			t.Errorf("terminate pgvector container: %v", err)
		}
	}()
	applyMigration(t, ctx, dsn)

	repository, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer repository.Close()

	doc, created, err := repository.GetOrCreateDocument(ctx, DocumentCreate{
		SourcePath: "api/auth.md", Title: "Authentication", ContentHash: "file-v1",
	})
	if err != nil || !created {
		t.Fatalf("GetOrCreateDocument() = created %v, error %v", created, err)
	}
	same, created, err := repository.GetOrCreateDocument(ctx, DocumentCreate{
		SourcePath: "api/auth.md", Title: "Other", ContentHash: "other",
	})
	if err != nil || created || same.ID != doc.ID {
		t.Fatalf("duplicate GetOrCreateDocument() = id %d created %v error %v", same.ID, created, err)
	}
	listed, err := repository.ListDocuments(ctx)
	if err != nil || len(listed) != 1 || listed[0].ID != doc.ID {
		t.Fatalf("ListDocuments() = %#v, error %v", listed, err)
	}
	cancelledCtx, cancelList := context.WithCancel(ctx)
	cancelList()
	if _, err := repository.ListDocuments(cancelledCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListDocuments(cancelled) error = %v, want context.Canceled", err)
	}

	v1 := []VersionChunk{testStoredChunk("v1", 0, "first", unitVector(0))}
	v1[0].HeadingPath = nil // Top-level prose is stored as an empty, non-NULL path.
	if err := repository.InsertVersion(ctx, doc.ID, "v1", v1); err != nil {
		t.Fatalf("InsertVersion(v1) error = %v", err)
	}
	if err := repository.InsertVersion(ctx, doc.ID, "v1", v1); err == nil {
		t.Fatal("duplicate chunk index was accepted")
	}
	badParent := Activation{
		DocumentID: doc.ID, Version: "v1", ExpectedChunkCount: 1,
		Title: "Authentication", ContentHash: "file-v1",
		Parent: ActivationParent{Content: "first parent", StartLine: 1, EndLine: 0},
	}
	if err := repository.ActivateVersion(ctx, badParent); err == nil {
		t.Fatal("activation with an invalid parent line range was accepted")
	}
	if err := repository.ActivateVersion(ctx, Activation{
		DocumentID: doc.ID, Version: "v1", ExpectedChunkCount: 1,
		Title: "Authentication", ContentHash: "file-v1",
		Parent: ActivationParent{Content: "first parent", StartLine: 1, EndLine: 12},
		Metadata: document.DocumentMetadata{
			Scalars: map[string]string{
				"category": "ccnubox", "type": "optimization", "topic": "crypto",
				"module": "bff", "status": "done",
			},
			Lists: map[string][]string{"tags": {"ccnubox", "ccnubox/bff", "ccnubox/crypto"}},
		},
	}); err != nil {
		t.Fatalf("ActivateVersion(v1) error = %v", err)
	}

	results, err := repository.Search(ctx, unitVector(0), 5, document.MetadataFilter{})
	if err != nil || len(results) != 1 || results[0].DocumentVersion != "v1" {
		t.Fatalf("Search(v1) = %#v, error %v", results, err)
	}

	// Activating v1 also stored the whole-document parent text.
	parents, err := repository.GetParentDocuments(ctx, []int64{doc.ID})
	if err != nil || len(parents) != 1 {
		t.Fatalf("GetParentDocuments(v1) = %#v, error %v", parents, err)
	}
	if parents[0].DocumentID != doc.ID || parents[0].SourcePath != "api/auth.md" ||
		parents[0].Title != "Authentication" || parents[0].Version != "v1" ||
		parents[0].Content != "first parent" || parents[0].StartLine != 1 || parents[0].EndLine != 12 {
		t.Fatalf("GetParentDocuments(v1) = %#v, want stored parent text and source range", parents[0])
	}
	if parents, err := repository.GetParentDocuments(ctx, []int64{doc.ID + 999}); err != nil || len(parents) != 0 {
		t.Fatalf("GetParentDocuments(unknown id) = %#v, error %v, want empty", parents, err)
	}

	// A second document with different front matter proves filters select
	// between documents, not just narrow a single-document result.
	guide, _, err := repository.GetOrCreateDocument(ctx, DocumentCreate{
		SourcePath: "guide/bff.md", Title: "BFF Guide", ContentHash: "guide-v1",
	})
	if err != nil {
		t.Fatalf("GetOrCreateDocument(guide) error = %v", err)
	}
	guideChunks := []VersionChunk{testStoredChunk("v1", 0, "guide content", unitVector(1))}
	if err := repository.InsertVersion(ctx, guide.ID, "v1", guideChunks); err != nil {
		t.Fatalf("InsertVersion(guide) error = %v", err)
	}
	if err := repository.ActivateVersion(ctx, Activation{
		DocumentID: guide.ID, Version: "v1", ExpectedChunkCount: 1,
		Title: "BFF Guide", ContentHash: "guide-v1",
		Parent: ActivationParent{Content: "guide parent", StartLine: 1, EndLine: 9},
		Metadata: document.DocumentMetadata{
			Scalars: map[string]string{"category": "notes", "module": "cli"},
			Lists:   map[string][]string{"tags": {"notes", "notes/cli"}},
		},
	}); err != nil {
		t.Fatalf("ActivateVersion(guide) error = %v", err)
	}

	// Scalar filters keep only exactly matching documents.
	results, err = repository.Search(ctx, unitVector(0), 5, document.MetadataFilter{
		Metadata: map[string]string{"category": "ccnubox"},
	})
	if err != nil || len(results) != 1 || results[0].DocumentID != doc.ID {
		t.Fatalf("Search(category=ccnubox) = %#v, error %v, want only the first document", results, err)
	}
	results, err = repository.Search(ctx, unitVector(0), 5, document.MetadataFilter{
		Metadata: map[string]string{"category": "ccnubox", "module": "bff"},
	})
	if err != nil || len(results) != 1 || results[0].DocumentID != doc.ID {
		t.Fatalf("Search(ccnubox+bff) = %#v, error %v, want only the first document", results, err)
	}
	// Contradictory scalar constraints match nothing.
	if results, err = repository.Search(ctx, unitVector(0), 5, document.MetadataFilter{
		Metadata: map[string]string{"category": "ccnubox", "module": "cli"},
	}); err != nil || len(results) != 0 {
		t.Fatalf("Search(contradictory) = %#v, error %v, want empty", results, err)
	}
	// Tags use any-of overlap.
	results, err = repository.Search(ctx, unitVector(0), 5, document.MetadataFilter{
		Tags: []string{"notes/cli", "missing"},
	})
	if err != nil || len(results) != 1 || results[0].DocumentID != guide.ID {
		t.Fatalf("Search(tags any-of) = %#v, error %v, want only the guide document", results, err)
	}
	// Scalar and tag constraints combine conjunctively.
	results, err = repository.Search(ctx, unitVector(0), 5, document.MetadataFilter{
		Metadata: map[string]string{"module": "bff"},
		Tags:     []string{"ccnubox/bff"},
	})
	if err != nil || len(results) != 1 || results[0].DocumentID != doc.ID {
		t.Fatalf("Search(module=bff and tag) = %#v, error %v, want only the first document", results, err)
	}
	// An empty filter keeps the previous, unfiltered behaviour.
	results, err = repository.Search(ctx, unitVector(0), 5, document.MetadataFilter{})
	if err != nil || len(results) != 2 {
		t.Fatalf("Search(no filter) = %#v, error %v, want both documents", results, err)
	}
	// Documents indexed before this feature carry no metadata and are
	// therefore excluded by any metadata filter.
	legacy, _, err := repository.GetOrCreateDocument(ctx, DocumentCreate{
		SourcePath: "legacy/old.md", Title: "Legacy", ContentHash: "legacy-v1",
	})
	if err != nil {
		t.Fatalf("GetOrCreateDocument(legacy) error = %v", err)
	}
	legacyChunks := []VersionChunk{testStoredChunk("v1", 0, "legacy content", unitVector(2))}
	if err := repository.InsertVersion(ctx, legacy.ID, "v1", legacyChunks); err != nil {
		t.Fatalf("InsertVersion(legacy) error = %v", err)
	}
	if err := repository.ActivateVersion(ctx, Activation{
		DocumentID: legacy.ID, Version: "v1", ExpectedChunkCount: 1,
		Title: "Legacy", ContentHash: "legacy-v1",
		Parent:   ActivationParent{Content: "legacy parent", StartLine: 1, EndLine: 2},
		Metadata: document.DocumentMetadata{},
	}); err != nil {
		t.Fatalf("ActivateVersion(legacy) error = %v", err)
	}
	if results, err = repository.Search(ctx, unitVector(0), 5, document.MetadataFilter{
		Metadata: map[string]string{"category": "ccnubox"},
	}); err != nil || len(results) != 1 || results[0].DocumentID != doc.ID {
		t.Fatalf("Search(legacy excluded) = %#v, error %v, want only the first document", results, err)
	}
	// Invalid filters are rejected before the query runs.
	if _, err := repository.Search(ctx, unitVector(0), 5, document.MetadataFilter{
		Metadata: map[string]string{"category": " "},
	}); err == nil {
		t.Fatal("Search(invalid filter) error = nil")
	}

	// The more similar v2 chunk is deliberately incomplete. It must stay
	// invisible and a failed activation must leave v1 current.
	v2 := []VersionChunk{testStoredChunk("v2", 0, "second", unitVector(0))}
	if err := repository.InsertVersion(ctx, doc.ID, "v2", v2); err != nil {
		t.Fatalf("InsertVersion(v2) error = %v", err)
	}
	err = repository.ActivateVersion(ctx, Activation{
		DocumentID: doc.ID, Version: "v2", ExpectedChunkCount: 2,
		Title: "Authentication v2", ContentHash: "file-v2",
		Parent: ActivationParent{Content: "second parent", StartLine: 1, EndLine: 20},
	})
	if !errors.Is(err, ErrIncompleteVersion) {
		t.Fatalf("ActivateVersion(incomplete v2) error = %v, want ErrIncompleteVersion", err)
	}
	results, err = repository.Search(ctx, unitVector(0), 5, document.MetadataFilter{})
	if err != nil {
		t.Fatalf("Search after failed activation error = %v", err)
	}
	// The incomplete v2 must stay invisible even though every other active
	// document is still searchable.
	for _, result := range results {
		if result.DocumentID == doc.ID && result.DocumentVersion != "v1" {
			t.Fatalf("Search after failed activation leaked version %q for document %d", result.DocumentVersion, result.DocumentID)
		}
	}
	foundV1 := false
	for _, result := range results {
		if result.DocumentID == doc.ID {
			foundV1 = true
		}
	}
	if !foundV1 {
		t.Fatalf("Search after failed activation = %#v, want the still-active v1", results)
	}

	deleted, err := repository.DeleteInactiveVersions(ctx, doc.ID)
	if err != nil || deleted != 1 {
		t.Fatalf("DeleteInactiveVersions() = %d, error %v", deleted, err)
	}

	// A successful re-activation replaces the parent text and version tag in
	// the same transaction that flips the active version.
	v3 := []VersionChunk{testStoredChunk("v3", 0, "third", unitVector(0))}
	if err := repository.InsertVersion(ctx, doc.ID, "v3", v3); err != nil {
		t.Fatalf("InsertVersion(v3) error = %v", err)
	}
	if err := repository.ActivateVersion(ctx, Activation{
		DocumentID: doc.ID, Version: "v3", ExpectedChunkCount: 1,
		Title: "Authentication v3", ContentHash: "file-v3",
		Parent: ActivationParent{Content: "third parent", StartLine: 2, EndLine: 30},
	}); err != nil {
		t.Fatalf("ActivateVersion(v3) error = %v", err)
	}
	parents, err = repository.GetParentDocuments(ctx, []int64{doc.ID})
	if err != nil || len(parents) != 1 || parents[0].Version != "v3" ||
		parents[0].Content != "third parent" || parents[0].Title != "Authentication v3" ||
		parents[0].StartLine != 2 || parents[0].EndLine != 30 {
		t.Fatalf("GetParentDocuments(v3) = %#v, error %v, want the re-activated parent", parents, err)
	}

	if err := repository.MarkDocumentDeleted(ctx, doc.ID); err != nil {
		t.Fatalf("MarkDocumentDeleted() error = %v", err)
	}
	results, err = repository.Search(ctx, unitVector(0), 5, document.MetadataFilter{})
	if err != nil {
		t.Fatalf("Search(deleted document) error = %v", err)
	}
	// The deleted document must never match again; the other documents stay.
	for _, result := range results {
		if result.DocumentID == doc.ID {
			t.Fatalf("Search(deleted document) = %#v, want the deleted document excluded", results)
		}
	}
	if len(results) != 2 {
		t.Fatalf("Search(deleted document) = %#v, want the two remaining documents", results)
	}
	// A deleted document never serves parent content either.
	if parents, err = repository.GetParentDocuments(ctx, []int64{doc.ID}); err != nil || len(parents) != 0 {
		t.Fatalf("GetParentDocuments(deleted) = %#v, error %v, want empty", parents, err)
	}
	cancelledParents, cancelParents := context.WithCancel(ctx)
	cancelParents()
	if _, err := repository.GetParentDocuments(cancelledParents, []int64{doc.ID}); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetParentDocuments(cancelled) error = %v, want context.Canceled", err)
	}

	run, err := repository.StartIndexRun(ctx, "sync")
	if err != nil {
		t.Fatalf("StartIndexRun() error = %v", err)
	}
	if err := repository.CompleteIndexRun(ctx, run.ID, 1, 1); err != nil {
		t.Fatalf("CompleteIndexRun() error = %v", err)
	}
	if err := repository.CompleteIndexRun(ctx, run.ID, 1, 1); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("second CompleteIndexRun() error = %v, want ErrInvalidTransition", err)
	}
}

func startPgvector(t *testing.T, ctx context.Context) (testcontainers.Container, string) {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "pgvector/pgvector:0.8.0-pg17",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_DB":       "gorag",
				"POSTGRES_USER":     "gorag",
				"POSTGRES_PASSWORD": "gorag-test",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(90 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start pgvector container: %v", err)
	}
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("pgvector container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("pgvector mapped port: %v", err)
	}
	dsn := fmt.Sprintf("postgres://gorag:gorag-test@%s/gorag?sslmode=disable", net.JoinHostPort(host, port.Port()))
	return container, dsn
}

func applyMigration(t *testing.T, ctx context.Context, dsn string) {
	t.Helper()
	connection, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect for migration: %v", err)
	}
	defer connection.Close(context.Background())
	_, currentFile, _, _ := runtime.Caller(0)
	migrationFiles, err := filepath.Glob(filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations", "*.up.sql"))
	if err != nil || len(migrationFiles) == 0 {
		t.Fatalf("list migrations: %v", err)
	}
	sort.Strings(migrationFiles)
	for _, migrationFile := range migrationFiles {
		sql, err := os.ReadFile(migrationFile)
		if err != nil {
			t.Fatalf("read migration %s: %v", filepath.Base(migrationFile), err)
		}
		if _, err := connection.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply migration %s: %v", filepath.Base(migrationFile), err)
		}
	}
}

func testStoredChunk(version string, index int, content string, vector []float32) VersionChunk {
	return VersionChunk{
		Chunk: document.Chunk{
			SourcePath:         "api/auth.md",
			DocumentTitle:      "Authentication",
			HeadingPath:        []string{"API", "Auth"},
			Index:              index,
			Content:            content,
			StartLine:          index + 1,
			EndLine:            index + 1,
			ContentHash:        fmt.Sprintf("chunk-%s-%d", version, index),
			DocumentVersion:    version,
			EmbeddingModel:     embedding.DefaultModel,
			EmbeddingDimension: embedding.VectorDimension,
		},
		Embedding: vector,
	}
}

func unitVector(index int) []float32 {
	vector := make([]float32, embedding.VectorDimension)
	vector[index] = 1
	return vector
}
