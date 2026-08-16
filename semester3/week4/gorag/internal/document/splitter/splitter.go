// Package splitter converts cleaned documents into deterministic, ordered
// chunks while retaining source structure and line metadata.
package splitter

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"gorag/internal/document"
)

var (
	ErrEmptyDocument = errors.New("cannot split an empty document")
	ErrUnknownKind   = errors.New("cannot split unknown document kind")
)

// Config sizes are Unicode character counts, not byte counts.
type Config struct {
	TargetSize  int
	MaxSize     int
	OverlapSize int
	MinSize     int
}

func DefaultConfig() Config {
	return Config{
		TargetSize:  1000,
		MaxSize:     1600,
		OverlapSize: 120,
		MinSize:     100,
	}
}

func (config Config) Validate() error {
	if config.MinSize <= 0 {
		return fmt.Errorf("minimum size must be positive: %d", config.MinSize)
	}
	if config.TargetSize < config.MinSize {
		return fmt.Errorf("target size %d is smaller than minimum size %d", config.TargetSize, config.MinSize)
	}
	if config.MaxSize < config.TargetSize {
		return fmt.Errorf("maximum size %d is smaller than target size %d", config.MaxSize, config.TargetSize)
	}
	if config.OverlapSize < 0 || config.OverlapSize >= config.TargetSize {
		return fmt.Errorf("overlap size %d must be non-negative and smaller than target size %d", config.OverlapSize, config.TargetSize)
	}
	return nil
}

type Splitter struct {
	config Config
}

func New(config Config) (*Splitter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Splitter{config: config}, nil
}

// Split divides a cleaned document. Markdown sections are never merged across
// heading boundaries; overlap is used only between chunks of the same section.
func (splitter *Splitter) Split(doc document.Document) ([]document.Chunk, error) {
	if strings.TrimSpace(doc.Content) == "" {
		return nil, ErrEmptyDocument
	}
	if !utf8.ValidString(doc.Content) {
		return nil, fmt.Errorf("split %q: invalid UTF-8", doc.SourcePath)
	}

	runes := []rune(doc.Content)
	lineStarts := runeLineStarts(runes)
	var sections []section
	switch doc.Kind {
	case document.KindMarkdown:
		sections = markdownSections(runes)
	case document.KindText:
		sections = []section{{start: 0, end: len(runes)}}
	default:
		return nil, fmt.Errorf("split %q: %w %q", doc.SourcePath, ErrUnknownKind, doc.Kind)
	}

	chunks := make([]document.Chunk, 0, len(sections))
	for _, current := range sections {
		if allWhitespace(runes[current.start:current.end]) {
			continue
		}
		codeRanges := []span(nil)
		if doc.Kind == document.KindMarkdown {
			codeRanges = fencedCodeRanges(runes, current.start, current.end)
		}
		for _, part := range splitter.splitSection(runes, current, doc.Kind, codeRanges) {
			start, end := trimSpan(runes, part.start, part.end)
			if start >= end {
				continue
			}
			content := string(runes[start:end])
			hash := sha256.Sum256([]byte(content))
			chunks = append(chunks, document.Chunk{
				DocumentID:      doc.ID,
				SourcePath:      doc.SourcePath,
				DocumentTitle:   doc.Title,
				HeadingPath:     append([]string(nil), current.headingPath...),
				Index:           len(chunks),
				Content:         content,
				StartLine:       originalLineAt(lineStarts, doc.LineNumbers, start),
				EndLine:         originalLineAt(lineStarts, doc.LineNumbers, end-1),
				ContentHash:     hex.EncodeToString(hash[:]),
				DocumentVersion: doc.Version,
			})
		}
	}
	return chunks, nil
}

// Split is a convenience wrapper around New and Splitter.Split.
func Split(doc document.Document, config Config) ([]document.Chunk, error) {
	splitter, err := New(config)
	if err != nil {
		return nil, err
	}
	return splitter.Split(doc)
}

