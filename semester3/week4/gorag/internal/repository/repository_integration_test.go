package repository

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
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
	if err := repository.ActivateVersion(ctx, Activation{
		DocumentID: doc.ID, Version: "v1", ExpectedChunkCount: 1,
		Title: "Authentication", ContentHash: "file-v1",
	}); err != nil {
		t.Fatalf("ActivateVersion(v1) error = %v", err)
	}

	results, err := repository.Search(ctx, unitVector(0), 5)
	if err != nil || len(results) != 1 || results[0].DocumentVersion != "v1" {
		t.Fatalf("Search(v1) = %#v, error %v", results, err)
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
	})
	if !errors.Is(err, ErrIncompleteVersion) {
		t.Fatalf("ActivateVersion(incomplete v2) error = %v, want ErrIncompleteVersion", err)
	}
	results, err = repository.Search(ctx, unitVector(0), 5)
	if err != nil || len(results) != 1 || results[0].DocumentVersion != "v1" {
		t.Fatalf("Search after failed activation = %#v, error %v", results, err)
	}

	deleted, err := repository.DeleteInactiveVersions(ctx, doc.ID)
	if err != nil || deleted != 1 {
		t.Fatalf("DeleteInactiveVersions() = %d, error %v", deleted, err)
	}
	if err := repository.MarkDocumentDeleted(ctx, doc.ID); err != nil {
		t.Fatalf("MarkDocumentDeleted() error = %v", err)
	}
	results, err = repository.Search(ctx, unitVector(0), 5)
	if err != nil || len(results) != 0 {
		t.Fatalf("Search(deleted document) = %#v, error %v", results, err)
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
	migrationPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations", "000001_embedding_storage.up.sql")
	sql, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := connection.Exec(ctx, string(sql)); err != nil {
		t.Fatalf("apply migration: %v", err)
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
