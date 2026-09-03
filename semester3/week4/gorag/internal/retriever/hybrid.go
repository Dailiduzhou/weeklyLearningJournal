package retriever

import (
	"context"
	"errors"
	"fmt"
	"sort"

	einoretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

// DefaultFusionK is the standard reciprocal-rank-fusion smoothing constant.
const DefaultFusionK = 60.0

// ErrInvalidFusion identifies fusion construction failures.
var ErrInvalidFusion = errors.New("retriever: invalid fusion configuration")

// FusionRetriever combines two parallel retrievers (typically vector and
// BM25) with reciprocal rank fusion: each document's fused score is the sum
// of 1/(k + rank) across the result lists it appears in. The fused score is
// exposed as the document score so downstream context selection stays
// backend-agnostic; it is a rank statistic, not a similarity.
type FusionRetriever struct {
	retrievers []einoretriever.Retriever
	k          float64
	topK       int
}

var _ einoretriever.Retriever = (*FusionRetriever)(nil)

// NewFusionRetriever fuses the given retrievers with reciprocal rank fusion.
// The candidate limit matches the vector retriever's CandidateTopK.
func NewFusionRetriever(topK int, k float64, retrievers ...einoretriever.Retriever) (*FusionRetriever, error) {
	if topK <= 0 || topK > 100 {
		return nil, fmt.Errorf("%w: TopK must be between 1 and 100, got %d", ErrInvalidFusion, topK)
	}
	if k <= 0 {
		return nil, fmt.Errorf("%w: fusion constant k must be positive, got %v", ErrInvalidFusion, k)
	}
	active := make([]einoretriever.Retriever, 0, len(retrievers))
	for _, retriever := range retrievers {
		if retriever == nil {
			return nil, fmt.Errorf("%w: retriever is nil", ErrInvalidFusion)
		}
		active = append(active, retriever)
	}
	if len(active) < 2 {
		return nil, fmt.Errorf("%w: at least two retrievers are required, got %d", ErrInvalidFusion, len(active))
	}
	return &FusionRetriever{retrievers: active, k: k, topK: topK}, nil
}

// Retrieve runs every underlying retriever with the same call options and
// merges their result lists into one deterministically ranked set.
func (r *FusionRetriever) Retrieve(ctx context.Context, query string, opts ...einoretriever.Option) ([]*schema.Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	fused := make(map[string]float64)
	byID := make(map[string]*schema.Document)
	for _, retriever := range r.retrievers {
		documents, err := retriever.Retrieve(ctx, query, opts...)
		if err != nil {
			return nil, err
		}
		for rank, document := range documents {
			if document == nil {
				return nil, fmt.Errorf("%w: retriever returned a nil document at rank %d", ErrInvalidFusion, rank)
			}
			if _, exists := byID[document.ID]; !exists {
				byID[document.ID] = document
			}
			fused[document.ID] += 1.0 / (r.k + float64(rank+1))
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ranked := make([]*schema.Document, 0, len(fused))
	for _, document := range byID {
		ranked = append(ranked, document)
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		left, right := fused[ranked[i].ID], fused[ranked[j].ID]
		if left != right {
			return left > right
		}
		return ranked[i].ID < ranked[j].ID
	})
	if len(ranked) > r.topK {
		ranked = ranked[:r.topK]
	}
	for _, document := range ranked {
		fusedScore := fused[document.ID]
		document = document.WithScore(fusedScore)
		if document.MetaData == nil {
			document.MetaData = map[string]any{}
		}
		document.MetaData[MetadataSimilarity] = fusedScore
	}
	return ranked, nil
}
