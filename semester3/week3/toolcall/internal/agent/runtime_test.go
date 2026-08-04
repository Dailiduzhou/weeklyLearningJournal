package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"toolcall/internal/audit"
	"toolcall/internal/llm"
	"toolcall/internal/tool"
	"toolcall/internal/tools/calculator"
)

type modelStep struct {
	response llm.Response
	err      error
}

type scriptedModel struct {
	mu       sync.Mutex
	steps    []modelStep
	requests []llm.Request
}

func (m *scriptedModel) Complete(_ context.Context, request llm.Request) (llm.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = append(m.requests, request)
	if len(m.steps) == 0 {
		return llm.Response{}, errors.New("unexpected model call")
	}
	step := m.steps[0]
	m.steps = m.steps[1:]
	return step.response, step.err
}

func answer(text string) modelStep {
	return modelStep{response: llm.Response{Message: llm.Message{Content: text}}}
}

func calls(content string, toolCalls ...llm.ToolCall) modelStep {
	return modelStep{response: llm.Response{Message: llm.Message{Content: content, ToolCalls: toolCalls}}}
}

func call(id, name, args string) llm.ToolCall {
	return llm.ToolCall{ID: id, Name: name, Arguments: json.RawMessage(args)}
}

func testRuntime(t *testing.T, model llm.Client, auditLogger audit.Logger, tools ...tool.Tool) *Runtime {
	t.Helper()
	registry, err := tool.NewRegistry(tools...)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := New(Config{
		MaxRounds: 4, TaskTimeout: time.Second, ModelTimeout: 200 * time.Millisecond,
		ToolTimeout: 100 * time.Millisecond, MaxToolResultBytes: 512, MaxHistoryBytes: 32 * 1024,
		MaxRepeatedFailures: 3, MaxUnknownTools: 2, ModelRetries: 1,
	}, model, registry, auditLogger)
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}

