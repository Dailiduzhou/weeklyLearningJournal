// Package transport maps the RAG answer boundary to net/http without owning
// retrieval, citation, or dependency business rules.
package transport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"gorag/internal/rag"
)

const DefaultMaxRequestBytes int64 = 1 << 20

type Answerer interface {
	AnswerQuestion(context.Context, string) (rag.Answer, error)
}

type ReadyChecker interface {
	Check(context.Context) error
}

type HandlerConfig struct {
	MaxRequestBytes int64
}

type API struct {
	answerer        Answerer
	ready           ReadyChecker
	logger          *slog.Logger
	maxRequestBytes int64
}

func NewHandler(answerer Answerer, ready ReadyChecker, logger *slog.Logger) http.Handler {
	return NewHandlerWithConfig(answerer, ready, logger, HandlerConfig{})
}

func NewHandlerWithConfig(answerer Answerer, ready ReadyChecker, logger *slog.Logger, config HandlerConfig) http.Handler {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if config.MaxRequestBytes <= 0 {
		config.MaxRequestBytes = DefaultMaxRequestBytes
	}
	api := &API{answerer: answerer, ready: ready, logger: logger, maxRequestBytes: config.MaxRequestBytes}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", api.health)
	mux.HandleFunc("GET /readyz", api.readiness)
	mux.HandleFunc("POST /api/v1/questions", api.question)
	return mux
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) readiness(w http.ResponseWriter, request *http.Request) {
	if a.ready == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	if err := a.ready.Check(request.Context()); err != nil {
		a.logger.WarnContext(request.Context(), "readiness check failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (a *API) question(w http.ResponseWriter, request *http.Request) {
	if a.answerer == nil {
		a.logger.ErrorContext(request.Context(), "question handler has no answer service")
		writeProblem(w, http.StatusServiceUnavailable, "service_unavailable", "问答服务暂不可用")
		return
	}
	request.Body = http.MaxBytesReader(w, request.Body, a.maxRequestBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var input struct {
		Question string `json:"question"`
	}
	if err := decoder.Decode(&input); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeProblem(w, http.StatusRequestEntityTooLarge, "request_too_large", "请求体过大")
			return
		}
		writeProblem(w, http.StatusBadRequest, "invalid_json", "请求必须是合法的 JSON 对象")
		return
	}
	if err := ensureJSONEnd(decoder); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeProblem(w, http.StatusRequestEntityTooLarge, "request_too_large", "请求体过大")
			return
		}
		writeProblem(w, http.StatusBadRequest, "invalid_json", "请求体只能包含一个 JSON 对象")
		return
	}
	input.Question = strings.TrimSpace(input.Question)
	if input.Question == "" {
		writeProblem(w, http.StatusBadRequest, "empty_question", "question 不能为空")
		return
	}
	if err := request.Context().Err(); err != nil {
		writeProblem(w, http.StatusRequestTimeout, "request_cancelled", "请求已取消或超时")
		return
	}

	answer, err := a.answerer.AnswerQuestion(request.Context(), input.Question)
	if ctxErr := request.Context().Err(); ctxErr != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		writeProblem(w, http.StatusRequestTimeout, "request_cancelled", "请求已取消或超时")
		return
	}
	if err != nil {
		// AnswerService pairs dependency/model/validation failures with the safe,
		// uniform refusal. Log the cause, never serialize it.
		a.logger.ErrorContext(request.Context(), "question answered with refusal", "error", err)
		answer = rag.Answer{Answerable: false, Text: rag.RefusalAnswer, Sources: []rag.AnswerSource{}}
	}
	writeJSON(w, http.StatusOK, answer)
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("additional JSON value")
	}
	return err
}

func writeProblem(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
