package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

const maxRetries = 3

var errInvalidResponse = errors.New("model output does not match Response")

type Response struct {
	Title    string   `json:"title"`
	Summary  string   `json:"summary"`
	Priority string   `json:"priority"`
	Tags     []string `json:"tags"`
}

var responseSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"title":    map[string]any{"type": "string"},
		"summary":  map[string]any{"type": "string"},
		"priority": map[string]any{"type": "string"},
		"tags": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string"},
		},
	},
	"required":             []string{"title", "summary", "priority", "tags"},
	"additionalProperties": false,
}

func main() {
	options, err := clientOptions()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	client := openai.NewClient(options...)
	err = interactiveLoop(
		context.Background(),
		os.Stdin,
		os.Stdout,
		os.Stderr,
		func(ctx context.Context, prompt string, output io.Writer) error {
			return createResponse(ctx, &client, prompt, output)
		},
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type promptHandler func(context.Context, string, io.Writer) error

func interactiveLoop(
	ctx context.Context,
	input io.Reader,
	output io.Writer,
	diagnostics io.Writer,
	handle promptHandler,
) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	for {
		fmt.Fprint(diagnostics, "> ")
		if !scanner.Scan() {
			return scanner.Err()
		}

		prompt := strings.TrimSpace(scanner.Text())
		if prompt == "" {
			continue
		}
		if strings.EqualFold(prompt, "quit") || strings.EqualFold(prompt, "exit") {
			return nil
		}

		if err := handle(ctx, prompt, output); err != nil {
			// Invalid model output must not be emitted, but it should not end the loop.
			if errors.Is(err, errInvalidResponse) {
				continue
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			fmt.Fprintln(diagnostics, err)
		}
	}
}

func createResponse(ctx context.Context, client *openai.Client, userPrompt string, output io.Writer) error {
	schemaJSON, err := json.Marshal(responseSchema)
	if err != nil {
		return fmt.Errorf("marshal response schema: %w", err)
	}
	systemPrompt := "Return only one JSON object that exactly matches the following JSON Schema. Do not use Markdown or add explanatory text.\nJSON Schema:\n" + string(schemaJSON)

	completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: "deepseek-v4-flash",
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
		// Temporarily disabled: enforce the schema through response_format.
		// ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
		// 	OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
		// 		JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
		// 			Name:   "response",
		// 			Schema: responseSchema,
		// 			Strict: openai.Bool(true),
		// 		},
		// 	},
		// },
	})
	if err != nil {
		return fmt.Errorf("create chat completion: %w", err)
	}
	if len(completion.Choices) == 0 || completion.Choices[0].Message.Refusal != "" {
		return errInvalidResponse
	}

	result, ok := decodeResponse(completion.Choices[0].Message.Content)
	if !ok {
		return errInvalidResponse
	}

	if err := json.NewEncoder(output).Encode(result); err != nil {
		return fmt.Errorf("write response: %w", err)
	}
	return nil
}

func clientOptions() ([]option.RequestOption, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_APIKEY")
	}
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY is required")
	}

	options := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithMaxRetries(maxRetries),
	}

	// Keep compatibility with the original environment variable while also letting
	// the SDK read its standard OPENAI_BASE_URL variable itself.
	if baseURL := os.Getenv("OPENAI_BASEURL"); baseURL != "" {
		options = append(options, option.WithBaseURL(baseURL))
	}

	return options, nil
}

func decodeResponse(raw string) (Response, bool) {
	// Pointer fields distinguish required values from missing fields and JSON null.
	// Pointer elements also reject null entries in tags.
	var decoded struct {
		Title    *string    `json:"title"`
		Summary  *string    `json:"summary"`
		Priority *string    `json:"priority"`
		Tags     *[]*string `json:"tags"`
	}

	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return Response{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Response{}, false
	}
	if decoded.Title == nil || decoded.Summary == nil || decoded.Priority == nil || decoded.Tags == nil {
		return Response{}, false
	}

	tags := make([]string, len(*decoded.Tags))
	for i, tag := range *decoded.Tags {
		if tag == nil {
			return Response{}, false
		}
		tags[i] = *tag
	}

	return Response{
		Title:    *decoded.Title,
		Summary:  *decoded.Summary,
		Priority: *decoded.Priority,
		Tags:     tags,
	}, true
}
