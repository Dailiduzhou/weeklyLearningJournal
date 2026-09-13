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
	// FrontMatterLists holds list-valued front matter such as tags. It is
	// populated by the cleaner, like FrontMatter.
	FrontMatterLists map[string][]string `json:"front_matter_lists,omitempty"`
	Version          string              `json:"version,omitempty"`
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

// TagsKey is the front-matter list key with array semantics: it is stored as
// a first-class tag array and filtered with any-of overlap.
const TagsKey = "tags"

// DocumentMetadata is the parsed, indexable front matter of a document.
// Scalar and list-valued keys are kept apart so every backend can store and
// match them with their native types. Metadata is document-level: chunks
// inherit it.
type DocumentMetadata struct {
	Scalars map[string]string   `json:"scalars,omitempty"`
	Lists   map[string][]string `json:"lists,omitempty"`
}

// Tags returns the tag list, or nil when the document has none.
func (m DocumentMetadata) Tags() []string {
	return m.Lists[TagsKey]
}

// MetadataFilter selects documents by parsed front matter: every metadata key
// must match exactly and at least one tag must be present (any-of overlap).
type MetadataFilter struct {
	Metadata map[string]string `json:"metadata"`
	Tags     []string          `json:"tags"`
}

// Empty reports whether the filter constrains nothing.
func (f MetadataFilter) Empty() bool {
	return len(f.Metadata) == 0 && len(f.Tags) == 0
}
