// Package loader recursively discovers and loads Markdown and text files.
package loader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"gorag/internal/document"
)

var (
	ErrEmpty           = errors.New("document is empty")
	ErrTooLarge        = errors.New("document exceeds size limit")
	ErrInvalidUTF8     = errors.New("document is not valid UTF-8")
	ErrUnsupportedType = errors.New("unsupported document type")
	ErrOutsideRoot     = errors.New("document path is outside docs root")
)

// Loader applies a fixed per-file size limit. The limit is measured in bytes.
type Loader struct {
	maxFileSize int64
}

func New(maxFileSize int64) (*Loader, error) {
	if maxFileSize <= 0 {
		return nil, fmt.Errorf("max file size must be positive: %d", maxFileSize)
	}
	return &Loader{maxFileSize: maxFileSize}, nil
}

// Scan recursively loads supported, valid files. Unsupported, empty, over-limit,
// and invalid UTF-8 files are skipped; filesystem and context errors are returned.
func (l *Loader) Scan(ctx context.Context, root string) ([]document.Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve docs root: %w", err)
	}

	var documents []document.Document
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() || !supportedExtension(path) {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("make %q relative to docs root: %w", path, err)
		}
		doc, err := l.Load(ctx, root, relative)
		if err != nil {
			if errors.Is(err, ErrEmpty) || errors.Is(err, ErrTooLarge) || errors.Is(err, ErrInvalidUTF8) || errors.Is(err, ErrUnsupportedType) {
				return nil
			}
			return err
		}
		documents = append(documents, doc)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan docs root %q: %w", root, err)
	}

	sort.Slice(documents, func(i, j int) bool {
		return documents[i].SourcePath < documents[j].SourcePath
	})
	return documents, nil
}

// Load validates and reads one path relative to root.
func (l *Loader) Load(ctx context.Context, root, relativePath string) (document.Document, error) {
	if err := ctx.Err(); err != nil {
		return document.Document{}, err
	}
	if !supportedExtension(relativePath) {
		return document.Document{}, fmt.Errorf("load %q: %w", relativePath, ErrUnsupportedType)
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return document.Document{}, fmt.Errorf("resolve docs root: %w", err)
	}
	candidateAbs, err := filepath.Abs(filepath.Join(rootAbs, filepath.FromSlash(relativePath)))
	if err != nil {
		return document.Document{}, fmt.Errorf("resolve document %q: %w", relativePath, err)
	}
	rel, err := filepath.Rel(rootAbs, candidateAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return document.Document{}, fmt.Errorf("load %q: %w", relativePath, ErrOutsideRoot)
	}

	file, err := os.Open(candidateAbs)
	if err != nil {
		return document.Document{}, fmt.Errorf("open document %q: %w", filepath.ToSlash(rel), err)
	}
	defer file.Close()

	contents, err := readLimited(ctx, file, l.maxFileSize)
	if err != nil {
		return document.Document{}, fmt.Errorf("read document %q: %w", filepath.ToSlash(rel), err)
	}
	if !utf8.Valid(contents) {
		return document.Document{}, fmt.Errorf("load %q: %w", filepath.ToSlash(rel), ErrInvalidUTF8)
	}
	text := string(contents)
	if strings.TrimSpace(strings.TrimPrefix(text, "\ufeff")) == "" {
		return document.Document{}, fmt.Errorf("load %q: %w", filepath.ToSlash(rel), ErrEmpty)
	}

	sourcePath := filepath.ToSlash(rel)
	contentSum := sha256.Sum256(contents)
	idSum := sha256.Sum256([]byte(sourcePath))
	kind := document.KindText
	if strings.EqualFold(filepath.Ext(sourcePath), ".md") {
		kind = document.KindMarkdown
	}

	return document.Document{
		ID:          hex.EncodeToString(idSum[:]),
		SourcePath:  sourcePath,
		Title:       discoverTitle(text, sourcePath, kind),
		Kind:        kind,
		Content:     text,
		ContentHash: hex.EncodeToString(contentSum[:]),
		Size:        int64(len(contents)),
	}, nil
}

// Scan is a convenience wrapper around Loader.Scan.
func Scan(ctx context.Context, root string, maxFileSize int64) ([]document.Document, error) {
	l, err := New(maxFileSize)
	if err != nil {
		return nil, err
	}
	return l.Scan(ctx, root)
}

// Load is a convenience wrapper around Loader.Load.
func Load(ctx context.Context, root, relativePath string, maxFileSize int64) (document.Document, error) {
	l, err := New(maxFileSize)
	if err != nil {
		return document.Document{}, err
	}
	return l.Load(ctx, root, relativePath)
}

func readLimited(ctx context.Context, reader io.Reader, maximum int64) ([]byte, error) {
	buffer := make([]byte, 32*1024)
	contents := make([]byte, 0, minInt64(maximum, int64(len(buffer))))
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		count, err := reader.Read(buffer)
		if count > 0 {
			total += int64(count)
			if total > maximum {
				return nil, ErrTooLarge
			}
			contents = append(contents, buffer[:count]...)
		}
		if errors.Is(err, io.EOF) {
			return contents, nil
		}
		if err != nil {
			return nil, err
		}
	}
}

func supportedExtension(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".txt":
		return true
	default:
		return false
	}
}

func discoverTitle(content, sourcePath string, kind document.Kind) string {
	if kind == document.KindMarkdown {
		normalized := strings.ReplaceAll(strings.ReplaceAll(strings.TrimPrefix(content, "\ufeff"), "\r\n", "\n"), "\r", "\n")
		for _, line := range strings.Split(normalized, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "# ") {
				if title := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(trimmed[2:]), "#")); title != "" {
					return title
				}
			}
		}
	}
	base := filepath.Base(sourcePath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func minInt64(a, b int64) int {
	if a < b {
		return int(a)
	}
	return int(b)
}
