package repository

import (
	"errors"
	"time"

	"gorag/internal/document"
)

const (
	DocumentStatusActive  = "active"
	DocumentStatusDeleted = "deleted"

	IndexRunStatusRunning   = "running"
	IndexRunStatusCompleted = "completed"
	IndexRunStatusFailed    = "failed"
)

var (
	ErrIncompleteVersion = errors.New("repository: document version is incomplete")
	ErrInvalidTransition = errors.New("repository: invalid state transition")
)

type DocumentCreate struct {
	SourcePath  string
	Title       string
	ContentHash string
}

type Document struct {
	ID             int64
	SourcePath     string
	Title          string
	ContentHash    string
	CurrentVersion *string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	IndexedAt      *time.Time
}

// VersionChunk combines the document pipeline's shared contract with the
// vector produced by embedding. Repository-specific storage fields do not
// leak back into document processing.
type VersionChunk struct {
	document.Chunk
	Embedding []float32
}

type Activation struct {
	DocumentID         int64
	Version            string
	ExpectedChunkCount int
	Title              string
	ContentHash        string
}

type SearchResult struct {
	DocumentID int64
	document.Chunk
	Similarity float64
}

type IndexRun struct {
	ID            int64
	RunType       string
	Status        string
	StartedAt     time.Time
	CompletedAt   *time.Time
	DocumentCount int
	ChunkCount    int
	ErrorMessage  *string
}
