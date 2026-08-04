package llm

import (
	"context"
	"encoding/json"
	"fmt"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"parameters"`
}

type Request struct {
	Messages []Message
	Tools    []ToolSpec
}

type Response struct {
	Message Message
}

type Client interface {
	Complete(context.Context, Request) (Response, error)
}

type ErrorKind string

const (
	ErrorTemporary   ErrorKind = "temporary"
	ErrorPermanent   ErrorKind = "permanent"
	ErrorInvalidData ErrorKind = "invalid_response"
)

type Error struct {
	Kind      ErrorKind
	Retryable bool
	Err       error
}

func (e *Error) Error() string { return fmt.Sprintf("llm %s error: %v", e.Kind, e.Err) }
func (e *Error) Unwrap() error { return e.Err }
