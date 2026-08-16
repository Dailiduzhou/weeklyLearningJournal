package loader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestScanLoadsSupportedDocumentsDeterministically(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "z.txt"), []byte("plain text"))
	mustWrite(t, filepath.Join(root, "nested", "a.md"), []byte("# Loaded title\r\n\r\nBody"))
	mustWrite(t, filepath.Join(root, "ignored.json"), []byte("{}"))
	mustWrite(t, filepath.Join(root, "empty.md"), nil)
	mustWrite(t, filepath.Join(root, "invalid.txt"), []byte{0xff, 0xfe})
	mustWrite(t, filepath.Join(root, "large.md"), []byte("1234567890123456789012345678901234567890"))

	documents, err := Scan(context.Background(), root, 32)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(documents) != 2 {
		t.Fatalf("Scan() got %d documents, want 2", len(documents))
	}
	if documents[0].SourcePath != "nested/a.md" || documents[1].SourcePath != "z.txt" {
		t.Fatalf("documents not sorted or relative: %#v", []string{documents[0].SourcePath, documents[1].SourcePath})
	}
	if documents[0].Title != "Loaded title" {
		t.Errorf("Markdown title = %q, want %q", documents[0].Title, "Loaded title")
	}
	expected := sha256.Sum256([]byte("# Loaded title\r\n\r\nBody"))
	if documents[0].ContentHash != hex.EncodeToString(expected[:]) {
		t.Errorf("content hash = %q, want %q", documents[0].ContentHash, hex.EncodeToString(expected[:]))
	}

	again, err := Scan(context.Background(), root, 32)
	if err != nil {
		t.Fatalf("second Scan() error = %v", err)
	}
	if again[0].ID != documents[0].ID || again[0].ContentHash != documents[0].ContentHash {
		t.Fatal("identical scan did not produce stable identifiers and hashes")
	}
}

func TestLoadRejectsInvalidInputsAndHonorsContext(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "empty.md"), []byte("\ufeff \r\n"))
	mustWrite(t, filepath.Join(root, "invalid.txt"), []byte{0xff})
	mustWrite(t, filepath.Join(root, "large.md"), []byte("12345"))
	mustWrite(t, filepath.Join(root, "other.bin"), []byte("value"))

	tests := []struct {
		name string
		path string
		max  int64
		want error
	}{
		{name: "empty", path: "empty.md", max: 32, want: ErrEmpty},
		{name: "invalid UTF-8", path: "invalid.txt", max: 32, want: ErrInvalidUTF8},
		{name: "too large", path: "large.md", max: 4, want: ErrTooLarge},
		{name: "unsupported", path: "other.bin", max: 32, want: ErrUnsupportedType},
		{name: "outside root", path: "../outside.md", max: 32, want: ErrOutsideRoot},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Load(context.Background(), root, test.path, test.max)
			if !errors.Is(err, test.want) {
				t.Fatalf("Load() error = %v, want %v", err, test.want)
			}
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Scan(ctx, root, 32)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Scan() canceled error = %v, want context.Canceled", err)
	}
}

func mustWrite(t *testing.T, path string, contents []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
