package document

import (
	"errors"
	"fmt"
	"strings"
)

// Bounded sizes keep front matter and filters from becoming a denial-of-service
// vector through the JSONB store, the lexical index, or the prompt pipeline.
const (
	maxMetadataKeys   = 32
	maxListKeys       = 16
	maxListItems      = 128
	maxKeyLength      = 128
	maxValueLength    = 512
	maxFilterKeys     = 16
	maxFilterTags     = 32
	maxFilterValueLen = 256
)

// Validate checks the parsed front matter before it is persisted. Keys and
// list items must be non-empty after trimming; scalar values may be empty,
// matching the cleaner's behavior for bare "key:" entries.
func (m DocumentMetadata) Validate() error {
	if len(m.Scalars) > maxMetadataKeys {
		return fmt.Errorf("document: %d scalar metadata keys exceed limit %d", len(m.Scalars), maxMetadataKeys)
	}
	if len(m.Lists) > maxListKeys {
		return fmt.Errorf("document: %d list metadata keys exceed limit %d", len(m.Lists), maxListKeys)
	}
	for key, value := range m.Scalars {
		if err := validateKey(key); err != nil {
			return err
		}
		if len(value) > maxValueLength {
			return fmt.Errorf("document: metadata key %q value exceeds %d characters", key, maxValueLength)
		}
	}
	for key, items := range m.Lists {
		if err := validateKey(key); err != nil {
			return err
		}
		if len(items) > maxListItems {
			return fmt.Errorf("document: metadata list %q has %d items, limit %d", key, len(items), maxListItems)
		}
		for _, item := range items {
			if strings.TrimSpace(item) == "" {
				return fmt.Errorf("document: metadata list %q has an empty item", key)
			}
			if len(item) > maxValueLength {
				return fmt.Errorf("document: metadata list %q item exceeds %d characters", key, maxValueLength)
			}
		}
	}
	return nil
}

// Validate rejects filters that cannot match anything meaningful: empty keys,
// empty values, or oversized constraints. Filters are user input, so bounds
// are tighter than storage limits.
func (f MetadataFilter) Validate() error {
	if len(f.Metadata) > maxFilterKeys {
		return fmt.Errorf("document: filter has %d metadata keys, limit %d", len(f.Metadata), maxFilterKeys)
	}
	for key, value := range f.Metadata {
		if err := validateKey(key); err != nil {
			return err
		}
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("document: filter metadata key %q has an empty value", key)
		}
		if len(value) > maxFilterValueLen {
			return fmt.Errorf("document: filter metadata key %q value exceeds %d characters", key, maxFilterValueLen)
		}
	}
	if len(f.Tags) > maxFilterTags {
		return fmt.Errorf("document: filter has %d tags, limit %d", len(f.Tags), maxFilterTags)
	}
	for _, tag := range f.Tags {
		if strings.TrimSpace(tag) == "" {
			return errors.New("document: filter tag is empty")
		}
		if len(tag) > maxFilterValueLen {
			return fmt.Errorf("document: filter tag %q exceeds %d characters", tag, maxFilterValueLen)
		}
	}
	return nil
}

func validateKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("document: metadata key is empty")
	}
	if len(key) > maxKeyLength {
		return fmt.Errorf("document: metadata key exceeds %d characters", maxKeyLength)
	}
	return nil
}