type section struct {
	start       int
	end         int
	headingPath []string
}

type span struct {
	start int
	end   int
}

type lineSpan struct {
	start int
	end   int
}

func markdownSections(content []rune) []section {
	lines := linesWithEnd(content, 0, len(content))
	sections := make([]section, 0)
	path := make([]string, 0, 6)
	current := section{start: 0}
	haveCurrent := false
	inFence := false
	fence := ""

	for _, line := range lines {
		lineText := strings.TrimSpace(string(content[line.start:line.end]))
		if marker := fenceMarker(lineText); marker != "" {
			if !inFence {
				inFence = true
				fence = marker
			} else if strings.HasPrefix(lineText, fence) {
				inFence = false
				fence = ""
			}
			continue
		}
		if inFence {
			continue
		}
		level, title, ok := parseHeading(lineText)
		if !ok {
			continue
		}
		if !haveCurrent && line.start > 0 && !allWhitespace(content[:line.start]) {
			sections = append(sections, section{start: 0, end: line.start})
		}
		if haveCurrent && !allWhitespace(content[current.start:line.start]) {
			current.end = line.start
			sections = append(sections, current)
		}
		if level <= len(path) {
			path = path[:level-1]
		}
		for len(path) < level-1 {
			path = append(path, "")
		}
		path = append(path, title)
		current = section{start: line.start, headingPath: compactPath(path)}
		haveCurrent = true
	}

	if !haveCurrent {
		return []section{{start: 0, end: len(content)}}
	}
	current.end = len(content)
	if !allWhitespace(content[current.start:current.end]) {
		sections = append(sections, current)
	}
	return sections
}

func (splitter *Splitter) splitSection(content []rune, current section, kind document.Kind, codeRanges []span) []span {
	position := current.start
	parts := make([]span, 0, 1)
	for position < current.end {
		if current.end-position <= splitter.config.MaxSize {
			parts = append(parts, span{start: position, end: current.end})
			break
		}
		end := chooseEnd(content, position, current.end, splitter.config, kind, codeRanges)
		if end <= position {
			end = min(position+splitter.config.MaxSize, current.end)
		}
		parts = append(parts, span{start: position, end: end})
		if end == current.end {
			break
		}
		next := end - splitter.config.OverlapSize
		if next <= position {
			next = end
		}
		if insideRange(next, codeRanges) {
			// Avoid beginning an overlap in a code block when the whole block fit
			// in the previous chunk. Oversized blocks still require a hard split.
			if code := containingRange(next, codeRanges); code.end <= end {
				next = code.end
			}
		}
		position = next
	}
	return parts
}

func chooseEnd(content []rune, start, sectionEnd int, config Config, kind document.Kind, codeRanges []span) int {
	preferred := min(start+config.TargetSize, sectionEnd)
	limit := min(start+config.MaxSize, sectionEnd)
	minimum := min(start+config.MinSize, preferred)

	separators := [][]rune{[]rune("\n\n"), []rune("\n"), []rune("。"), []rune("？"), []rune("！"), []rune("."), []rune("?"), []rune("!")}
	if kind == document.KindMarkdown {
		// Markdown uses the same hierarchy, but fenced ranges are excluded below.
		separators = append(separators, []rune(" "))
	}
	for _, separator := range separators {
		positions := boundaryPositions(content, minimum, limit, separator)
		if len(positions) == 0 {
			continue
		}
		bestBefore := -1
		bestAfter := -1
		for _, position := range positions {
			if kind == document.KindMarkdown && insideRange(position, codeRanges) {
				continue
			}
			if position <= preferred {
				bestBefore = position
			} else if bestAfter < 0 {
				bestAfter = position
			}
		}
		if bestBefore > start {
			return bestBefore
		}
		if bestAfter > start {
			return bestAfter
		}
	}
	return preferred
}

