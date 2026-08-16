package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"gorag/internal/config"
	"gorag/internal/document/loader"
	"gorag/internal/document/splitter"
	"gorag/internal/embedding"
	indexerusecase "gorag/internal/indexer"
	"gorag/internal/repository"
)

const defaultMaxFileSize = 10 << 20

type lifecycle interface {
	Sync(context.Context) (indexerusecase.Result, error)
	IndexFile(context.Context, string) (indexerusecase.Result, error)
	DeleteFile(context.Context, string) (indexerusecase.Result, error)
	ReindexFile(context.Context, string) (indexerusecase.Result, error)
	ReindexAll(context.Context) (indexerusecase.Result, error)
}

type cliOptions struct {
	configPath string
	command    string
	target     string
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	os.Exit(run(ctx, os.Args[1:], logger, os.Stderr))
}

func run(ctx context.Context, args []string, logger *slog.Logger, stderr io.Writer) int {
	options, err := parseArgs(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stderr)
			return 0
		}
		fmt.Fprintf(stderr, "indexer: %v\n", err)
		printUsage(stderr)
		return 2
	}
	cfg, err := config.Load(options.configPath)
	if err != nil {
		logger.Error("load configuration", "error", err)
		return 2
	}

	documentLoader, err := loader.New(defaultMaxFileSize)
	if err != nil {
		logger.Error("create document loader", "error", err)
		return 1
	}
	documentSplitter, err := splitter.New(splitter.DefaultConfig())
	if err != nil {
		logger.Error("create document splitter", "error", err)
		return 1
	}
	embedder, err := embedding.NewClient(embedding.Config{
		BaseURL: cfg.Ollama.BaseURL, Model: embedding.DefaultModel,
		RequestTimeout: cfg.Ollama.Timeout,
	})
	if err != nil {
		logger.Error("create embedding client", "error", err)
		return 1
	}
	store, err := repository.Open(ctx, cfg.Database.URL)
	if err != nil {
		logger.Error("open repository", "error", err)
		return 1
	}
	defer store.Close()

	usecase, err := indexerusecase.New(indexerusecase.Config{DocsRoot: cfg.Documents.Dir}, documentLoader, documentSplitter, embedder, store)
	if err != nil {
		logger.Error("create indexer", "error", err)
		return 1
	}
	result, err := dispatch(ctx, usecase, options)
	if err != nil {
		logger.Error("index operation failed",
			"operation", options.command, "target", options.target,
			"run_id", result.RunID, "documents", result.Documents,
			"chunks", result.Chunks, "added", result.Added,
			"updated", result.Updated, "deleted", result.Deleted,
			"skipped", result.Skipped, "failed", result.Failed,
			"failure_paths", result.FailurePaths, "error", err)
		return 1
	}
	logger.Info("index operation completed",
		"operation", options.command, "target", options.target,
		"run_id", result.RunID, "documents", result.Documents,
		"chunks", result.Chunks, "added", result.Added,
		"updated", result.Updated, "deleted", result.Deleted,
		"skipped", result.Skipped, "failed", result.Failed)
	return 0
}

func dispatch(ctx context.Context, usecase lifecycle, options cliOptions) (indexerusecase.Result, error) {
	switch options.command {
	case "sync":
		return usecase.Sync(ctx)
	case "index":
		return usecase.IndexFile(ctx, options.target)
	case "delete":
		return usecase.DeleteFile(ctx, options.target)
	case "reindex":
		return usecase.ReindexFile(ctx, options.target)
	case "reindex-all":
		return usecase.ReindexAll(ctx)
	default:
		return indexerusecase.Result{}, fmt.Errorf("unsupported command %q", options.command)
	}
}

func parseArgs(args []string) (cliOptions, error) {
	options := cliOptions{configPath: config.DefaultPath}
	positionals := make([]string, 0, 2)
	for index := 0; index < len(args); index++ {
		argument := args[index]
		switch {
		case argument == "-h" || argument == "--help":
			return cliOptions{}, flag.ErrHelp
		case argument == "-config" || argument == "--config":
			index++
			if index >= len(args) || strings.TrimSpace(args[index]) == "" {
				return cliOptions{}, errors.New("--config requires a path")
			}
			options.configPath = args[index]
		case strings.HasPrefix(argument, "--config="):
			options.configPath = strings.TrimPrefix(argument, "--config=")
			if strings.TrimSpace(options.configPath) == "" {
				return cliOptions{}, errors.New("--config requires a path")
			}
		case strings.HasPrefix(argument, "-"):
			return cliOptions{}, fmt.Errorf("unknown option %q", argument)
		default:
			positionals = append(positionals, argument)
		}
	}
	if len(positionals) == 0 {
		return cliOptions{}, errors.New("a command is required")
	}
	options.command = positionals[0]
	requiresTarget := options.command == "index" || options.command == "delete" || options.command == "reindex"
	if options.command != "sync" && options.command != "reindex-all" && !requiresTarget {
		return cliOptions{}, fmt.Errorf("unsupported command %q", options.command)
	}
	if requiresTarget {
		if len(positionals) != 2 {
			return cliOptions{}, fmt.Errorf("command %q requires exactly one target path", options.command)
		}
		options.target = positionals[1]
		return options, nil
	}
	if len(positionals) != 1 {
		return cliOptions{}, fmt.Errorf("command %q does not accept a target path", options.command)
	}
	return options, nil
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, "usage: indexer [--config path] <sync|index|delete|reindex|reindex-all> [relative-document-path]")
}
