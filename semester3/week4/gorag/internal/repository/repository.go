package repository

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgvector "github.com/pgvector/pgvector-go"
	pgxvector "github.com/pgvector/pgvector-go/pgx"

	"gorag/internal/embedding"
)

type Repository struct {
	pool *pgxpool.Pool
}

// Open creates a pool whose connections know how to encode pgvector.Vector.
func Open(ctx context.Context, connectionString string) (*Repository, error) {
	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, fmt.Errorf("repository: parse database connection string: %w", err)
	}
	previousAfterConnect := config.AfterConnect
	config.AfterConnect = func(ctx context.Context, connection *pgx.Conn) error {
		if previousAfterConnect != nil {
			if err := previousAfterConnect(ctx, connection); err != nil {
				return err
			}
		}
		if err := pgxvector.RegisterTypes(ctx, connection); err != nil {
			return fmt.Errorf("register pgvector types: %w", err)
		}
		return nil
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("repository: create database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("repository: ping database: %w", err)
	}
	return New(pool), nil
}

// New wraps a pool. Connections must have pgvector types registered before
// vector reads or writes; Open does this automatically.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Close() {
	if r != nil && r.pool != nil {
		r.pool.Close()
	}
}

func (r *Repository) GetOrCreateDocument(ctx context.Context, input DocumentCreate) (Document, bool, error) {
	if err := validateDocumentCreate(input); err != nil {
		return Document{}, false, err
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO documents (source_path, title, content_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (source_path) DO NOTHING
		RETURNING id, source_path, title, content_hash, current_version, status,
		          created_at, updated_at, indexed_at`, input.SourcePath, input.Title, input.ContentHash)
	documentRecord, err := scanDocument(row)
	if err == nil {
		return documentRecord, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Document{}, false, fmt.Errorf("repository: create document %q: %w", input.SourcePath, err)
	}
	documentRecord, err = r.GetDocumentByPath(ctx, input.SourcePath)
	if err != nil {
		return Document{}, false, err
	}
	return documentRecord, false, nil
}

func (r *Repository) GetDocumentByPath(ctx context.Context, sourcePath string) (Document, error) {
	if err := validateSourcePath(sourcePath); err != nil {
		return Document{}, err
	}
	row := r.pool.QueryRow(ctx, `
		SELECT id, source_path, title, content_hash, current_version, status,
		       created_at, updated_at, indexed_at
		FROM documents WHERE source_path = $1`, sourcePath)
	documentRecord, err := scanDocument(row)
	if err != nil {
		return Document{}, fmt.Errorf("repository: get document by path %q: %w", sourcePath, err)
	}
	return documentRecord, nil
}

func (r *Repository) GetDocument(ctx context.Context, documentID int64) (Document, error) {
	if documentID <= 0 {
		return Document{}, errors.New("repository: document ID must be positive")
	}
	row := r.pool.QueryRow(ctx, `
		SELECT id, source_path, title, content_hash, current_version, status,
		       created_at, updated_at, indexed_at
		FROM documents WHERE id = $1`, documentID)
	documentRecord, err := scanDocument(row)
	if err != nil {
		return Document{}, fmt.Errorf("repository: get document %d: %w", documentID, err)
	}
	return documentRecord, nil
}

// ListDocuments returns the lifecycle metadata needed by a full sync. Chunk
// contents and vectors are deliberately excluded from this boundary.
func (r *Repository) ListDocuments(ctx context.Context) ([]Document, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, source_path, title, content_hash, current_version, status,
		       created_at, updated_at, indexed_at
		FROM documents
		ORDER BY source_path`)
	if err != nil {
		return nil, fmt.Errorf("repository: list documents: %w", err)
	}
	defer rows.Close()

	documents := make([]Document, 0)
	for rows.Next() {
		documentRecord, err := scanDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("repository: scan listed document: %w", err)
		}
		documents = append(documents, documentRecord)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate documents: %w", err)
	}
	return documents, nil
}

