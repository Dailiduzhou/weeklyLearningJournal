package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"toolcall/internal/agent"
	"toolcall/internal/audit"
	"toolcall/internal/config"
	"toolcall/internal/llm/openaiadapter"
	"toolcall/internal/tool"
	"toolcall/internal/tools/calculator"
	"toolcall/internal/tools/docsearch"
	postgresTool "toolcall/internal/tools/postgres"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("agent-runtime", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "", "path to YAML configuration")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	level := slog.LevelInfo
	if strings.EqualFold(cfg.Audit.Level, "debug") {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(stderr, &slog.HandlerOptions{Level: level}))

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	docs, err := docsearch.New(cfg.Documents.Directory, cfg.Documents.ChunkRunes, cfg.Documents.MaxResults)
	if err != nil {
		return err
	}

	var pool *pgxpool.Pool
	if cfg.Database.Enabled {
		pool, err = pgxpool.New(rootCtx, cfg.Database.DSN)
		if err != nil {
			return fmt.Errorf("create database pool: %w", err)
		}
		defer pool.Close()
		if err := pool.Ping(rootCtx); err != nil {
			return fmt.Errorf("connect database: %w", err)
		}
	}
	queries := make([]postgresTool.QueryDefinition, 0, len(cfg.Database.Queries))
	for _, q := range cfg.Database.Queries {
		params := make([]postgresTool.ParamDefinition, 0, len(q.Params))
		for _, p := range q.Params {
			params = append(params, postgresTool.ParamDefinition{
				Name: p.Name, Type: p.Type, Required: p.Required, MaxLength: p.MaxLength,
				Minimum: p.Minimum, Maximum: p.Maximum,
			})
		}
		queries = append(queries, postgresTool.QueryDefinition{Name: q.Name, Description: q.Description, SQL: q.SQL, Params: params})
	}
	dbTool := postgresTool.New(pool, queries, cfg.Database.QueryTimeout, cfg.Database.MaxRows, cfg.Database.MaxBytes)
	registry, err := tool.NewRegistry(calculator.New(), docs, dbTool)
	if err != nil {
		return err
	}
	model := openaiadapter.New(cfg.Model.APIKey, cfg.Model.BaseURL, cfg.Model.Name)
	runtime, err := agent.New(agent.Config{
		MaxRounds: cfg.Agent.MaxRounds, TaskTimeout: cfg.Agent.TaskTimeout,
		ModelTimeout: cfg.Model.Timeout, ToolTimeout: cfg.Agent.ToolTimeout,
		MaxToolResultBytes: cfg.Agent.MaxToolResultBytes, MaxHistoryBytes: cfg.Agent.MaxHistoryBytes,
		MaxRepeatedFailures: cfg.Agent.MaxRepeatedFailures, MaxUnknownTools: cfg.Agent.MaxUnknownTools,
		ModelRetries: cfg.Model.MaxRetries,
	}, model, registry, audit.NewSlog(logger))
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if prompt := strings.TrimSpace(strings.Join(flags.Args(), " ")); prompt != "" {
		return encoder.Encode(runtime.Run(rootCtx, prompt))
	}
	return interactive(rootCtx, stdin, stderr, encoder, runtime)
}

func interactive(ctx context.Context, input io.Reader, diagnostics io.Writer, encoder *json.Encoder, runtime *agent.Runtime) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for {
		fmt.Fprint(diagnostics, "> ")
		if !scanner.Scan() {
			return scanner.Err()
		}
		prompt := strings.TrimSpace(scanner.Text())
		if prompt == "" {
			continue
		}
		if strings.EqualFold(prompt, "exit") || strings.EqualFold(prompt, "quit") {
			return nil
		}
		if err := encoder.Encode(runtime.Run(ctx, prompt)); err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}
