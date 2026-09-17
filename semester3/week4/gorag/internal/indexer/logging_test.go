package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"gorag/internal/document"
)

func TestIndexingLogsProgressAndOutcomes(t *testing.T) {
	var output bytes.Buffer
	doc := testDocument("guide.md", "v1")
	doc.Content = "# Private document body\n\nDo not log this content."
	loader := &fakeLoader{documents: []document.Document{doc}}
	embedder := &fakeEmbedder{}
	indexer, err := New(Config{DocsRoot: "docs"}, loader, fakeSplitter{}, embedder, newMemoryStore(),
		WithLogger(slog.New(slog.NewJSONHandler(&output, nil))))
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name   string
		force  bool
		fail   bool
		status string
	}{
		{name: "add", status: "added"},
		{name: "skip", status: "skipped"},
		{name: "reindex", force: true, status: "updated"},
		{name: "failure", force: true, fail: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			output.Reset()
			if test.fail {
				embedder.failPath = doc.SourcePath
			}
			var result Result
			var err error
			if test.force {
				result, err = indexer.ReindexFile(context.Background(), doc.SourcePath)
			} else {
				result, err = indexer.IndexFile(context.Background(), doc.SourcePath)
			}
			if (err != nil) != test.fail {
				t.Fatalf("index error = %v", err)
			}
			if strings.Contains(output.String(), "Private document body") || strings.Contains(output.String(), "Do not log this content") {
				t.Fatalf("logs contain document content: %s", &output)
			}
			entries := decodeIndexLogs(t, &output)
			requireIndexLog(t, entries, "index run started")
			requireIndexLog(t, entries, "document indexing started")
			if test.status != "skipped" {
				requireIndexLog(t, entries, "document split completed")
				requireIndexLog(t, entries, "document embedding started")
			}
			documentMessage, runMessage := "document indexing completed", "index run completed"
			if test.fail {
				documentMessage, runMessage = "document indexing failed", "index run failed"
			} else if test.status != "skipped" {
				requireIndexLog(t, entries, "document embedding completed")
				requireIndexLog(t, entries, "document storage started")
			}
			entry := requireIndexLog(t, entries, documentMessage)
			if entry["source_path"] != doc.SourcePath || entry["run_id"] != float64(result.RunID) || entry["operation"] != result.Operation || entry["duration"] == nil {
				t.Fatalf("missing document context: %#v", entry)
			}
			if test.fail {
				if entry["level"] != "ERROR" || !strings.Contains(entry["error"].(string), "embed document") {
					t.Fatalf("missing failure details: %#v", entry)
				}
			} else if entry["status"] != test.status || entry["chunks"] != float64(result.Chunks) {
				t.Fatalf("incorrect outcome: %#v", entry)
			}
			summary := requireIndexLog(t, entries, runMessage)
			if summary["documents"] != float64(result.Documents) || summary["failed"] != float64(result.Failed) || summary["duration"] == nil {
				t.Fatalf("incorrect summary: %#v", summary)
			}
		})
	}
}

func TestSyncLogsConcurrentProgress(t *testing.T) {
	var output bytes.Buffer
	loader := &fakeLoader{documents: []document.Document{testDocument("a.md", "a"), testDocument("b.md", "b")}}
	indexer, err := New(Config{DocsRoot: "docs", DocumentConcurrency: 2}, loader, fakeSplitter{}, &fakeEmbedder{}, newMemoryStore(),
		WithLogger(slog.New(slog.NewJSONHandler(&output, nil))))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := indexer.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	entries := decodeIndexLogs(t, &output)
	if scan := requireIndexLog(t, entries, "document scan completed"); scan["documents"] != float64(2) {
		t.Fatalf("incorrect scan count: %#v", scan)
	}
	paths := make(map[string]bool)
	for _, entry := range entries {
		if entry["msg"] == "document indexing completed" {
			paths[entry["source_path"].(string)] = true
		}
	}
	if !paths["a.md"] || !paths["b.md"] {
		t.Fatalf("missing concurrent progress: %#v", paths)
	}
}

func TestScanFailureIsLogged(t *testing.T) {
	var output bytes.Buffer
	indexer, err := New(Config{DocsRoot: "docs"}, &fakeLoader{scanErr: errors.New("scan unavailable")}, fakeSplitter{}, &fakeEmbedder{}, newMemoryStore(),
		WithLogger(nil), WithLogger(slog.New(slog.NewJSONHandler(&output, nil))))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := indexer.Sync(context.Background()); err == nil {
		t.Fatal("expected scan failure")
	}
	entry := requireIndexLog(t, decodeIndexLogs(t, &output), "index run failed")
	if entry["level"] != "ERROR" || !strings.Contains(entry["error"].(string), "scan docs root") {
		t.Fatalf("missing scan failure: %#v", entry)
	}
}

func decodeIndexLogs(t *testing.T, output *bytes.Buffer) []map[string]any {
	t.Helper()
	var entries []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("invalid log %q: %v", line, err)
		}
		entries = append(entries, entry)
	}
	return entries
}

func requireIndexLog(t *testing.T, entries []map[string]any, message string) map[string]any {
	t.Helper()
	for _, entry := range entries {
		if entry["msg"] == message {
			return entry
		}
	}
	t.Fatalf("missing log %q", message)
	return nil
}
