package retriever

import (
	"context"
	"fmt"
	"sort"
	"strings"

	einoretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"

	"gorag/internal/bm25"
)

// BM25Searcher is implemented by bm25.Index. Its contract requires active
// documents at their current version only, like the vector Searcher.
type BM25Searcher interface {
	Search(ctx context.Context, query string, topK int) ([]bm25.SearchResult, error)
}

// BM25Retriever adapts the local bluge lexical index to Eino's Retriever
// component interface. Scores are raw BM25 values; ScoreThreshold is applied
// as a minimum-score floor (default 0 keeps every match).
type BM25Retriever struct {
	searcher BM25Searcher
	config   Config
}

var _ einoretriever.Retriever = (*BM25Retriever)(nil)

// NewBM25Retriever creates a bounded lexical retriever.
func NewBM25Retriever(searcher BM25Searcher, config Config) (*BM25Retriever, error) {
	if searcher == nil {
		return nil, fmt.Errorf("%w: searcher is nil", ErrInvalidConfig)
	}
	config = withDefaults(config)
	if err := validateLimits(config.CandidateTopK, config.SimilarityThreshold); err != nil {
		return nil, err
	}
	return &BM25Retriever{searcher: searcher, config: config}, nil
}

// Retrieve performs the BM25 search and returns deterministic Eino Documents.
// Standard Eino TopK and ScoreThreshold call options override configured
// defaults; the threshold is a minimum BM25 score, not a cosine similarity.
func (r *BM25Retriever) Retrieve(ctx context.Context, query string, opts ...einoretriever.Option) ([]*schema.Document, error) {
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

	results, err := r.searcher.Search(ctx, query, *options.TopK)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSearch, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].SourcePath != results[j].SourcePath {
			return results[i].SourcePath < results[j].SourcePath
		}
		if results[i].DocumentVersion != results[j].DocumentVersion {
			return results[i].DocumentVersion < results[j].DocumentVersion
		}
		return results[i].Index < results[j].Index
	})

	documents := make([]*schema.Document, 0, min(len(results), *options.TopK))
	for _, result := range results {
		if len(documents) == *options.TopK {
			break
		}
		if result.Score < *options.ScoreThreshold {
			continue
		}
		documents = append(documents, toEinoDocument(repositoryResult(result)))
	}
	return documents, nil
}

// repositoryResult converts a bm25 hit into the shared search-result shape so
// the same document mapping code serves both backends.
func repositoryResult(result bm25.SearchResult) repositorySearchResult {
	return repositorySearchResult{
		DocumentID: result.DocumentID,
		Chunk:      result.Chunk,
		Similarity: result.Score,
	}
}
