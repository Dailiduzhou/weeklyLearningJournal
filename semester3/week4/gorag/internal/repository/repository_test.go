package repository

import (
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
