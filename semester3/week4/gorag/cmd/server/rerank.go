package main

import (
	"fmt"
	"log/slog"
	"net/http"

	einoretriever "github.com/cloudwego/eino/components/retriever"

	"gorag/internal/config"
	"gorag/internal/retriever"
)

// buildRerankRetriever is wired after fusion and before parent expansion.
// Construction performs no network calls; the optional model is deliberately
// absent from both startup dependency checks and /readyz.
func buildRerankRetriever(cfg config.RerankConfig, child einoretriever.Retriever, logger *slog.Logger) (einoretriever.Retriever, error) {
	if !cfg.Enabled {
		return child, nil
	}
	chatModel, err := newHTTPChatModel(cfg.Provider, cfg.BaseURL, cfg.Model, cfg.APIKey, &http.Client{Timeout: cfg.Timeout})
	if err != nil {
		return nil, fmt.Errorf("construct rerank model: %w", err)
	}
	ranked, err := retriever.NewLLMRerankRetriever(child, chatModel, retriever.RerankConfig{
		Timeout: cfg.Timeout, MaxInputChars: cfg.MaxInputChars, MaxConcurrency: cfg.MaxConcurrency,
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("construct rerank retriever: %w", err)
	}
	return ranked, nil
}
