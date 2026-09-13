package document

import (
	"fmt"
	"strings"
	"testing"
)

func TestMetadataFilterEmptyAndValid(t *testing.T) {
	t.Parallel()

	empty := MetadataFilter{}
	if !empty.Empty() {
		t.Fatal("zero filter should be empty")
	}
	if err := empty.Validate(); err != nil {
		t.Fatalf("Validate(empty) error = %v", err)
	}

	valid := MetadataFilter{
		Metadata: map[string]string{"category": "ccnubox", "module": "bff"},
		Tags:     []string{"ccnubox/bff", "ccnubox/crypto"},
	}
	if valid.Empty() {
		t.Fatal("populated filter should not be empty")
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}

	tagsOnly := MetadataFilter{Tags: []string{"ccnubox"}}
	if tagsOnly.Empty() {
		t.Fatal("tags-only filter should not be empty")
	}
	if err := tagsOnly.Validate(); err != nil {
		t.Fatalf("Validate(tagsOnly) error = %v", err)
	}
}

func TestMetadataFilterValidateRejectsMalformedInput(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name       string
		filter     MetadataFilter
		wantPhrase string
	}{
		{name: "empty metadata key", filter: MetadataFilter{Metadata: map[string]string{" ": "v"}}, wantPhrase: "empty"},
		{name: "empty metadata value", filter: MetadataFilter{Metadata: map[string]string{"category": " "}}, wantPhrase: "empty value"},
		{name: "empty tag", filter: MetadataFilter{Tags: []string{"  "}}, wantPhrase: "tag is empty"},
		{name: "too many metadata keys", filter: MetadataFilter{Metadata: mapOf(17, "v")}, wantPhrase: "limit"},
		{name: "too many tags", filter: MetadataFilter{Tags: tagsOf(33)}, wantPhrase: "limit"},
		{name: "oversized value", filter: MetadataFilter{Metadata: map[string]string{"category": strings.Repeat("x", 257)}}, wantPhrase: "exceeds"},
		{name: "oversized key", filter: MetadataFilter{Metadata: map[string]string{strings.Repeat("k", 129): "v"}}, wantPhrase: "exceeds"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.filter.Validate()
			if err == nil || !strings.Contains(err.Error(), testCase.wantPhrase) {
				t.Fatalf("Validate(%s) error = %v, want phrase %q", testCase.name, err, testCase.wantPhrase)
			}
		})
	}
}

func TestDocumentMetadataTags(t *testing.T) {
	t.Parallel()

	none := DocumentMetadata{}
	if none.Tags() != nil {
		t.Fatalf("Tags() = %#v, want nil without a tags list", none.Tags())
	}
	tagged := DocumentMetadata{Lists: map[string][]string{
		TagsKey: {"ccnubox", "ccnubox/bff"},
		"other": {"unrelated"},
	}}
	if tags := tagged.Tags(); len(tags) != 2 || tags[0] != "ccnubox" || tags[1] != "ccnubox/bff" {
		t.Fatalf("Tags() = %#v, want only the tags list", tags)
	}
}

func TestDocumentMetadataValidate(t *testing.T) {
	t.Parallel()

	valid := DocumentMetadata{
		Scalars: map[string]string{"category": "ccnubox", "type": "optimization", "status": "done"},
		Lists:   map[string][]string{TagsKey: {"ccnubox", "ccnubox/bff"}},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}
	// Scalar values may be empty, matching the cleaner's bare-key behavior.
	bareKey := DocumentMetadata{Scalars: map[string]string{"status": ""}}
	if err := bareKey.Validate(); err != nil {
		t.Fatalf("Validate(bare key) error = %v", err)
	}

	for _, testCase := range []struct {
		name       string
		metadata   DocumentMetadata
		wantPhrase string
	}{
		{name: "empty key", metadata: DocumentMetadata{Scalars: map[string]string{"": "v"}}, wantPhrase: "empty"},
		{name: "empty list item", metadata: DocumentMetadata{Lists: map[string][]string{TagsKey: {"ccnubox", " "}}}, wantPhrase: "empty item"},
		{name: "too many scalar keys", metadata: DocumentMetadata{Scalars: mapOf(33, "v")}, wantPhrase: "exceed limit"},
		{name: "too many list keys", metadata: DocumentMetadata{Lists: listsOf(17)}, wantPhrase: "exceed limit"},
		{name: "too many list items", metadata: DocumentMetadata{Lists: map[string][]string{TagsKey: tagsOf(129)}}, wantPhrase: "limit"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.metadata.Validate()
			if err == nil || !strings.Contains(err.Error(), testCase.wantPhrase) {
				t.Fatalf("Validate(%s) error = %v, want phrase %q", testCase.name, err, testCase.wantPhrase)
			}
		})
	}
}

func mapOf(count int, value string) map[string]string {
	result := make(map[string]string, count)
	for index := range count {
		result[fmt.Sprintf("key%02d", index)] = value
	}
	return result
}

func listsOf(count int) map[string][]string {
	result := make(map[string][]string, count)
	for index := range count {
		result[fmt.Sprintf("list%02d", index)] = []string{"item"}
	}
	return result
}

func tagsOf(count int) []string {
	tags := make([]string, count)
	for index := range tags {
		tags[index] = "tag"
	}
	return tags
}
