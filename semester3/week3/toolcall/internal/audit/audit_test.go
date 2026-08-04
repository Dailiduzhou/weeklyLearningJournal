package audit

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactArguments(t *testing.T) {
	redacted := RedactArguments(json.RawMessage(`{"query":"safe","password":"p","nested":{"access_token":"t"}}`))
	b, err := json.Marshal(redacted)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if strings.Contains(text, `"p"`) || strings.Contains(text, `"t"`) || !strings.Contains(text, "[REDACTED]") || !strings.Contains(text, "safe") {
		t.Fatalf("redaction failed: %s", text)
	}
}
