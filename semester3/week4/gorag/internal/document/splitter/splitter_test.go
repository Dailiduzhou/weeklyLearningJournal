package splitter

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"gorag/internal/document"
)

func TestMarkdownPreservesDeepHeadingPathsAndLineNumbers(t *testing.T) {
	content := "Preface\n\n# Root\nroot body\n## Child\n- item\n| A | B |\n| - | - |\n### Grandchild\ndeep body"
	doc := document.Document{ID: "doc-1", SourcePath: "guide.md", Title: "Root", Kind: document.KindMarkdown, Content: content, Version: "v1"}
	chunks, err := Split(doc, Config{TargetSize: 100, MaxSize: 120, OverlapSize: 10, MinSize: 10})
	if err != nil {
		t.Fatalf("Split() error = %v", err)
	}
	if len(chunks) != 4 {
		t.Fatalf("Split() got %d chunks, want 4: %#v", len(chunks), chunks)
	}
	wantPaths := [][]string{nil, {"Root"}, {"Root", "Child"}, {"Root", "Child", "Grandchild"}}
	for index, chunk := range chunks {
		if !reflect.DeepEqual(chunk.HeadingPath, wantPaths[index]) {
			t.Errorf("chunk %d heading path = %#v, want %#v", index, chunk.HeadingPath, wantPaths[index])
		}
		if chunk.Index != index {
			t.Errorf("chunk index = %d, want %d", chunk.Index, index)
		}
		if chunk.DocumentVersion != "v1" || chunk.ContentHash == "" {
			t.Errorf("chunk metadata incomplete: %#v", chunk)
		}
	}
	if chunks[1].StartLine != 3 || chunks[1].EndLine != 4 {
		t.Errorf("root chunk lines = %d-%d, want 3-4", chunks[1].StartLine, chunks[1].EndLine)
	}
	if !strings.Contains(chunks[2].Content, "- item") || !strings.Contains(chunks[2].Content, "| A | B |") {
		t.Errorf("list or table semantics lost: %q", chunks[2].Content)
	}
}

func TestSplitUsesDocumentLineNumbers(t *testing.T) {
	doc := document.Document{
		ID: "doc", SourcePath: "guide.md", Title: "Root",
		Kind: document.KindMarkdown, Content: "# Root\nroot body",
		Version: "v1", LineNumbers: []int{10, 11},
	}
	chunks, err := Split(doc, Config{TargetSize: 100, MaxSize: 120, OverlapSize: 10, MinSize: 10})
	if err != nil {
		t.Fatalf("Split() error = %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("Split() got %d chunks, want 1", len(chunks))
	}
	if chunks[0].StartLine != 10 || chunks[0].EndLine != 11 {
		t.Fatalf("chunk lines = %d-%d, want 10-11", chunks[0].StartLine, chunks[0].EndLine)
	}
}

func TestMarkdownAvoidsSplittingFencedCodeBlock(t *testing.T) {
	code := "```go\nfunc main() {\n\tprintln(\"hello\")\n}\n```"
	content := "# Code\n" + strings.Repeat("intro sentence. ", 5) + "\n\n" + code + "\n\n" + strings.Repeat("tail sentence. ", 6)
	doc := document.Document{ID: "code", SourcePath: "code.md", Title: "Code", Kind: document.KindMarkdown, Content: content}
	chunks, err := Split(doc, Config{TargetSize: 80, MaxSize: 110, OverlapSize: 10, MinSize: 20})
	if err != nil {
		t.Fatalf("Split() error = %v", err)
	}
	found := false
	for _, chunk := range chunks {
		if utf8.RuneCountInString(chunk.Content) > 110 {
			t.Fatalf("chunk exceeds max: %d characters", utf8.RuneCountInString(chunk.Content))
		}
		if strings.Contains(chunk.Content, "```go") {
			found = true
			if !strings.Contains(chunk.Content, "\n```") {
				t.Fatalf("code block was split despite fitting max size: %q", chunk.Content)
			}
		}
	}
	if !found {
		t.Fatal("code block not found in chunks")
	}
}

func TestTextSplitsLongChineseAndEnglishAndIsStable(t *testing.T) {
	content := strings.Repeat("这是中文句子。还有问题？", 12) + "\n" + strings.Repeat("An English sentence. Another question? ", 10)
	doc := document.Document{ID: "text", SourcePath: "mixed.txt", Title: "mixed", Kind: document.KindText, Content: content}
	config := Config{TargetSize: 70, MaxSize: 90, OverlapSize: 8, MinSize: 20}
	first, err := Split(doc, config)
	if err != nil {
		t.Fatalf("Split() error = %v", err)
	}
	second, err := Split(doc, config)
	if err != nil {
		t.Fatalf("second Split() error = %v", err)
	}
	if len(first) < 3 || !reflect.DeepEqual(first, second) {
		t.Fatalf("split is not sufficiently segmented or deterministic: first=%#v second=%#v", first, second)
	}
	for _, chunk := range first {
		if size := utf8.RuneCountInString(chunk.Content); size > config.MaxSize {
			t.Errorf("chunk size = %d, max = %d", size, config.MaxSize)
		}
	}
}

func TestBoundarySizesAndOversizedUnbrokenText(t *testing.T) {
	config := Config{TargetSize: 40, MaxSize: 80, OverlapSize: 5, MinSize: 10}
	exact := document.Document{Kind: document.KindText, Content: strings.Repeat("x", 80)}
	chunks, err := Split(exact, config)
	if err != nil || len(chunks) != 1 || utf8.RuneCountInString(chunks[0].Content) != 80 {
		t.Fatalf("exact max boundary: chunks=%#v err=%v", chunks, err)
	}

	over := document.Document{Kind: document.KindText, Content: strings.Repeat("界", 181)}
	chunks, err = Split(over, config)
	if err != nil {
		t.Fatalf("oversized Split() error = %v", err)
	}
	for _, chunk := range chunks {
		if size := utf8.RuneCountInString(chunk.Content); size > config.MaxSize {
			t.Errorf("fixed-length fallback produced %d characters", size)
		}
	}
}

func TestConfigAndEmptyInputValidation(t *testing.T) {
	if _, err := New(Config{TargetSize: 10, MaxSize: 9, MinSize: 1}); err == nil {
		t.Fatal("New() accepted max smaller than target")
	}
	_, err := Split(document.Document{Kind: document.KindText, Content: " \n"}, Config{TargetSize: 10, MaxSize: 20, MinSize: 1})
	if err != ErrEmptyDocument {
		t.Fatalf("empty Split() error = %v, want ErrEmptyDocument", err)
	}
}
