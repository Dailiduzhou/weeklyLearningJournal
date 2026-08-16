// Package document defines the data exchanged by the document pipeline and
// the indexing pipeline. It deliberately contains no storage or model logic.
package document

// Kind identifies a supported source document format.
type Kind string

const (
	KindMarkdown Kind = "markdown"
	KindText     Kind = "text"
)

// Document is a source file loaded from the configured docs root.
// SourcePath always uses forward slashes and is relative to that root.
type Document struct {
	ID          string            `json:"id"`
	SourcePath  string            `json:"source_path"`
	Title       string            `json:"title"`
	Kind        Kind              `json:"kind"`
	Content     string            `json:"content"`
	ContentHash string            `json:"content_hash"`
	Size        int64             `json:"size"`
	FrontMatter map[string]string `json:"front_matter,omitempty"`
	Version     string            `json:"version,omitempty"`
	// LineNumbers maps each line of Content back to the 1-based line number
	// in the original source file. It is populated by cleaner.Clean and used
	// by the splitter so citations point at real document lines.
	LineNumbers []int `json:"line_numbers,omitempty"`
}

// Chunk is an ordered, independently indexable portion of a Document.
// Embedding fields are intentionally left empty by this package and are
// populated by the embedding/indexing pipeline.
type Chunk struct {
	DocumentID         string   `json:"document_id"`
	SourcePath         string   `json:"source_path"`
	DocumentTitle      string   `json:"document_title"`
	HeadingPath        []string `json:"heading_path,omitempty"`
	Index              int      `json:"index"`
	Content            string   `json:"content"`
	StartLine          int      `json:"start_line"`
	EndLine            int      `json:"end_line"`
	ContentHash        string   `json:"content_hash"`
	DocumentVersion    string   `json:"document_version,omitempty"`
	EmbeddingModel     string   `json:"embedding_model,omitempty"`
	EmbeddingDimension int      `json:"embedding_dimension,omitempty"`
}