func boundaryPositions(content []rune, start, end int, separator []rune) []int {
	if len(separator) == 0 || start >= end {
		return nil
	}
	positions := make([]int, 0)
	for index := start; index+len(separator) <= end; index++ {
		match := true
		for offset := range separator {
			if content[index+offset] != separator[offset] {
				match = false
				break
			}
		}
		if match {
			positions = append(positions, index+len(separator))
			index += len(separator) - 1
		}
	}
	return positions
}

func fencedCodeRanges(content []rune, start, end int) []span {
	lines := linesWithEnd(content, start, end)
	ranges := make([]span, 0)
	open := -1
	fence := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(string(content[line.start:line.end]))
		marker := fenceMarker(trimmed)
		if open < 0 && marker != "" {
			open = line.start
			fence = marker
			continue
		}
		if open >= 0 && strings.HasPrefix(trimmed, fence) {
			ranges = append(ranges, span{start: open, end: line.end})
			open = -1
			fence = ""
		}
	}
	if open >= 0 {
		ranges = append(ranges, span{start: open, end: end})
	}
	return ranges
}

func linesWithEnd(content []rune, start, end int) []lineSpan {
	lines := make([]lineSpan, 0, 1)
	lineStart := start
	for index := start; index < end; index++ {
		if content[index] == '\n' {
			lines = append(lines, lineSpan{start: lineStart, end: index + 1})
			lineStart = index + 1
		}
	}
	if lineStart < end || len(lines) == 0 {
		lines = append(lines, lineSpan{start: lineStart, end: end})
	}
	return lines
}

func parseHeading(line string) (int, string, bool) {
	if line == "" || line[0] != '#' {
		return 0, "", false
	}
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level == 0 || level > 6 || level >= len(line) || (line[level] != ' ' && line[level] != '\t') {
		return 0, "", false
	}
	title := strings.TrimSpace(line[level:])
	title = strings.TrimSpace(strings.TrimRight(title, "#"))
	if title == "" {
		return 0, "", false
	}
	return level, title, true
}

func compactPath(path []string) []string {
	result := make([]string, 0, len(path))
	for _, item := range path {
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func fenceMarker(line string) string {
	if strings.HasPrefix(line, "```") {
		return "```"
	}
	if strings.HasPrefix(line, "~~~") {
		return "~~~"
	}
	return ""
}

func trimSpan(content []rune, start, end int) (int, int) {
	for start < end && unicode.IsSpace(content[start]) {
		start++
	}
	for end > start && unicode.IsSpace(content[end-1]) {
		end--
	}
	return start, end
}

func allWhitespace(content []rune) bool {
	for _, current := range content {
		if !unicode.IsSpace(current) {
			return false
		}
	}
	return true
}

// runeLineStarts returns the rune offset where each line begins. Line indices
// are 0-based. A trailing newline does not create an empty trailing line.
func runeLineStarts(content []rune) []int {
	starts := []int{0}
	for index, current := range content {
		if current == '\n' && index+1 < len(content) {
			starts = append(starts, index+1)
		}
	}
	return starts
}

// originalLineAt maps a rune position in cleaned content to the 1-based line
// number in the original source document. When doc.LineNumbers is missing or
// shorter than the cleaned content, it falls back to cleaned-content line
// numbers, preserving the previous behavior for callers that do not clean.
func originalLineAt(lineStarts []int, originalLines []int, position int) int {
	if position < 0 {
		position = 0
	}
	lineIndex := sort.Search(len(lineStarts), func(index int) bool {
		return lineStarts[index] > position
	}) - 1
	lineIndex = max(lineIndex, 0)
	if lineIndex < len(originalLines) && originalLines[lineIndex] > 0 {
		return originalLines[lineIndex]
	}
	return lineIndex + 1
}

func insideRange(position int, ranges []span) bool {
	for _, current := range ranges {
		if current.start < position && position < current.end {
			return true
		}
	}
	return false
}

func containingRange(position int, ranges []span) span {
	for _, current := range ranges {
		if current.start < position && position < current.end {
			return current
		}
	}
	return span{}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
