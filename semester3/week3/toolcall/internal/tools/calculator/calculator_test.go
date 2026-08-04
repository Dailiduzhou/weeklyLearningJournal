package calculator

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestCalculator(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expression string
		want       float64
	}{
		{"(17 + 5) * 3", 66},
		{"2^3^2", 512},
		{"-2 + 0.5", -1.5},
	}
	for _, tt := range tests {
		result := New().Execute(context.Background(), json.RawMessage(`{"expression":`+mustJSON(tt.expression)+`}`))
		if !result.OK {
			t.Fatalf("%q failed: %+v", tt.expression, result.Error)
		}
		value := result.Data.(map[string]any)["value"].(float64)
		if math.Abs(value-tt.want) > 1e-9 {
			t.Fatalf("%q = %v, want %v", tt.expression, value, tt.want)
		}
	}
}

func TestCalculatorRejectsUnsafeOrUnboundedInput(t *testing.T) {
	t.Parallel()
	for _, expression := range []string{"system(\"id\")", "1/0", "10^100", strings.Repeat("(", 40) + "1" + strings.Repeat(")", 40)} {
		result := New().Execute(context.Background(), json.RawMessage(`{"expression":`+mustJSON(expression)+`}`))
		if result.OK || result.Error == nil || result.Error.Code != "invalid_expression" {
			t.Fatalf("expected %q to be rejected, got %+v", expression, result)
		}
	}
}

func mustJSON(value string) string {
	b, _ := json.Marshal(value)
	return string(b)
}
