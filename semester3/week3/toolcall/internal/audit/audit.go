package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"toolcall/internal/tool"
)

type Entry struct {
	TaskID       string        `json:"task_id"`
	Round        int           `json:"round"`
	ToolName     string        `json:"tool_name"`
	ToolType     tool.Type     `json:"tool_type"`
	Arguments    any           `json:"arguments_summary"`
	Status       string        `json:"status"`
	Duration     time.Duration `json:"duration"`
	ErrorType    string        `json:"error_type,omitempty"`
	ResultBytes  int           `json:"result_bytes"`
	WasTruncated bool          `json:"truncated"`
}

type Logger interface {
	Log(context.Context, Entry)
}

type SlogLogger struct {
	logger *slog.Logger
}

func NewSlog(logger *slog.Logger) *SlogLogger { return &SlogLogger{logger: logger} }

func (l *SlogLogger) Log(ctx context.Context, entry Entry) {
	l.logger.LogAttrs(ctx, slog.LevelInfo, "tool_call",
		slog.String("task_id", entry.TaskID),
		slog.Int("round", entry.Round),
		slog.String("tool_name", entry.ToolName),
		slog.String("tool_type", string(entry.ToolType)),
		slog.Any("arguments_summary", entry.Arguments),
		slog.String("status", entry.Status),
		slog.Duration("duration", entry.Duration),
		slog.String("error_type", entry.ErrorType),
		slog.Int("result_bytes", entry.ResultBytes),
		slog.Bool("truncated", entry.WasTruncated),
	)
}

type Discard struct{}

func (Discard) Log(context.Context, Entry) {}

func RedactArguments(raw json.RawMessage) any {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return map[string]any{"invalid_json": true, "size_bytes": len(raw)}
	}
	return redact(value)
}

func redact(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, child := range v {
			if sensitive(key) {
				out[key] = "[REDACTED]"
			} else {
				out[key] = redact(child)
			}
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, child := range v {
			out[i] = redact(child)
		}
		return out
	case string:
		if len(v) > 200 {
			return v[:200] + "…"
		}
		return v
	default:
		return value
	}
}

func sensitive(key string) bool {
	key = strings.ToLower(key)
	for _, marker := range []string{"password", "passwd", "secret", "token", "api_key", "apikey", "dsn", "connection_string"} {
		if strings.Contains(key, marker) {
			return true
		}
	}
	return false
}
