package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func (r *Repository) StartIndexRun(ctx context.Context, runType string) (IndexRun, error) {
	if !validRunType(runType) {
		return IndexRun{}, fmt.Errorf("repository: unsupported index run type %q", runType)
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO index_runs (run_type) VALUES ($1)
		RETURNING id, run_type, status, started_at, completed_at,
		          document_count, chunk_count, error_message`, runType)
	run, err := scanIndexRun(row)
	if err != nil {
		return IndexRun{}, fmt.Errorf("repository: start %q index run: %w", runType, err)
	}
	return run, nil
}

func (r *Repository) CompleteIndexRun(ctx context.Context, runID int64, documentCount, chunkCount int) error {
	return r.finishIndexRun(ctx, runID, IndexRunStatusCompleted, documentCount, chunkCount, nil)
}

func (r *Repository) FailIndexRun(ctx context.Context, runID int64, documentCount, chunkCount int, runErr error) error {
	if runErr == nil {
		return errors.New("repository: failed index run requires an error")
	}
	message := strings.TrimSpace(runErr.Error())
	if len(message) > 4096 {
		message = message[:4096]
	}
	return r.finishIndexRun(ctx, runID, IndexRunStatusFailed, documentCount, chunkCount, &message)
}

func (r *Repository) finishIndexRun(ctx context.Context, runID int64, status string, documentCount, chunkCount int, errorMessage *string) error {
	if runID <= 0 {
		return errors.New("repository: index run ID must be positive")
	}
	if documentCount < 0 || chunkCount < 0 {
		return errors.New("repository: index run counts cannot be negative")
	}
	command, err := r.pool.Exec(ctx, `
		UPDATE index_runs
		SET status = $2, completed_at = now(), document_count = $3,
		    chunk_count = $4, error_message = $5
		WHERE id = $1 AND status = 'running'`, runID, status, documentCount, chunkCount, errorMessage)
	if err != nil {
		return fmt.Errorf("repository: finish index run %d as %s: %w", runID, status, err)
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("repository: finish index run %d: %w", runID, ErrInvalidTransition)
	}
	return nil
}

func scanIndexRun(row rowScanner) (IndexRun, error) {
	var run IndexRun
	err := row.Scan(
		&run.ID, &run.RunType, &run.Status, &run.StartedAt, &run.CompletedAt,
		&run.DocumentCount, &run.ChunkCount, &run.ErrorMessage,
	)
	return run, err
}

func validRunType(runType string) bool {
	switch runType {
	case "sync", "add", "delete", "reindex":
		return true
	default:
		return false
	}
}
