package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"gorag/internal/httpurl"
)

var vectorTypePattern = regexp.MustCompile(`^vector\(([1-9][0-9]*)\)$`)

type databaseChecker interface {
	Ping(context.Context) error
	QueryRow(context.Context, string, ...any) pgx.Row
}

type embeddingProbe interface {
	EmbedQuery(context.Context, string) ([]float32, error)
	Model() string
	Dimension() int
}

type dependencyChecker struct {
	database          databaseChecker
	httpClient        *http.Client
	answerHTTPClient  *http.Client
	ollamaBaseURL     *url.URL
	answerBaseURL     *url.URL
	answerAPIKey      string
	embedder          embeddingProbe
	embeddingModel    string
	answerProvider    string
	answerModel       string
	expectedDimension int
}

func newDependencyChecker(database databaseChecker, ollamaClient, answerClient *http.Client, ollamaBaseURL, answerBaseURL string, embedder embeddingProbe, embeddingModel, answerProvider, answerModel, answerAPIKey string, dimension int) (*dependencyChecker, error) {
	baseURL, err := url.Parse(ollamaBaseURL)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, errors.New("server: Ollama base URL must be absolute")
	}
	parsedAnswerBaseURL, err := url.Parse(answerBaseURL)
	if err != nil || parsedAnswerBaseURL.Scheme == "" || parsedAnswerBaseURL.Host == "" {
		return nil, errors.New("server: answer model base URL must be absolute")
	}
	if ollamaClient == nil {
		ollamaClient = http.DefaultClient
	}
	if answerClient == nil {
		answerClient = http.DefaultClient
	}
	if database == nil || embedder == nil || dimension <= 0 {
		return nil, errors.New("server: readiness dependencies are incomplete")
	}
	return &dependencyChecker{
		database: database, httpClient: ollamaClient, answerHTTPClient: answerClient,
		ollamaBaseURL: baseURL, answerBaseURL: parsedAnswerBaseURL, answerAPIKey: answerAPIKey,
		embedder: embedder, embeddingModel: embeddingModel,
		answerProvider: answerProvider, answerModel: answerModel,
		expectedDimension: dimension,
	}, nil
}

func (c *dependencyChecker) Check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.database.Ping(ctx); err != nil {
		return fmt.Errorf("PostgreSQL ping: %w", err)
	}
	var vectorInstalled bool
	if err := c.database.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector')`).Scan(&vectorInstalled); err != nil {
		return fmt.Errorf("check vector extension: %w", err)
	}
	if !vectorInstalled {
		return errors.New("check vector extension: extension is not installed")
	}

	var vectorType string
	if err := c.database.QueryRow(ctx, `
		SELECT format_type(a.atttypid, a.atttypmod)
		FROM pg_attribute a
		JOIN pg_class t ON t.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = current_schema()
		  AND t.relname = 'document_chunks'
		  AND a.attname = 'embedding'
		  AND NOT a.attisdropped`).Scan(&vectorType); err != nil {
		return fmt.Errorf("check database vector dimension: %w", err)
	}
	wantType := fmt.Sprintf("vector(%d)", c.expectedDimension)
	if !vectorTypePattern.MatchString(vectorType) || vectorType != wantType {
		return fmt.Errorf("database embedding type is %q, want %q", vectorType, wantType)
	}
	if c.embedder.Model() != c.embeddingModel {
		return fmt.Errorf("embedding client model is %q, want %q", c.embedder.Model(), c.embeddingModel)
	}
	if c.embedder.Dimension() != c.expectedDimension {
		return fmt.Errorf("embedding client dimension is %d, want %d", c.embedder.Dimension(), c.expectedDimension)
	}

	models, err := c.ollamaModels(ctx)
	if err != nil {
		return err
	}
	if !modelsContain(models, c.embeddingModel) {
		return fmt.Errorf("Ollama embedding model %q is unavailable", c.embeddingModel)
	}
	if c.answerProvider == "ollama" {
		if !modelsContain(models, c.answerModel) {
			return fmt.Errorf("Ollama answer model %q is unavailable", c.answerModel)
		}
	} else {
		answerModels, err := c.openAICompatibleModels(ctx)
		if err != nil {
			return err
		}
		if !modelsContain(answerModels, c.answerModel) {
			return fmt.Errorf("OpenAI-compatible answer model %q is unavailable", c.answerModel)
		}
	}

	vector, err := c.embedder.EmbedQuery(ctx, "readiness dimension probe")
	if err != nil {
		return fmt.Errorf("Ollama embedding probe: %w", err)
	}
	if len(vector) != c.expectedDimension {
		return fmt.Errorf("Ollama embedding dimension is %d, want %d", len(vector), c.expectedDimension)
	}
	return nil
}

func (c *dependencyChecker) ollamaModels(ctx context.Context) ([]string, error) {
	endpoint := httpurl.JoinPath(c.ollamaBaseURL, "api/tags")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create Ollama model check: %w", err)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("check Ollama models: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		return nil, fmt.Errorf("check Ollama models: HTTP %d", response.StatusCode)
	}
	var payload struct {
		Models []struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		} `json:"models"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode Ollama models: %w", err)
	}
	models := make([]string, 0, len(payload.Models)*2)
	for _, model := range payload.Models {
		models = append(models, model.Name, model.Model)
	}
	return models, nil
}

// openAICompatibleModels validates the configured answer endpoint and API key
// by listing models through the OpenAI-compatible GET /models surface.
func (c *dependencyChecker) openAICompatibleModels(ctx context.Context) ([]string, error) {
	endpoint := httpurl.OpenAICompatible(c.answerBaseURL, "models")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create OpenAI-compatible model check: %w", err)
	}
	if c.answerAPIKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.answerAPIKey)
	}
	response, err := c.answerHTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("check OpenAI-compatible answer models: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		return nil, fmt.Errorf("check OpenAI-compatible answer models: HTTP %d", response.StatusCode)
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode OpenAI-compatible answer models: %w", err)
	}
	models := make([]string, 0, len(payload.Data))
	for _, model := range payload.Data {
		models = append(models, model.ID)
	}
	return models, nil
}

func modelsContain(models []string, wanted string) bool {
	wanted = normalizeModelName(wanted)
	for _, model := range models {
		if normalizeModelName(model) == wanted {
			return true
		}
	}
	return false
}

func normalizeModelName(model string) string {
	model = strings.TrimSpace(model)
	return strings.TrimSuffix(model, ":latest")
}

func waitForDependencies(ctx context.Context, checker interface{ Check(context.Context) error }, retryInterval time.Duration) error {
	if checker == nil {
		return errors.New("server: dependency checker is nil")
	}
	if retryInterval <= 0 {
		return errors.New("server: retry interval must be positive")
	}
	var lastErr error
	for {
		if err := checker.Check(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}
		timer := time.NewTimer(retryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("server: startup dependency check failed: %w: %v", ctx.Err(), lastErr)
		case <-timer.C:
		}
	}
}
