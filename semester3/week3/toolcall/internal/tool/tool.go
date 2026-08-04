package tool

import (
	"context"
	"encoding/json"
)

type Type string

const (
	TypeRead  Type = "read"
	TypeWrite Type = "write"
)

type Definition struct {
	Name        string
	Description string
	Schema      map[string]any
	Type        Type
}

type Error struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

type Result struct {
	OK        bool   `json:"ok"`
	Data      any    `json:"data,omitempty"`
	Error     *Error `json:"error,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Size      int    `json:"size_bytes"`
	Truncated bool   `json:"truncated"`
}

func Success(data any, summary string) Result {
	return Result{OK: true, Data: data, Summary: summary}
}

func Failure(code, message string, retryable bool) Result {
	return Result{OK: false, Error: &Error{Code: code, Message: message, Retryable: retryable}}
}

type Tool interface {
	Definition() Definition
	Execute(context.Context, json.RawMessage) Result
}
