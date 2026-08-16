package rag

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// RefusalAnswer is deliberately stable so every insufficient-context and
// dependency-failure path has the same product semantics.
const RefusalAnswer = "当前知识库中没有找到足够可靠的依据，暂时无法回答该问题。"

const modelRefusalToken = "INSUFFICIENT_CONTEXT"

var (
	validCitationPattern = regexp.MustCompile(`\[S[1-9][0-9]*\]`)
	// citationLikePattern also catches malformed attempts such as [S0], [Sx]
	// and [S 1]. A model cannot smuggle an unverified source through formatting.
	citationLikePattern = regexp.MustCompile(`(?i)\[\s*S[^\]]*\]`)
)

// AnswerSource is the public, provenance-only representation of a retrieved
// source. Document contents and internal identifiers are intentionally absent.
type AnswerSource struct {
	ID            string   `json:"id"`
	SourcePath    string   `json:"source_path"`
	DocumentTitle string   `json:"document_title"`
	HeadingPath   []string `json:"heading_path"`
	StartLine     int      `json:"start_line"`
	EndLine       int      `json:"end_line"`
	Similarity    float64  `json:"similarity"`
}

// Answer is shared by the answer layer and HTTP transport.
type Answer struct {
	Answerable bool           `json:"answerable"`
	Text       string         `json:"answer"`
	Sources    []AnswerSource `json:"sources"`
}

// Invoker is the narrow hand-off implemented by Chain.
type Invoker interface {
	Invoke(context.Context, string) (Result, error)
}

// AnswerService owns refusal normalization and citation validation. It does
// not know about HTTP status codes or JSON.
type AnswerService struct {
	chain Invoker
}

func NewAnswerService(chain Invoker) (*AnswerService, error) {
	if chain == nil {
		return nil, errors.New("rag: answer service chain is nil")
	}
	return &AnswerService{chain: chain}, nil
}

// AnswerQuestion always returns a safe product result. Internal dependency or
// model errors are returned alongside the refusal so callers can log them
// without exposing them. Context cancellation is preserved for transport.
func (s *AnswerService) AnswerQuestion(ctx context.Context, question string) (Answer, error) {
	if err := ctx.Err(); err != nil {
		return Answer{}, err
	}
	question = strings.TrimSpace(question)
	if question == "" {
		return Answer{}, errors.New("rag: question is empty")
	}
	if isUnderspecifiedQuestion(question) {
		return refusal(), nil
	}

	result, err := s.chain.Invoke(ctx, question)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Answer{}, ctxErr
		}
		if errors.Is(err, ErrInsufficientContext) {
			return refusal(), nil
		}
		return refusal(), fmt.Errorf("rag: answer question: %w", err)
	}
	answer, err := ValidateAnswer(result)
	if err != nil {
		// An ungrounded model response is a normal no-answer product result.
		return refusal(), nil
	}
	return answer, nil
}

// isUnderspecifiedQuestion rejects short pronoun-only questions whose subject
// cannot be resolved from this stateless request. Sending these to retrieval
// can produce a superficially similar chunk and an ungrounded answer.
func isUnderspecifiedQuestion(question string) bool {
	normalized := strings.TrimFunc(question, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})
	if utf8.RuneCountInString(normalized) > 8 {
		return false
	}
	for _, marker := range []string{"它", "这个", "那个", "这件事", "那件事", "该怎么", "怎么办"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

// ValidateAnswer accepts only exact [S1]-style citations from this request's
// source set. Sources are returned once, in first-citation order.
func ValidateAnswer(result Result) (Answer, error) {
	if result.Message == nil {
		return Answer{}, errors.New("rag: validate answer: model message is nil")
	}
	text := strings.TrimSpace(result.Message.Content)
	if text == "" {
		return Answer{}, errors.New("rag: validate answer: model answer is empty")
	}
	if modelSignalsInsufficientContext(text) {
		return Answer{}, errors.New("rag: validate answer: model reported insufficient context")
	}

	allowed := make(map[string]Source, len(result.Sources))
	for _, source := range result.Sources {
		if !validSourceID(source.ID) {
			return Answer{}, fmt.Errorf("rag: validate answer: invalid context source ID %q", source.ID)
		}
		if _, duplicate := allowed[source.ID]; duplicate {
			return Answer{}, fmt.Errorf("rag: validate answer: duplicate context source ID %q", source.ID)
		}
		if document, retrieved := result.SourceDocuments[source.ID]; !retrieved || document == nil {
			return Answer{}, fmt.Errorf("rag: validate answer: source %q has no retrieved document", source.ID)
		}
		allowed[source.ID] = source
	}

	allCitationLike := citationLikePattern.FindAllString(text, -1)
	validCitations := validCitationPattern.FindAllString(text, -1)
	if len(validCitations) == 0 {
		return Answer{}, errors.New("rag: validate answer: no valid citation")
	}
	if len(allCitationLike) != len(validCitations) {
		return Answer{}, errors.New("rag: validate answer: malformed citation")
	}

	seen := make(map[string]struct{}, len(validCitations))
	sources := make([]AnswerSource, 0, len(validCitations))
	for _, citation := range validCitations {
		id := citation[1 : len(citation)-1]
		source, ok := allowed[id]
		if !ok {
			return Answer{}, fmt.Errorf("rag: validate answer: unknown citation %q", id)
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		sources = append(sources, publicSource(source))
	}

	return Answer{Answerable: true, Text: text, Sources: sources}, nil
}

func modelSignalsInsufficientContext(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == strings.ToLower(modelRefusalToken) {
		return true
	}
	for _, phrase := range []string{
		"insufficient context",
		"insufficient information",
		"not enough information",
		"available material is insufficient",
		"资料不足",
		"材料不足",
		"没有足够",
		"无法回答",
		"不能回答",
	} {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}

func refusal() Answer {
	return Answer{Answerable: false, Text: RefusalAnswer, Sources: []AnswerSource{}}
}

func validSourceID(id string) bool {
	return len(id) > 1 && validCitationPattern.MatchString("["+id+"]") &&
		validCitationPattern.FindString("["+id+"]") == "["+id+"]"
}

func publicSource(source Source) AnswerSource {
	return AnswerSource{
		ID:            source.ID,
		SourcePath:    source.SourcePath,
		DocumentTitle: source.DocumentTitle,
		HeadingPath:   append([]string(nil), source.HeadingPath...),
		StartLine:     source.StartLine,
		EndLine:       source.EndLine,
		Similarity:    source.Similarity,
	}
}
