// Package retriever implements the query-side vector retrieval boundary.
package retriever

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	einoretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"

	"gorag/internal/embedding"
	"gorag/internal/repository"
)

const (
	// DefaultCandidateTopK is the number of database candidates fetched before
	// the RAG layer applies its separate final-context limit.
	DefaultCandidateTopK = 10
	// DefaultSimilarityThreshold excludes weak cosine matches.
	DefaultSimilarityThreshold = 0.5
)

const (
	MetadataDocumentID         = "document_id"
	MetadataSourcePath         = "source_path"
	MetadataDocumentTitle      = "document_title"
	MetadataHeadingPath        = "heading_path"
	MetadataStartLine          = "start_line"
	MetadataEndLine            = "end_line"
	MetadataChunkIndex         = "chunk_index"
	MetadataContentHash        = "content_hash"
	MetadataDocumentVersion    = "document_version"
	MetadataEmbeddingModel     = "embedding_model"
	MetadataEmbeddingDimension = "embedding_dimension"
	MetadataSimilarity         = "similarity"
)

var (
	// ErrInvalidConfig identifies construction or call-time retrieval options
	// that cannot produce a bounded cosine search.
	ErrInvalidConfig = errors.New("retriever: invalid configuration")
	// ErrEmbedding identifies query-vector generation failures.
	ErrEmbedding = errors.New("retriever: query embedding failed")
	// ErrSearch identifies vector-store query failures.
	ErrSearch = errors.New("retriever: vector search failed")
)

// Config controls candidate retrieval. CandidateTopK is deliberately distinct
// from rag.ContextBuilder's MaxContext.
type Config struct {
	CandidateTopK       int
	SimilarityThreshold float64
}

// QueryEmbedder is the narrow portion of embedding.Embedder needed online.
type QueryEmbedder interface {
	EmbedQuery(context.Context, string) ([]float32, error)
}

// Searcher is implemented by repository.Repository. Its contract requires
// active documents at their current version only.
type Searcher interface {
	Search(context.Context, []float32, int) ([]repository.SearchResult, error)
}

// PgVectorRetriever adapts the local query embedder and repository search to
// Eino's Retriever component interface.
type PgVectorRetriever struct {
	embedder QueryEmbedder
	searcher Searcher
	config   Config
}

var _ einoretriever.Retriever = (*PgVectorRetriever)(nil)

// NewPgVectorRetriever creates a bounded cosine retriever.
func NewPgVectorRetriever(embedder QueryEmbedder, searcher Searcher, config Config) (*PgVectorRetriever, error) {
	if embedder == nil {
		return nil, fmt.Errorf("%w: embedder is nil", ErrInvalidConfig)
	}
	if searcher == nil {
		return nil, fmt.Errorf("%w: searcher is nil", ErrInvalidConfig)
	}
	config = withDefaults(config)
	if err := validateLimits(config.CandidateTopK, config.SimilarityThreshold); err != nil {
		return nil, err
	}
	return &PgVectorRetriever{embedder: embedder, searcher: searcher, config: config}, nil
}

// Retrieve embeds query, performs exact pgvector cosine search, applies the
// score threshold, and returns deterministic Eino Documents. Standard Eino
// TopK and ScoreThreshold call options override configured defaults.
func (r *PgVectorRetriever) Retrieve(ctx context.Context, query string, opts ...einoretriever.Option) ([]*schema.Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("%w: query is empty", ErrInvalidConfig)
	}

	topK := r.config.CandidateTopK
	threshold := r.config.SimilarityThreshold
	options := einoretriever.GetCommonOptions(&einoretriever.Options{
		TopK:           &topK,
		ScoreThreshold: &threshold,
	}, opts...)
	if options.TopK == nil || options.ScoreThreshold == nil {
		return nil, fmt.Errorf("%w: TopK and ScoreThreshold are required", ErrInvalidConfig)
	}
	if err := validateLimits(*options.TopK, *options.ScoreThreshold); err != nil {
		return nil, err
	}

	queryVector, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEmbedding, err)
	}
	if len(queryVector) != embedding.VectorDimension {
		return nil, fmt.Errorf("%w: vector dimension %d, want %d", ErrEmbedding, len(queryVector), embedding.VectorDimension)
	}
	results, err := r.searcher.Search(ctx, queryVector, *options.TopK)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSearch, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// PostgreSQL orders by cosine distance, but an explicit deterministic tie
	// break makes source numbering stable even for equal vectors.
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Similarity != results[j].Similarity {
			return results[i].Similarity > results[j].Similarity
		}
		if results[i].SourcePath != results[j].SourcePath {
			return results[i].SourcePath < results[j].SourcePath
		}
		if results[i].DocumentVersion != results[j].DocumentVersion {
			return results[i].DocumentVersion < results[j].DocumentVersion
		}
		if results[i].Index != results[j].Index {
			return results[i].Index < results[j].Index
		}
		return results[i].DocumentID < results[j].DocumentID
	})

	documents := make([]*schema.Document, 0, min(len(results), *options.TopK))
	for _, result := range results {
		if len(documents) == *options.TopK {
			break
		}
		if result.Similarity < *options.ScoreThreshold {
			continue
		}
		documents = append(documents, toEinoDocument(result))
	}
	return documents, nil
}

func withDefaults(config Config) Config {
	if config.CandidateTopK == 0 && config.SimilarityThreshold == 0 {
		return Config{
			CandidateTopK:       DefaultCandidateTopK,
			SimilarityThreshold: DefaultSimilarityThreshold,
		}
	}
	if config.CandidateTopK == 0 {
		config.CandidateTopK = DefaultCandidateTopK
	}
	return config
}

func validateLimits(topK int, threshold float64) error {
	if topK <= 0 || topK > 100 {
		return fmt.Errorf("%w: CandidateTopK must be between 1 and 100, got %d", ErrInvalidConfig, topK)
	}
	if threshold < -1 || threshold > 1 {
		return fmt.Errorf("%w: similarity threshold must be between -1 and 1, got %v", ErrInvalidConfig, threshold)
	}
	return nil
}

func toEinoDocument(result repository.SearchResult) *schema.Document {
	documentID := strconv.FormatInt(result.DocumentID, 10)
	document := &schema.Document{
		ID:      documentID + ":" + result.DocumentVersion + ":" + strconv.Itoa(result.Index),
		Content: result.Content,
		MetaData: map[string]any{
			MetadataDocumentID:         documentID,
			MetadataSourcePath:         result.SourcePath,
			MetadataDocumentTitle:      result.DocumentTitle,
			MetadataHeadingPath:        append([]string(nil), result.HeadingPath...),
			MetadataStartLine:          result.StartLine,
			MetadataEndLine:            result.EndLine,
			MetadataChunkIndex:         result.Index,
			MetadataContentHash:        result.ContentHash,
			MetadataDocumentVersion:    result.DocumentVersion,
			MetadataEmbeddingModel:     result.EmbeddingModel,
			MetadataEmbeddingDimension: result.EmbeddingDimension,
			MetadataSimilarity:         result.Similarity,
		},
	}
	return document.WithScore(result.Similarity)
}
