package rag

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/schema"

	queryretriever "gorag/internal/retriever"
)

const DefaultMaxContext = 5

var (
	// ErrInvalidContext identifies malformed retrieval metadata or builder
	// configuration. It is a dependency/data-contract failure, not permission
	// to ask the model without sources.
	ErrInvalidContext = errors.New("rag: invalid retrieval context")
	// ErrInsufficientContext is the normal no-answer result for an empty or
	// fully filtered retrieval set.
	ErrInsufficientContext = errors.New("rag: insufficient context")
)

// Source is the stable, request-scoped citation record exposed to the answer
// and citation-validation layer.
type Source struct {
	ID              string
	DocumentID      string
	SourcePath      string
	DocumentTitle   string
	HeadingPath     []string
	StartLine       int
	EndLine         int
	ChunkIndex      int
	DocumentVersion string
	Similarity      float64
}

// BuiltContext contains the exact prompt context and both directions of the
// source mapping. DocumentsBySource maps S1/S2/... back to the retrieved Eino
// document without relying on global state.
type BuiltContext struct {
	Text              string
	Sources           []Source
	DocumentsBySource map[string]*schema.Document
}

// ContextBuilder selects a final, separately bounded set from the candidate
// TopK returned by the retriever.
type ContextBuilder struct {
	MaxContext int
}

// NewContextBuilder creates a final-context selector. A zero value uses five
// chunks, independently of the retriever candidate count.
func NewContextBuilder(maxContext int) (*ContextBuilder, error) {
	if maxContext == 0 {
		maxContext = DefaultMaxContext
	}
	if maxContext < 1 || maxContext > 100 {
		return nil, fmt.Errorf("%w: MaxContext must be between 1 and 100, got %d", ErrInvalidContext, maxContext)
	}
	return &ContextBuilder{MaxContext: maxContext}, nil
}

// Build deterministically ranks documents, assigns S1..Sn, and creates prompt
// text plus a reverse mapping to the exact retrieval results.
func (b *ContextBuilder) Build(ctx context.Context, documents []*schema.Document) (BuiltContext, error) {
	if err := ctx.Err(); err != nil {
		return BuiltContext{}, err
	}
	if len(documents) == 0 {
		return BuiltContext{}, ErrInsufficientContext
	}

	ranked := append([]*schema.Document(nil), documents...)
	for i, document := range ranked {
		if document == nil {
			return BuiltContext{}, fmt.Errorf("%w: document %d is nil", ErrInvalidContext, i)
		}
		if strings.TrimSpace(document.Content) == "" {
			return BuiltContext{}, fmt.Errorf("%w: document %q content is empty", ErrInvalidContext, document.ID)
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score() != ranked[j].Score() {
			return ranked[i].Score() > ranked[j].Score()
		}
		leftPath, _ := stringMetadata(ranked[i], queryretriever.MetadataSourcePath)
		rightPath, _ := stringMetadata(ranked[j], queryretriever.MetadataSourcePath)
		if leftPath != rightPath {
			return leftPath < rightPath
		}
		leftIndex, _ := intMetadata(ranked[i], queryretriever.MetadataChunkIndex)
		rightIndex, _ := intMetadata(ranked[j], queryretriever.MetadataChunkIndex)
		if leftIndex != rightIndex {
			return leftIndex < rightIndex
		}
		return ranked[i].ID < ranked[j].ID
	})
	if len(ranked) > b.MaxContext {
		ranked = ranked[:b.MaxContext]
	}

	sources := make([]Source, 0, len(ranked))
	documentsBySource := make(map[string]*schema.Document, len(ranked))
	var text strings.Builder
	for i, document := range ranked {
		if err := ctx.Err(); err != nil {
			return BuiltContext{}, err
		}
		source, err := sourceFromDocument(document)
		if err != nil {
			return BuiltContext{}, err
		}
		source.ID = "S" + strconv.Itoa(i+1)
		sources = append(sources, source)
		documentsBySource[source.ID] = document

		if i > 0 {
			text.WriteString("\n\n")
		}
		fmt.Fprintf(&text, "[%s]\npath: %s\ntitle: %s\nheading: %s\nlines: %d-%d\ncontent:\n%s",
			source.ID, source.SourcePath, source.DocumentTitle,
			strings.Join(source.HeadingPath, " > "), source.StartLine,
			source.EndLine, document.Content)
	}

	return BuiltContext{
		Text:              text.String(),
		Sources:           sources,
		DocumentsBySource: documentsBySource,
	}, nil
}

func sourceFromDocument(document *schema.Document) (Source, error) {
	documentID, err := stringMetadata(document, queryretriever.MetadataDocumentID)
	if err != nil {
		return Source{}, err
	}
	sourcePath, err := stringMetadata(document, queryretriever.MetadataSourcePath)
	if err != nil {
		return Source{}, err
	}
	title, err := stringMetadata(document, queryretriever.MetadataDocumentTitle)
	if err != nil {
		return Source{}, err
	}
	headingPath, err := stringsMetadata(document, queryretriever.MetadataHeadingPath)
	if err != nil {
		return Source{}, err
	}
	startLine, err := intMetadata(document, queryretriever.MetadataStartLine)
	if err != nil {
		return Source{}, err
	}
	endLine, err := intMetadata(document, queryretriever.MetadataEndLine)
	if err != nil {
		return Source{}, err
	}
	chunkIndex, err := intMetadata(document, queryretriever.MetadataChunkIndex)
	if err != nil {
		return Source{}, err
	}
	version, err := stringMetadata(document, queryretriever.MetadataDocumentVersion)
	if err != nil {
		return Source{}, err
	}
	if startLine <= 0 || endLine < startLine || chunkIndex < 0 {
		return Source{}, fmt.Errorf("%w: document %q has invalid location", ErrInvalidContext, document.ID)
	}

	return Source{
		DocumentID:      documentID,
		SourcePath:      sourcePath,
		DocumentTitle:   title,
		HeadingPath:     headingPath,
		StartLine:       startLine,
		EndLine:         endLine,
		ChunkIndex:      chunkIndex,
		DocumentVersion: version,
		Similarity:      document.Score(),
	}, nil
}

func stringMetadata(document *schema.Document, key string) (string, error) {
	value, ok := document.MetaData[key]
	if !ok {
		return "", fmt.Errorf("%w: document %q metadata %q is missing", ErrInvalidContext, document.ID, key)
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("%w: document %q metadata %q is not a non-empty string", ErrInvalidContext, document.ID, key)
	}
	return text, nil
}

func stringsMetadata(document *schema.Document, key string) ([]string, error) {
	value, ok := document.MetaData[key]
	if !ok {
		return nil, fmt.Errorf("%w: document %q metadata %q is missing", ErrInvalidContext, document.ID, key)
	}
	values, ok := value.([]string)
	if !ok {
		return nil, fmt.Errorf("%w: document %q metadata %q is not []string", ErrInvalidContext, document.ID, key)
	}
	return append([]string(nil), values...), nil
}

func intMetadata(document *schema.Document, key string) (int, error) {
	value, ok := document.MetaData[key]
	if !ok {
		return 0, fmt.Errorf("%w: document %q metadata %q is missing", ErrInvalidContext, document.ID, key)
	}
	switch number := value.(type) {
	case int:
		return number, nil
	case int32:
		return int(number), nil
	case int64:
		return int(number), nil
	case float64:
		integer := int(number)
		if float64(integer) == number {
			return integer, nil
		}
	}
	return 0, fmt.Errorf("%w: document %q metadata %q is not an integer", ErrInvalidContext, document.ID, key)
}