// InsertVersion writes all chunks in one transaction. No activation is
// performed here, so an indexing failure cannot expose a partial version.
func (r *Repository) InsertVersion(ctx context.Context, documentID int64, version string, chunks []VersionChunk) (err error) {
	if documentID <= 0 {
		return errors.New("repository: document ID must be positive")
	}
	if strings.TrimSpace(version) == "" {
		return errors.New("repository: document version is empty")
	}
	if len(chunks) == 0 {
		return errors.New("repository: cannot write an empty document version")
	}
	seen := make(map[int]struct{}, len(chunks))
	for i, chunk := range chunks {
		if err := validateVersionChunk(chunk, version); err != nil {
			return fmt.Errorf("repository: validate chunk input %d: %w", i, err)
		}
		if _, exists := seen[chunk.Index]; exists {
			return fmt.Errorf("repository: duplicate chunk index %d in input", chunk.Index)
		}
		seen[chunk.Index] = struct{}{}
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("repository: begin version write: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}
	}()

	batch := &pgx.Batch{}
	for _, chunk := range chunks {
		headingPath := chunk.HeadingPath
		if headingPath == nil {
			headingPath = []string{}
		}
		batch.Queue(`
			INSERT INTO document_chunks (
				document_id, document_version, chunk_index, content, heading_path,
				start_line, end_line, content_hash, embedding_model,
				embedding_dimension, embedding
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			documentID, version, chunk.Index, chunk.Content, headingPath,
			chunk.StartLine, chunk.EndLine, chunk.ContentHash, chunk.EmbeddingModel,
			chunk.EmbeddingDimension, pgvector.NewVector(chunk.Embedding),
		)
	}
	results := tx.SendBatch(ctx, batch)
	for i := range chunks {
		if _, execErr := results.Exec(); execErr != nil {
			_ = results.Close()
			return fmt.Errorf("repository: insert chunk %d for document %d version %q: %w", i, documentID, version, execErr)
		}
	}
	if closeErr := results.Close(); closeErr != nil {
		return fmt.Errorf("repository: finish chunk batch for document %d version %q: %w", documentID, version, closeErr)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("repository: commit document %d version %q: %w", documentID, version, err)
	}
	return nil
}

// ActivateVersion uses a short transaction and checks that indices are the
// exact continuous range [0, ExpectedChunkCount). Database constraints already
// guarantee the fixed model and vector dimension.
func (r *Repository) ActivateVersion(ctx context.Context, activation Activation) (err error) {
	if activation.DocumentID <= 0 || strings.TrimSpace(activation.Version) == "" {
		return errors.New("repository: activation requires document ID and version")
	}
	if activation.ExpectedChunkCount <= 0 {
		return errors.New("repository: expected chunk count must be positive")
	}
	if strings.TrimSpace(activation.Title) == "" || strings.TrimSpace(activation.ContentHash) == "" {
		return errors.New("repository: activation requires title and content hash")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("repository: begin activation: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}
	}()

	var lockedID int64
	if err = tx.QueryRow(ctx, `SELECT id FROM documents WHERE id = $1 FOR UPDATE`, activation.DocumentID).Scan(&lockedID); err != nil {
		return fmt.Errorf("repository: lock document %d for activation: %w", activation.DocumentID, err)
	}
	var count int
	var minimum, maximum *int
	if err = tx.QueryRow(ctx, `
		SELECT count(*)::integer, min(chunk_index), max(chunk_index)
		FROM document_chunks
		WHERE document_id = $1 AND document_version = $2`, activation.DocumentID, activation.Version).
		Scan(&count, &minimum, &maximum); err != nil {
		return fmt.Errorf("repository: validate document %d version %q: %w", activation.DocumentID, activation.Version, err)
	}
	if count != activation.ExpectedChunkCount || minimum == nil || maximum == nil || *minimum != 0 || *maximum != count-1 {
		return fmt.Errorf("%w: document %d version %q has count=%d range=%v..%v, expected count=%d range=0..%d",
			ErrIncompleteVersion, activation.DocumentID, activation.Version, count, nullableInt(minimum), nullableInt(maximum),
			activation.ExpectedChunkCount, activation.ExpectedChunkCount-1)
	}
	command, err := tx.Exec(ctx, `
		UPDATE documents
		SET current_version = $2, status = 'active', title = $3, content_hash = $4,
		    indexed_at = now(), updated_at = now()
		WHERE id = $1`, activation.DocumentID, activation.Version, activation.Title, activation.ContentHash)
	if err != nil {
		return fmt.Errorf("repository: activate document %d version %q: %w", activation.DocumentID, activation.Version, err)
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("repository: activate document %d: %w", activation.DocumentID, pgx.ErrNoRows)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("repository: commit activation for document %d: %w", activation.DocumentID, err)
	}
	return nil
}

func (r *Repository) MarkDocumentDeleted(ctx context.Context, documentID int64) error {
	if documentID <= 0 {
		return errors.New("repository: document ID must be positive")
	}
	command, err := r.pool.Exec(ctx, `
		UPDATE documents SET status = 'deleted', updated_at = now()
		WHERE id = $1 AND status <> 'deleted'`, documentID)
	if err != nil {
		return fmt.Errorf("repository: mark document %d deleted: %w", documentID, err)
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("repository: mark document %d deleted: %w", documentID, ErrInvalidTransition)
	}
	return nil
}

func (r *Repository) DeleteInactiveVersions(ctx context.Context, documentID int64) (int64, error) {
	if documentID <= 0 {
		return 0, errors.New("repository: document ID must be positive")
	}
	command, err := r.pool.Exec(ctx, `
		DELETE FROM document_chunks chunks
		USING documents documents
		WHERE documents.id = $1
		  AND chunks.document_id = documents.id
		  AND (documents.current_version IS NULL OR chunks.document_version <> documents.current_version)`, documentID)
	if err != nil {
		return 0, fmt.Errorf("repository: delete inactive versions for document %d: %w", documentID, err)
	}
	return command.RowsAffected(), nil
}

func (r *Repository) Search(ctx context.Context, queryVector []float32, limit int) ([]SearchResult, error) {
	if len(queryVector) != embedding.VectorDimension {
		return nil, fmt.Errorf("repository: query vector dimension %d, want %d", len(queryVector), embedding.VectorDimension)
	}
	if limit <= 0 {
		return nil, errors.New("repository: search limit must be positive")
	}
	rows, err := r.pool.Query(ctx, `
		SELECT chunks.document_id, documents.source_path, documents.title,
		       chunks.document_version, chunks.chunk_index, chunks.content,
		       chunks.heading_path, chunks.start_line, chunks.end_line,
		       chunks.content_hash, chunks.embedding_model, chunks.embedding_dimension,
		       1 - (chunks.embedding <=> $1) AS similarity
		FROM document_chunks chunks
		JOIN documents ON documents.id = chunks.document_id
		              AND documents.current_version = chunks.document_version
		WHERE documents.status = 'active'
		ORDER BY chunks.embedding <=> $1
		LIMIT $2`, pgvector.NewVector(queryVector), limit)
	if err != nil {
		return nil, fmt.Errorf("repository: vector search: %w", err)
	}
	defer rows.Close()

	results := make([]SearchResult, 0, limit)
	for rows.Next() {
		var result SearchResult
		if err := rows.Scan(
			&result.DocumentID, &result.SourcePath, &result.DocumentTitle,
			&result.DocumentVersion, &result.Index, &result.Content,
			&result.HeadingPath, &result.StartLine, &result.EndLine,
			&result.ContentHash, &result.EmbeddingModel, &result.EmbeddingDimension,
			&result.Similarity,
		); err != nil {
			return nil, fmt.Errorf("repository: scan vector search result: %w", err)
		}
		result.Chunk.DocumentID = strconv.FormatInt(result.DocumentID, 10)
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate vector search results: %w", err)
	}
	return results, nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanDocument(row rowScanner) (Document, error) {
	var documentRecord Document
	err := row.Scan(
		&documentRecord.ID, &documentRecord.SourcePath, &documentRecord.Title,
		&documentRecord.ContentHash, &documentRecord.CurrentVersion, &documentRecord.Status,
		&documentRecord.CreatedAt, &documentRecord.UpdatedAt, &documentRecord.IndexedAt,
	)
	return documentRecord, err
}

func validateDocumentCreate(input DocumentCreate) error {
	if err := validateSourcePath(input.SourcePath); err != nil {
		return err
	}
	if strings.TrimSpace(input.Title) == "" {
		return errors.New("repository: document title is empty")
	}
	if strings.TrimSpace(input.ContentHash) == "" {
		return errors.New("repository: document content hash is empty")
	}
	return nil
}

func validateSourcePath(sourcePath string) error {
	if sourcePath == "" || strings.Contains(sourcePath, "\\") || path.IsAbs(sourcePath) || path.Clean(sourcePath) != sourcePath || sourcePath == "." || strings.HasPrefix(sourcePath, "../") {
		return fmt.Errorf("repository: source path %q must be a normalized relative slash path", sourcePath)
	}
	return nil
}

func validateVersionChunk(chunk VersionChunk, version string) error {
	if chunk.Index < 0 {
		return fmt.Errorf("chunk index %d must be non-negative", chunk.Index)
	}
	if strings.TrimSpace(chunk.Content) == "" {
		return fmt.Errorf("chunk %d content is empty", chunk.Index)
	}
	if chunk.StartLine <= 0 || chunk.EndLine < chunk.StartLine {
		return fmt.Errorf("chunk %d has invalid line range %d..%d", chunk.Index, chunk.StartLine, chunk.EndLine)
	}
	if strings.TrimSpace(chunk.ContentHash) == "" {
		return fmt.Errorf("chunk %d content hash is empty", chunk.Index)
	}
	if chunk.DocumentVersion != "" && chunk.DocumentVersion != version {
		return fmt.Errorf("chunk %d version %q does not match %q", chunk.Index, chunk.DocumentVersion, version)
	}
	if chunk.EmbeddingModel != embedding.DefaultModel {
		return fmt.Errorf("chunk %d embedding model %q, want %q", chunk.Index, chunk.EmbeddingModel, embedding.DefaultModel)
	}
	if chunk.EmbeddingDimension != embedding.VectorDimension {
		return fmt.Errorf("chunk %d embedding dimension %d, want %d", chunk.Index, chunk.EmbeddingDimension, embedding.VectorDimension)
	}
	if len(chunk.Embedding) != embedding.VectorDimension {
		return fmt.Errorf("chunk %d vector dimension %d, want %d", chunk.Index, len(chunk.Embedding), embedding.VectorDimension)
	}
	return nil
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}
