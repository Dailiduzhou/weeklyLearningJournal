package retriever

// This file implements Parent Document Retrieval on top of Eino's
// flow/retriever/parent package: small chunks stay the searchable unit, but
// each retrieved chunk is expanded back to its whole parent document before
// context selection.

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	einoretriever "github.com/cloudwego/eino/components/retriever"
	einoparent "github.com/cloudwego/eino/flow/retriever/parent"
	"github.com/cloudwego/eino/schema"

	"gorag/internal/repository"
)

// ErrParentFetch identifies parent-document expansion failures, either from
// malformed child metadata or from the parent store itself.
var ErrParentFetch = errors.New("retriever: parent document fetch failed")

// ParentStore is the whole-document store boundary implemented by
// repository.Repository. Its contract is the same as vector Search: only
// active documents at their current version are visible.
type ParentStore interface {
	GetParentDocuments(ctx context.Context, ids []int64) ([]repository.ParentDocument, error)
}

// NewParentDocumentRetriever wraps a child-chunk retriever with Eino's parent
// flow retriever. Child hits are mapped to their parent document by the
// document_id metadata key, deduplicated in child-rank order, and expanded to
// whole documents fetched from the parent store. Each returned parent carries
// the best child score of the request, so context selection keeps ranking by
// retrieval relevance.
func NewParentDocumentRetriever(ctx context.Context, child einoretriever.Retriever, store ParentStore) (einoretriever.Retriever, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if child == nil {
		return nil, fmt.Errorf("%w: child retriever is nil", ErrInvalidConfig)
	}
	if store == nil {
		return nil, fmt.Errorf("%w: parent store is nil", ErrInvalidConfig)
	}
	inner, err := einoparent.NewRetriever(ctx, &einoparent.Config{
		Retriever:     &scoreRecordingRetriever{child: child},
		ParentIDKey:   MetadataDocumentID,
		OrigDocGetter: (&parentDocumentGetter{store: store}).get,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: construct parent document retriever: %w", ErrParentFetch, err)
	}
	return parentDocumentRetriever{inner: inner}, nil
}

// parentScoreCollector keeps the best child score per parent document ID for
// one request. Eino's parent flow retriever does not propagate child scores
// into the parent documents it fetches, so this collector lets the getter
// restore each parent's best-child relevance score request-locally.
type parentScoreCollector struct {
	mu   sync.Mutex
	best map[string]float64
}

func (c *parentScoreCollector) record(parentID string, score float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if current, exists := c.best[parentID]; !exists || score > current {
		c.best[parentID] = score
	}
}

func (c *parentScoreCollector) score(parentID string) float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.best[parentID]
}

// parentScoreCollectorKey namespaces the request-scoped collector.
type parentScoreCollectorKey struct{}

// parentDocumentRetriever opens one fresh collector per request before
// delegating to Eino's parent flow retriever. The adapter satisfies Eino's
// Retriever interface so the RAG chain needs no parent-specific wiring.
type parentDocumentRetriever struct {
	inner einoretriever.Retriever
}

var _ einoretriever.Retriever = parentDocumentRetriever{}

func (r parentDocumentRetriever) Retrieve(ctx context.Context, query string, opts ...einoretriever.Option) ([]*schema.Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, parentScoreCollectorKey{}, &parentScoreCollector{best: make(map[string]float64)})
	return r.inner.Retrieve(ctx, query, opts...)
}

// scoreRecordingRetriever records every child hit's score against its parent
// document ID before returning the children to Eino's parent flow retriever.
type scoreRecordingRetriever struct {
	child einoretriever.Retriever
}

func (r *scoreRecordingRetriever) Retrieve(ctx context.Context, query string, opts ...einoretriever.Option) ([]*schema.Document, error) {
	documents, err := r.child.Retrieve(ctx, query, opts...)
	if err != nil {
		return nil, err
	}
	if collector, ok := ctx.Value(parentScoreCollectorKey{}).(*parentScoreCollector); ok {
		for _, document := range documents {
			if documentID, exists := document.MetaData[MetadataDocumentID].(string); exists && documentID != "" {
				collector.record(documentID, document.Score())
			}
		}
	}
	return documents, nil
}

// parentDocumentGetter adapts ParentStore to Eino's OrigDocGetter contract.
type parentDocumentGetter struct {
	store ParentStore
}

func (g *parentDocumentGetter) get(ctx context.Context, ids []string) ([]*schema.Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []*schema.Document{}, nil
	}
	parsed := make([]int64, 0, len(ids))
	for _, id := range ids {
		documentID, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
		if err != nil || documentID <= 0 {
			return nil, fmt.Errorf("%w: parent document ID %q is invalid", ErrParentFetch, id)
		}
		parsed = append(parsed, documentID)
	}
	parents, err := g.store.GetParentDocuments(ctx, parsed)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrParentFetch, err)
	}
	collector, _ := ctx.Value(parentScoreCollectorKey{}).(*parentScoreCollector)
	documents := make([]*schema.Document, 0, len(parents))
	for _, parent := range parents {
		score := 0.0
		if collector != nil {
			score = collector.score(strconv.FormatInt(parent.DocumentID, 10))
		}
		documents = append(documents, parentToEinoDocument(parent, score))
	}
	return documents, nil
}

// parentToEinoDocument maps a stored parent document onto the same metadata
// contract the RAG context builder applies to chunks. The heading path is
// empty because a parent spans the whole document; the line range cites the
// cleaned content's position in the original source file.
func parentToEinoDocument(parent repository.ParentDocument, score float64) *schema.Document {
	documentID := strconv.FormatInt(parent.DocumentID, 10)
	document := &schema.Document{
		ID:      documentID + ":" + parent.Version + ":parent",
		Content: parent.Content,
		MetaData: map[string]any{
			MetadataDocumentID:      documentID,
			MetadataSourcePath:      parent.SourcePath,
			MetadataDocumentTitle:   parent.Title,
			MetadataHeadingPath:     []string{},
			MetadataStartLine:       parent.StartLine,
			MetadataEndLine:         parent.EndLine,
			MetadataChunkIndex:      0,
			MetadataDocumentVersion: parent.Version,
			MetadataSimilarity:      score,
		},
	}
	return document.WithScore(score)
}
