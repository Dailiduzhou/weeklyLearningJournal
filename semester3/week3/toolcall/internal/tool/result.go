package tool

import (
	"encoding/json"
	"unicode/utf8"
)

func EncodeResult(result Result, maxBytes int) ([]byte, Result) {
	b, err := json.Marshal(result)
	if err != nil {
		result = Failure("result_encoding", "tool result could not be encoded", false)
		b, _ = json.Marshal(result)
	}
	result.Size = len(b)
	if len(b) <= maxBytes {
		b, _ = json.Marshal(result)
		return b, result
	}

	previewBytes := maxBytes / 3
	preview := utf8Prefix(string(b), previewBytes)
	result.Data = map[string]any{"preview": preview}
	result.Truncated = true
	result.Summary = "tool result exceeded the runtime size limit; preview returned"
	result.Size = len(b)
	b, _ = json.Marshal(result)
	if len(b) > maxBytes {
		result.Data = nil
		b, _ = json.Marshal(result)
	}
	return b, result
}

func utf8Prefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}
