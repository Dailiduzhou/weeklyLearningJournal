package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	einoretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvector "github.com/pgvector/pgvector-go/pgx"

	"gorag/internal/bm25"
	"gorag/internal/config"
	"gorag/internal/embedding"
	"gorag/internal/rag"
	"gorag/internal/repository"
	"gorag/internal/retriever"
	"gorag/internal/transport"
)

// SearchStore is the vector-search boundary of repository.Repository needed
// by the online retriever.
type SearchStore interface {
	Search(ctx context.Context, queryVector []float32, limit int) ([]repository.SearchResult, error)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := runApplication(ctx, logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func runApplication(ctx context.Context, logger *slog.Logger) error {
	cfg, err := config.Load("")
	if err != nil {
		return err
	}
	logger.Info("configuration loaded", "config", cfg)

	startupCtx, cancelStartup := context.WithTimeout(ctx, cfg.Startup.CheckTimeout)
	defer cancelStartup()
	database, err := openDatabase(startupCtx, cfg.Database.URL)
	if err != nil {
		return err
	}
	defer database.Close()
	repositoryStore := repository.New(database)

	ollamaHTTPClient := &http.Client{Timeout: cfg.Ollama.Timeout}
	embedder, err := embedding.NewClient(embedding.Config{
		BaseURL: cfg.Ollama.BaseURL, Model: embedding.DefaultModel,
		BatchSize: cfg.Embedding.BatchSize, MaxConcurrency: cfg.Embedding.MaxConcurrency,
		RequestTimeout: cfg.Ollama.Timeout,
	}, embedding.WithHTTPClient(ollamaHTTPClient))
	if err != nil {
		return fmt.Errorf("construct embedding client: %w", err)
	}
	onlineRetriever, closeBM25, err := buildRetriever(cfg, embedder, repositoryStore)
	if err != nil {
		return fmt.Errorf("construct retriever: %w", err)
	}
	defer closeBM25()
	contextBuilder, err := rag.NewContextBuilder(cfg.Retrieval.MaxContext)
	if err != nil {
		return fmt.Errorf("construct context builder: %w", err)
	}
	answerHTTPClient := &http.Client{Timeout: cfg.Answer.Timeout}
	chatModel, err := newHTTPChatModel(cfg.Answer.Provider, cfg.Answer.BaseURL, cfg.Answer.Model, cfg.Answer.APIKey, answerHTTPClient)
	if err != nil {
		return err
	}
	chain, err := rag.NewChain(startupCtx, onlineRetriever, contextBuilder, nil, chatModel)
	if err != nil {
		return fmt.Errorf("construct RAG chain: %w", err)
	}
	answerService, err := rag.NewAnswerService(chain)
	if err != nil {
		return err
	}
	var queryEmbedder embedding.Embedder
	if cfg.Retrieval.Vector.Enabled {
		queryEmbedder = embedder
	}
	checker, err := newDependencyChecker(database, ollamaHTTPClient, answerHTTPClient,
		cfg.Ollama.BaseURL, cfg.Answer.BaseURL, queryEmbedder, embedding.DefaultModel,
		cfg.Answer.Provider, cfg.Answer.Model, cfg.Answer.APIKey, cfg.Embedding.Dimension)
	if err != nil {
		return err
	}
	if err := waitForDependencies(startupCtx, checker, cfg.Startup.RetryInterval); err != nil {
		return err
	}

	listener, err := net.Listen("tcp", cfg.Server.Address)
	if err != nil {
		return fmt.Errorf("listen on %q: %w", cfg.Server.Address, err)
	}
	server := &http.Server{
		Handler:           transport.NewHandler(answerService, checker, logger),
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
	}
	logger.Info("server ready", "address", listener.Addr().String())
	return serve(ctx, server, listener, cfg.Server.ShutdownTimeout)
}

func openDatabase(ctx context.Context, connectionString string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		// pgx parse errors may echo the original URL, which can contain a
		// password. Keep the public/loggable error contextual but sanitized.
		return nil, errors.New("parse database connection string: invalid PostgreSQL URL")
	}
	previousAfterConnect := poolConfig.AfterConnect
	poolConfig.AfterConnect = func(ctx context.Context, connection *pgx.Conn) error {
		if previousAfterConnect != nil {
			if err := previousAfterConnect(ctx, connection); err != nil {
				return err
			}
		}
		return pgxvector.RegisterTypes(ctx, connection)
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}
	return pool, nil
}

func buildRetriever(cfg config.Config, embedder *embedding.Client, store SearchStore) (einoretriever.Retriever, func(), error) {
	noClose := func() {}
	vectorEnabled := cfg.Retrieval.Vector.Enabled
	bm25Enabled := cfg.Retrieval.BM25.Enabled

	var vectorRetriever *retriever.PgVectorRetriever
	if vectorEnabled {
		var err error
		vectorRetriever, err = retriever.NewPgVectorRetriever(embedder, store, retriever.Config{
			CandidateTopK: cfg.Retrieval.TopK, SimilarityThreshold: cfg.Retrieval.SimilarityThreshold,
		})
		if err != nil {
			return nil, noClose, fmt.Errorf("construct vector retriever: %w", err)
		}
	}

	var bm25Retriever *retriever.BM25Retriever
	closeBM25 := noClose
	if bm25Enabled {
		index, err := bm25.Open(cfg.Retrieval.BM25.IndexPath)
		if err != nil {
			return nil, noClose, fmt.Errorf("open bm25 index: %w", err)
		}
		closeBM25 = func() { _ = index.Close() }
		bm25Retriever, err = retriever.NewBM25Retriever(index, retriever.Config{
			CandidateTopK: cfg.Retrieval.TopK, SimilarityThreshold: cfg.Retrieval.BM25.MinScore,
		})
		if err != nil {
			return nil, closeBM25, fmt.Errorf("construct bm25 retriever: %w", err)
		}
	}

	switch {
	case vectorEnabled && bm25Enabled:
		fused, err := retriever.NewFusionRetriever(cfg.Retrieval.TopK, retriever.DefaultFusionK, vectorRetriever, bm25Retriever)
		if err != nil {
			return nil, closeBM25, fmt.Errorf("construct fusion retriever: %w", err)
		}
		return fused, closeBM25, nil
	case vectorEnabled:
		return vectorRetriever, closeBM25, nil
	case bm25Enabled:
		return bm25Retriever, closeBM25, nil
	default:
		return nil, closeBM25, errors.New("at least one of retrieval.vector.enabled or retrieval.bm25.enabled must be true")
	}
}

func serve(ctx context.Context, server *http.Server, listener net.Listener, shutdownTimeout time.Duration) error {
	if server == nil || listener == nil {
		return errors.New("server: HTTP server and listener are required")
	}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		err := <-serveErr
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP during shutdown: %w", err)
		}
		return nil
	}
}
