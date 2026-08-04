package tool

import (
	"context"
	"encoding/json"
	"testing"
)

type schemaTool struct{}

func (schemaTool) Definition() Definition {
	return Definition{Name: "schema_test", Description: "test", Type: TypeRead, Schema: map[string]any{
		"type": "object", "properties": map[string]any{"value": map[string]any{"type": "integer", "minimum": 1}},
		"required": []string{"value"}, "additionalProperties": false,
	}}
}
func (schemaTool) Execute(context.Context, json.RawMessage) Result { return Success(nil, "ok") }

func TestRegistryCompilesAndValidatesSchema(t *testing.T) {
	registry, err := NewRegistry(schemaTool{})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Validate("schema_test", json.RawMessage(`{"value":1}`)); err != nil {
		t.Fatalf("valid arguments rejected: %v", err)
	}
	for _, raw := range []string{`{"value":0}`, `{"value":1,"extra":true}`, `{broken`} {
		if err := registry.Validate("schema_test", json.RawMessage(raw)); err == nil {
			t.Fatalf("invalid arguments accepted: %s", raw)
		}
	}
}

func TestEncodeResultTruncatesAsValidJSON(t *testing.T) {
	encoded, result := EncodeResult(Success(map[string]any{"value": string(make([]byte, 2000))}, "large"), 512)
	if !result.Truncated || len(encoded) > 512 || !json.Valid(encoded) {
		t.Fatalf("invalid truncated result: truncated=%v bytes=%d valid=%v", result.Truncated, len(encoded), json.Valid(encoded))
	}
}
