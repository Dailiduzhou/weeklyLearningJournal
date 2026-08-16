// Package cleaner normalizes document text without changing Markdown syntax.
package cleaner

import (
	"strings"
	"unicode"
)

// Options controls optional cleanup behavior.
type Options struct {
	ParseFrontMatter bool
}

// Result contains normalized content, optionally parsed front matter, and a
// per-output-line mapping back to the original 1-based source line numbers.
// LineNumbers is empty when the cleaned content is empty.
type Result struct {
	Content     string
	FrontMatter map[string]string
	LineNumbers []int
}

// Clean removes transport-level noise while preserving headings, lists, tables,
// and fenced code blocks. It is deterministic and has no I/O side effects.
func Clean(content string, options Options) Result {
	content = strings.TrimPrefix(content, "\ufeff")
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	lines := splitNumberedLines(content)
	result := Result{}
	if options.ParseFrontMatter {
		frontMatter, end := parseFrontMatterLines(lines)
		if end >= 0 {
			result.FrontMatter = frontMatter
			lines = lines[end+1:]
		}
	}
	lines = removeHTMLCommentsLines(lines)
	lines = compressBlankLines(lines)
	lines = trimSurroundingBlankLines(lines)
	if len(lines) > 0 {
		// Match strings.TrimSpace on the joined content: only the very first
		// and very last characters are trimmed, internal indentation is kept.
		lines[0].text = strings.TrimLeftFunc(lines[0].text, unicode.IsSpace)
		lines[len(lines)-1].text = strings.TrimRightFunc(lines[len(lines)-1].text, unicode.IsSpace)
	}
	if len(lines) == 1 && strings.TrimSpace(lines[0].text) == "" {
		lines = nil
	}

	var builder strings.Builder
	result.LineNumbers = make([]int, 0, len(lines))
	for index, line := range lines {
		if index > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(line.text)
		result.LineNumbers = append(result.LineNumbers, line.originalLine)
	}
	result.Content = builder.String()
	return result
}

type numberedLine struct {
	text         string
	originalLine int
}

func splitNumberedLines(content string) []numberedLine {
	rawLines := strings.Split(content, "\n")
	lines := make([]numberedLine, len(rawLines))
	for index, line := range rawLines {
		lines[index] = numberedLine{text: line, originalLine: index + 1}
	}
	return lines
}

func parseFrontMatterLines(lines []numberedLine) (map[string]string, int) {
	if len(lines) < 3 || strings.TrimSpace(lines[0].text) != "---" {
		return nil, -1
	}
	end := -1
	for index := 1; index < len(lines); index++ {
		marker := strings.TrimSpace(lines[index].text)
		if marker == "---" || marker == "..." {
			end = index
			break
		}
	}
	if end < 0 {
		return nil, -1
	}
	metadata := make(map[string]string)
	for _, line := range lines[1:end] {
		key, value, ok := strings.Cut(line.text, ":")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			continue
		}
		metadata[key] = strings.Trim(strings.TrimSpace(value), "\"'")
	}
	if len(metadata) == 0 {
		metadata = nil
	}
	return metadata, end
}

func removeHTMLCommentsLines(lines []numberedLine) []numberedLine {
	inComment := false
	inFence := false
	fence := ""
	for index := range lines {
		trimmed := strings.TrimSpace(lines[index].text)
		if !inComment {
			if marker := fenceMarker(trimmed); marker != "" {
				if !inFence {
					inFence = true
					fence = marker
				} else if strings.HasPrefix(trimmed, fence) {
					inFence = false
					fence = ""
				}
			}
		}
		if inFence {
			continue
		}
		lines[index].text, inComment = stripCommentsFromLine(lines[index].text, inComment)
	}
	return lines
}

func stripCommentsFromLine(line string, inComment bool) (string, bool) {
	var output strings.Builder
	remaining := line
	for {
		if inComment {
			end := strings.Index(remaining, "-->")
			if end < 0 {
				return output.String(), true
			}
			remaining = remaining[end+3:]
			inComment = false
			continue
		}
		start := strings.Index(remaining, "<!--")
		if start < 0 {
			output.WriteString(remaining)
			return output.String(), false
		}
		output.WriteString(remaining[:start])
		remaining = remaining[start+4:]
		inComment = true
	}
}

func fenceMarker(trimmed string) string {
	if strings.HasPrefix(trimmed, "```") {
		return "```"
	}
	if strings.HasPrefix(trimmed, "~~~") {
		return "~~~"
	}
	return ""
}

func compressBlankLines(lines []numberedLine) []numberedLine {
	result := make([]numberedLine, 0, len(lines))
	previousBlank := false
	for _, line := range lines {
		blank := strings.TrimSpace(line.text) == ""
		if blank && previousBlank {
			continue
		}
		result = append(result, line)
		previousBlank = blank
	}
	return result
}

func trimSurroundingBlankLines(lines []numberedLine) []numberedLine {
	for len(lines) > 0 && strings.TrimSpace(lines[0].text) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1].text) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
