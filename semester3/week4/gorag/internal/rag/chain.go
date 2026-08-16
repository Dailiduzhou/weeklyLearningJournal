package rag

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	einoretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	promptQuestionKey = "question"
	promptContextKey  = "context"
)

var (
	// ErrInvalidChain identifies missing dependencies or an empty question.
	ErrInvalidChain = errors.New("rag: invalid chain")
	// ErrGeneration identifies ChatTemplate or ChatModel execution failures.
	ErrGeneration = errors.New("rag: generation failed")
)

// Result is the hand-off to the answer/citation layer. It deliberately keeps
// the raw model message separate from request-scoped source provenance.
type Result struct {
	Message         *schema.Message
	Sources         []Source
	SourceDocuments map[string]*schema.Document
}

// Chain owns the online RAG orchestration boundary. Retrieval and context
// selection run before the compiled Eino ChatTemplate -> ChatModel sub-chain,
// so an empty or weak result set cannot reach the model.
type Chain struct {
	retriever einoretriever.Retriever
	builder   *ContextBuilder
	generator compose.Runnable[map[string]any, *schema.Message]
}

// NewChain compiles the Eino ChatTemplate -> ChatModel portion of the online
// chain. Passing nil template uses the knowledge-grounded default template.
func NewChain(ctx context.Context, retriever einoretriever.Retriever, builder *ContextBuilder, template prompt.ChatTemplate, chatModel model.BaseChatModel) (*Chain, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if retriever == nil {
		return nil, fmt.Errorf("%w: retriever is nil", ErrInvalidChain)
	}
	if builder == nil {
		return nil, fmt.Errorf("%w: context builder is nil", ErrInvalidChain)
	}
	if chatModel == nil {
		return nil, fmt.Errorf("%w: chat model is nil", ErrInvalidChain)
	}
	if template == nil {
		template = DefaultChatTemplate()
	}

	generation := compose.NewChain[map[string]any, *schema.Message]()
	generation.AppendChatTemplate(template).AppendChatModel(chatModel)
	runnable, err := generation.Compile(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: compile Eino generation chain: %w", ErrInvalidChain, err)
	}
	return &Chain{retriever: retriever, builder: builder, generator: runnable}, nil
}

// Invoke runs retrieval, context building, prompt formatting, and model
// generation with the same Context. No-context outcomes stop before the model.
func (c *Chain) Invoke(ctx context.Context, question string) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	question = strings.TrimSpace(question)
	if question == "" {
		return Result{}, fmt.Errorf("%w: question is empty", ErrInvalidChain)
	}

	documents, err := c.retriever.Retrieve(ctx, question)
	if err != nil {
		return Result{}, err
	}
	if len(documents) == 0 {
		return Result{}, ErrInsufficientContext
	}
	built, err := c.builder.Build(ctx, documents)
	if err != nil {
		return Result{}, err
	}
	message, err := c.generator.Invoke(ctx, map[string]any{
		promptQuestionKey: question,
		promptContextKey:  built.Text,
	})
	if err != nil {
		return Result{}, fmt.Errorf("%w: %w", ErrGeneration, err)
	}
	if message == nil {
		return Result{}, fmt.Errorf("%w: chat model returned a nil message", ErrGeneration)
	}
	return Result{
		Message:         message,
		Sources:         append([]Source(nil), built.Sources...),
		SourceDocuments: built.DocumentsBySource,
	}, nil
}

// DefaultChatTemplate constrains the model to the supplied context and stable
// S-number citations. Citation syntax validation belongs to work package 50.
func DefaultChatTemplate() prompt.ChatTemplate {
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage("You are a grounded backend-learning knowledge-base assistant. Answer only from the retrieval context supplied in this request; never add facts from model knowledge. Cite every material claim with its source ID, such as [S1]. If the context does not directly support an answer, reply with exactly INSUFFICIENT_CONTEXT and nothing else."),
		schema.UserMessage("Question: {question}\n\nRetrieval context:\n{context}\n\nAnswer using only this context and include source citations."),
	)
}