func TestDirectAnswer(t *testing.T) {
	model := &scriptedModel{steps: []modelStep{answer("done")}}
	result := testRuntime(t, model, nil, calculator.New()).Run(context.Background(), "hello")
	if result.Stop != StopFinalAnswer || result.Answer != "done" || result.Rounds != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestInvalidToolArgumentsThenCorrection(t *testing.T) {
	model := &scriptedModel{steps: []modelStep{
		calls("", call("1", "calculator", `{"expression":7}`)),
		calls("", call("2", "calculator", `{"expression":"6*7"}`)),
		answer("42"),
	}}
	result := testRuntime(t, model, nil, calculator.New()).Run(context.Background(), "calculate")
	if result.Stop != StopFinalAnswer || result.Answer != "42" || len(model.requests) != 3 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !requestContains(model.requests[1], "invalid_arguments") || !requestContains(model.requests[2], `"ok":true`) {
		t.Fatalf("tool results were not returned to model: %#v", model.requests)
	}
}

func TestUnknownToolStopsAfterLimit(t *testing.T) {
	model := &scriptedModel{steps: []modelStep{
		calls("", call("1", "missing", `{}`)),
		calls("", call("2", "missing", `{}`)),
	}}
	result := testRuntime(t, model, nil, calculator.New()).Run(context.Background(), "unknown")
	if result.Stop != StopUnknownTool {
		t.Fatalf("stop = %s, want %s", result.Stop, StopUnknownTool)
	}
}

func TestRepeatedFailedCallStops(t *testing.T) {
	failing := stubTool{name: "always_fails", execute: func(context.Context, json.RawMessage) tool.Result {
		return tool.Failure("failed", "no", false)
	}}
	model := &scriptedModel{steps: []modelStep{
		calls("", call("1", failing.name, `{}`)),
		calls("", call("2", failing.name, `{}`)),
		calls("", call("3", failing.name, `{}`)),
	}}
	result := testRuntime(t, model, nil, failing).Run(context.Background(), "fail")
	if result.Stop != StopRepeatedFailure {
		t.Fatalf("stop = %s, want %s", result.Stop, StopRepeatedFailure)
	}
}

func TestMaxRounds(t *testing.T) {
	success := stubTool{name: "ok", execute: func(context.Context, json.RawMessage) tool.Result {
		return tool.Success(map[string]any{"ok": true}, "ok")
	}}
	model := &scriptedModel{steps: []modelStep{
		calls("", call("1", success.name, `{}`)), calls("", call("2", success.name, `{}`)),
		calls("", call("3", success.name, `{}`)), calls("", call("4", success.name, `{}`)),
	}}
	result := testRuntime(t, model, nil, success).Run(context.Background(), "loop")
	if result.Stop != StopMaxRounds || result.Rounds != 4 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestContextCancellationStopsTool(t *testing.T) {
	started := make(chan struct{})
	blocking := stubTool{name: "blocking", execute: func(ctx context.Context, _ json.RawMessage) tool.Result {
		close(started)
		<-ctx.Done()
		return tool.Failure("canceled", ctx.Err().Error(), true)
	}}
	model := &scriptedModel{steps: []modelStep{calls("", call("1", blocking.name, `{}`))}}
	runtime := testRuntime(t, model, nil, blocking)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan Result, 1)
	go func() { done <- runtime.Run(ctx, "cancel") }()
	<-started
	cancel()
	select {
	case result := <-done:
		if result.Stop != StopContextCanceled {
			t.Fatalf("stop = %s, want %s", result.Stop, StopContextCanceled)
		}
	case <-time.After(time.Second):
		t.Fatal("runtime did not observe cancellation")
	}
}

func TestToolTimeoutReturnedToModel(t *testing.T) {
	blocking := stubTool{name: "slow", execute: func(ctx context.Context, _ json.RawMessage) tool.Result {
		<-ctx.Done()
		return tool.Failure("timeout", ctx.Err().Error(), true)
	}}
	model := &scriptedModel{steps: []modelStep{calls("", call("1", blocking.name, `{}`)), answer("fallback")}}
	result := testRuntime(t, model, nil, blocking).Run(context.Background(), "slow")
	if result.Stop != StopFinalAnswer || result.Answer != "fallback" || !requestContains(model.requests[1], `"code":"timeout"`) {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestTemporaryModelErrorIsRetried(t *testing.T) {
	model := &scriptedModel{steps: []modelStep{
		{err: &llm.Error{Kind: llm.ErrorTemporary, Retryable: true, Err: errors.New("rate limited")}},
		answer("recovered"),
	}}
	result := testRuntime(t, model, nil, calculator.New()).Run(context.Background(), "retry")
	if result.Stop != StopFinalAnswer || result.Answer != "recovered" || len(model.requests) != 2 {
		t.Fatalf("unexpected result: %+v requests=%d", result, len(model.requests))
	}
}

func TestLargeToolResultIsTruncatedAndAudited(t *testing.T) {
	large := stubTool{name: "large", execute: func(context.Context, json.RawMessage) tool.Result {
		return tool.Success(map[string]any{"text": strings.Repeat("x", 4000)}, "large")
	}}
	audits := &captureAudit{}
	model := &scriptedModel{steps: []modelStep{calls("PRIVATE_REASONING", call("1", large.name, `{}`)), answer("done")}}
	result := testRuntime(t, model, audits, large).Run(context.Background(), "large")
	if result.Stop != StopFinalAnswer || !requestContains(model.requests[1], `"truncated":true`) {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(audits.entries) != 1 || !audits.entries[0].WasTruncated {
		t.Fatalf("missing truncation audit: %#v", audits.entries)
	}
	publicTrace, _ := json.Marshal(result.Trace)
	if strings.Contains(string(publicTrace), "PRIVATE_REASONING") || strings.Contains(string(publicTrace), "You are a tool-using") {
		t.Fatalf("trace leaked private model/system content: %s", publicTrace)
	}
}

func requestContains(request llm.Request, substring string) bool {
	for _, message := range request.Messages {
		if strings.Contains(message.Content, substring) {
			return true
		}
	}
	return false
}

type stubTool struct {
	name    string
	execute func(context.Context, json.RawMessage) tool.Result
}

func (s stubTool) Definition() tool.Definition {
	return tool.Definition{Name: s.name, Description: "test tool", Type: tool.TypeRead, Schema: map[string]any{
		"type": "object", "properties": map[string]any{}, "additionalProperties": false,
	}}
}

func (s stubTool) Execute(ctx context.Context, raw json.RawMessage) tool.Result {
	return s.execute(ctx, raw)
}

type captureAudit struct {
	entries []audit.Entry
}

func (a *captureAudit) Log(_ context.Context, entry audit.Entry) {
	a.entries = append(a.entries, entry)
}
