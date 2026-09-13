package repository

import (
	"strings"
	"testing"

	"gorag/internal/document"
	"gorag/internal/embedding"
)

func TestValidateSourcePath(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		path    string
		wantErr bool
	}{
		{path: "api/auth.md"},
		{path: "README.txt"},
		{path: "../secret.md", wantErr: true},
		{path: "/absolute.md", wantErr: true},
		{path: "api\\auth.md", wantErr: true},
		{path: "api/../auth.md", wantErr: true},
		{path: "", wantErr: true},
	} {
		t.Run(testCase.path, func(t *testing.T) {
			err := validateSourcePath(testCase.path)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("validateSourcePath(%q) error = %v, wantErr %v", testCase.path, err, testCase.wantErr)
			}
		})
	}
}

func TestValidateVersionChunk(t *testing.T) {
	t.Parallel()

	valid := VersionChunk{
		Chunk: document.Chunk{
			Index:              0,
			Content:            "body",
			StartLine:          1,
			EndLine:            2,
			ContentHash:        "hash",
			DocumentVersion:    "v1",
			EmbeddingModel:     embedding.DefaultModel,
			EmbeddingDimension: embedding.VectorDimension,
		},
		Embedding: make([]float32, embedding.VectorDimension),
	}
	if err := validateVersionChunk(valid, "v1"); err != nil {
		t.Fatalf("validateVersionChunk(valid) error = %v", err)
	}

	wrongDimension := valid
	wrongDimension.Embedding = wrongDimension.Embedding[:10]
	if err := validateVersionChunk(wrongDimension, "v1"); err == nil {
		t.Fatal("validateVersionChunk(wrongDimension) error = nil")
	}
	wrongModel := valid
	wrongModel.EmbeddingModel = "other"
	if err := validateVersionChunk(wrongModel, "v1"); err == nil {
		t.Fatal("validateVersionChunk(wrongModel) error = nil")
	}
	wrongVersion := valid
	wrongVersion.DocumentVersion = "v2"
	if err := validateVersionChunk(wrongVersion, "v1"); err == nil {
		t.Fatal("validateVersionChunk(wrongVersion) error = nil")
	}
}

func TestValidateActivation(t *testing.T) {
	t.Parallel()

	valid := Activation{
		DocumentID: 1, Version: "v1", ExpectedChunkCount: 1,
		Title: "Authentication", ContentHash: "hash",
		Parent: ActivationParent{Content: "whole document", StartLine: 1, EndLine: 9},
	}
	if err := validateActivation(valid); err != nil {
		t.Fatalf("validateActivation(valid) error = %v", err)
	}

	for _, testCase := range []struct {
		name       string
		mutate     func(*Activation)
		wantPhrase string
	}{
		{name: "document id", mutate: func(a *Activation) { a.DocumentID = 0 }, wantPhrase: "document ID"},
		{name: "version", mutate: func(a *Activation) { a.Version = " " }, wantPhrase: "document ID and version"},
		{name: "chunk count", mutate: func(a *Activation) { a.ExpectedChunkCount = 0 }, wantPhrase: "chunk count"},
		{name: "title", mutate: func(a *Activation) { a.Title = "" }, wantPhrase: "title"},
		{name: "content hash", mutate: func(a *Activation) { a.ContentHash = "" }, wantPhrase: "title and content hash"},
		{name: "parent content", mutate: func(a *Activation) { a.Parent.Content = "  " }, wantPhrase: "parent document content"},
		{name: "parent start line", mutate: func(a *Activation) { a.Parent.StartLine = 0 }, wantPhrase: "line range"},
		{name: "parent end line", mutate: func(a *Activation) { a.Parent.EndLine = 0 }, wantPhrase: "line range"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			invalid := valid
			testCase.mutate(&invalid)
			err := validateActivation(invalid)
			if err == nil || !strings.Contains(err.Error(), testCase.wantPhrase) {
				t.Fatalf("validateActivation(%s) error = %v, want phrase %q", testCase.name, err, testCase.wantPhrase)
			}
		})
	}
}

func TestCanonicalParentIDs(t *testing.T) {
	t.Parallel()

	if canonical, err := canonicalParentIDs(nil); err != nil || len(canonical) != 0 {
		t.Fatalf("canonicalParentIDs(nil) = %v, error %v, want empty", canonical, err)
	}
	if canonical, err := canonicalParentIDs([]int64{3, 1, 3, 1}); err != nil || len(canonical) != 2 || canonical[0] != 3 || canonical[1] != 1 {
		t.Fatalf("canonicalParentIDs(duplicates) = %v, error %v, want first-seen order [3, 1]", canonical, err)
	}
	if _, err := canonicalParentIDs([]int64{17, 0}); err == nil {
		t.Fatal("canonicalParentIDs(zero id) error = nil")
	}
	if _, err := canonicalParentIDs([]int64{-1}); err == nil {
		t.Fatal("canonicalParentIDs(negative id) error = nil")
	}
	oversized := make([]int64, MaxParentBatchSize+1)
	if _, err := canonicalParentIDs(oversized); err == nil {
		t.Fatalf("canonicalParentIDs(%d ids) error = nil, want batch limit", len(oversized))
	}
	limit := make([]int64, MaxParentBatchSize)
	for index := range limit {
		limit[index] = int64(index + 1)
	}
	if canonical, err := canonicalParentIDs(limit); err != nil || len(canonical) != MaxParentBatchSize {
		t.Fatalf("canonicalParentIDs(limit) = %d ids, error %v, want exactly the limit", len(canonical), err)
	}
}
