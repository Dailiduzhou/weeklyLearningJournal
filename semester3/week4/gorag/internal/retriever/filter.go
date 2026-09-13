package retriever

import (
	"fmt"

	einoretriever "github.com/cloudwego/eino/components/retriever"

	"gorag/internal/document"
)

// filterOptions is the implementation-specific option payload both retrieval
// backends parse alongside Eino's common options.
type filterOptions struct {
	Filter document.MetadataFilter
}

// WithFilter constrains retrieval to documents whose parsed front matter
// matches: every Metadata key must match exactly and at least one Tags entry
// must be present (any-of overlap). Passing an empty filter is equivalent to
// not constraining the search.
func WithFilter(filter document.MetadataFilter) einoretriever.Option {
	return einoretriever.WrapImplSpecificOptFn(func(options *filterOptions) {
		options.Filter = filter
	})
}

// FilterFromOptions extracts the call-time metadata filter from Eino options.
// Retrieval backends must honour it for filters to survive fusion and parent
// expansion, which forward options verbatim; wrappers can use it to inspect a
// request's filter themselves.
func FilterFromOptions(opts ...einoretriever.Option) (document.MetadataFilter, error) {
	options := einoretriever.GetImplSpecificOptions(&filterOptions{}, opts...)
	if err := options.Filter.Validate(); err != nil {
		return document.MetadataFilter{}, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}
	return options.Filter, nil
}
