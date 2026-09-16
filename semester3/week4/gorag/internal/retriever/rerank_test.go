package retriever

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/cloudwego/eino/components/model"
	einoretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"

	"gorag/internal/document"
)

type rerankModelFunc func(context.Context, []*schema.Message) (*schema.Message, error)

func (f rerankModelFunc) Generate(ctx context.Context, messages []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	return f(ctx, messages)
}
func (f rerankModelFunc) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("not implemented")
}

type rerankChildFunc func(context.Context, string, ...einoretriever.Option) ([]*schema.Document, error)

func (f rerankChildFunc) Retrieve(ctx context.Context, query string, opts ...einoretriever.Option) ([]*schema.Document, error) {
	return f(ctx, query, opts...)
}

func rerankTestConfig() RerankConfig {
	return RerankConfig{Timeout: time.Second, MaxInputChars: 24000, MaxConcurrency: 1}
}

func rerankTestDocuments() []*schema.Document {
	a := childDocument("17", 0, 0.9)
	b := childDocument("42", 0, 0.6)
	a.MetaData[MetadataDocumentTitle] = "事务"
	a.MetaData[MetadataHeadingPath] = []string{"事务", "边界"}
	return []*schema.Document{a, b}
}

func TestLLMRerankReordersClonesPreservesScoresAndForwardsOptions(t *testing.T) {
	docs := rerankTestDocuments()
	filter := document.MetadataFilter{Metadata: map[string]string{"category": "go"}, Tags: []string{"backend"}}
	ctx := context.WithValue(context.Background(), struct{}{}, "request")
	child := rerankChildFunc(func(gotCtx context.Context, query string, opts ...einoretriever.Option) ([]*schema.Document, error) {
		gotFilter, err := FilterFromOptions(opts...)
		if gotCtx != ctx || query != "事务边界？" || err != nil || !reflect.DeepEqual(gotFilter, filter) {
			t.Fatalf("query/context/filter not forwarded: %q %#v %v", query, gotFilter, err)
		}
		common := einoretriever.GetCommonOptions(&einoretriever.Options{}, opts...)
		if common.TopK == nil || *common.TopK != 9 {
			t.Fatal("TopK not forwarded")
		}
		return docs, nil
	})
	var calls int
	llm := rerankModelFunc(func(_ context.Context, messages []*schema.Message) (*schema.Message, error) {
		calls++
		if len(messages) != 2 || messages[0].Role != schema.System || !strings.Contains(messages[0].Content, "untrusted") {
			t.Fatalf("messages = %#v", messages)
		}
		for _, want := range []string{"事务边界？", "事务", "边界", `"id":"C1"`, docs[0].Content, docs[1].Content} {
			if !strings.Contains(messages[1].Content, want) {
				t.Fatalf("prompt missing %q", want)
			}
		}
		return schema.AssistantMessage(`{"ranking":["C2","C1"]}`, nil), nil
	})
	r, err := NewLLMRerankRetriever(child, llm, rerankTestConfig(), nil)
	if err != nil {
		t.Fatal(err)
	}
	ranked, err := r.Retrieve(ctx, "事务边界？", WithFilter(filter), einoretriever.WithTopK(9))
	if err != nil || calls != 1 || len(ranked) != 2 {
		t.Fatalf("Retrieve = %v, %v calls=%d", ranked, err, calls)
	}
	for i, original := range []*schema.Document{docs[1], docs[0]} {
		got := ranked[i]
		position, ok := RerankPosition(got)
		if got == original || got.ID != original.ID || got.Content != original.Content || got.Score() != original.Score() || !ok || position != i+1 {
			t.Fatalf("reranked document lost identity/score/rank: %#v", got)
		}
		if _, exists := original.MetaData[MetadataRerankPosition]; exists {
			t.Fatal("original metadata mutated")
		}
		delete(got.MetaData, MetadataRerankPosition)
		if !reflect.DeepEqual(got.MetaData, original.MetaData) {
			t.Fatal("provenance changed")
		}
	}
}

func TestParseRerankOrderStrictProtocol(t *testing.T) {
	valid, err := parseRerankOrder(" \n{\"ranking\":[\"C2\",\"C1\"]}\n", 2)
	if err != nil || !reflect.DeepEqual(valid, []int{1, 0}) {
		t.Fatalf("valid order = %v, %v", valid, err)
	}
	for _, raw := range []string{
		"", "null", "[]", "{}", `{"ranking":null}`, `{"ranking":[]}`,
		`{"ranking":["C1"]}`, `{"ranking":["C1","C1"]}`, `{"ranking":["C1","C3"]}`,
		`{"ranking":["C1","C2","C3"]}`, `{"ranking":["C01","C2"]}`, `{"ranking":["C+1","C2"]}`,
		`{"ranking":["C0","C2"]}`, `{"ranking":["c1","C2"]}`, `{"ranking":[1,2]}`,
		`{"ranking":["C1",null]}`, `{"ranking":["C1","C2"],"score":1}`,
		`{"ranking":["C1","C2"],"ranking":["C2","C1"]}`, `{"Ranking":["C1","C2"]}`,
		`{"ranking":["C1","C2"]} {}`, `{"ranking":["C1","C2"]} explanation`,
		"```json\n{\"ranking\":[\"C1\",\"C2\"]}\n```", "<think>ok</think>{\"ranking\":[\"C1\",\"C2\"]}",
		strings.Repeat(" ", 64<<10) + `{"ranking":["C1","C2"]}`,
	} {
		t.Run(raw[:min(len(raw), 90)], func(t *testing.T) {
			if order, err := parseRerankOrder(raw, 2); err == nil || order != nil {
				t.Fatalf("accepted %q: %v", raw, order)
			}
		})
	}
}

func TestLLMRerankFailuresReturnExactOriginalAndSafeLogs(t *testing.T) {
	for _, test := range []struct {
		name, reason string
		model        rerankModelFunc
	}{
		{"model error", "model_error", func(context.Context, []*schema.Message) (*schema.Message, error) {
			return nil, errors.New("secret-api-key private-query private-content")
		}},
		{"nil response", "invalid_response", func(context.Context, []*schema.Message) (*schema.Message, error) { return nil, nil }},
		{"invalid response", "invalid_response", func(context.Context, []*schema.Message) (*schema.Message, error) {
			return schema.AssistantMessage("private-output", nil), nil
		}},
		{"timeout", "timeout", func(ctx context.Context, _ []*schema.Message) (*schema.Message, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}},
		{"late success", "timeout", func(ctx context.Context, _ []*schema.Message) (*schema.Message, error) {
			<-ctx.Done()
			return schema.AssistantMessage(`{"ranking":["C2","C1"]}`, nil), nil
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			docs := rerankTestDocuments()
			docs[0].Content = "private-content"
			var logs bytes.Buffer
			cfg := rerankTestConfig()
			cfg.Timeout = 5 * time.Millisecond
			var calls int
			llm := rerankModelFunc(func(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
				calls++
				return test.model(ctx, messages)
			})
			r, err := NewLLMRerankRetriever(&stubChildRetriever{documents: docs}, llm, cfg, slog.New(slog.NewJSONHandler(&logs, nil)))
			if err != nil {
				t.Fatal(err)
			}
			for attempt := 0; attempt < 2; attempt++ {
				got, err := r.Retrieve(context.Background(), "private-query")
				if err != nil || len(got) != len(docs) || got[0] != docs[0] || got[1] != docs[1] {
					t.Fatalf("fallback changed input: %v %v", got, err)
				}
			}
			if calls != 2 {
				t.Fatalf("calls = %d; retried or leaked slot", calls)
			}
			if !strings.Contains(logs.String(), `"reason":"`+test.reason+`"`) {
				t.Fatal(logs.String())
			}
			for _, secret := range []string{"secret-api-key", "private-query", "private-content", "private-output"} {
				if strings.Contains(logs.String(), secret) {
					t.Fatalf("logs leaked %s", secret)
				}
			}
		})
	}
}

func TestLLMRerankUnicodeBudgetBoundaryAndNoTruncation(t *testing.T) {
	docs := rerankTestDocuments()
	docs[0].Content = strings.Repeat("中文", 20)
	messages, err := rerankMessages("问题", docs)
	if err != nil {
		t.Fatal(err)
	}
	chars := 0
	for _, message := range messages {
		chars += utf8.RuneCountInString(message.Content)
	}
	for _, budget := range []int{chars - 1, chars, chars + 1} {
		cfg := rerankTestConfig()
		cfg.MaxInputChars = budget
		calls := 0
		llm := rerankModelFunc(func(_ context.Context, actual []*schema.Message) (*schema.Message, error) {
			calls++
			if !reflect.DeepEqual(actual, messages) {
				t.Fatal("input truncated or changed")
			}
			return schema.AssistantMessage(`{"ranking":["C2","C1"]}`, nil), nil
		})
		r, err := NewLLMRerankRetriever(&stubChildRetriever{documents: docs}, llm, cfg, nil)
		if err != nil {
			t.Fatal(err)
		}
		got, err := r.Retrieve(context.Background(), "问题")
		if err != nil {
			t.Fatal(err)
		}
		if budget < chars {
			if calls != 0 || got[0] != docs[0] {
				t.Fatal("over-budget request called model or changed input")
			}
		} else if calls != 1 || got[0].ID != docs[1].ID {
			t.Fatal("within-budget request skipped")
		}
	}
}

func TestLLMRerankSkipsSmallOrMalformedCandidateSets(t *testing.T) {
	docs := rerankTestDocuments()
	for _, input := range [][]*schema.Document{nil, {}, {docs[0]}, {nil, docs[0]}, {docs[0], docs[0]}, {docs[0], {ID: "empty"}}} {
		llm := rerankModelFunc(func(context.Context, []*schema.Message) (*schema.Message, error) {
			t.Fatal("model must not run")
			return nil, nil
		})
		r, err := NewLLMRerankRetriever(&stubChildRetriever{documents: input}, llm, rerankTestConfig(), nil)
		if err != nil {
			t.Fatal(err)
		}
		got, err := r.Retrieve(context.Background(), "query")
		if err != nil || !reflect.DeepEqual(got, input) {
			t.Fatalf("got %v, %v", got, err)
		}
	}
}

func TestLLMRerankPropagatesChildFailureAndRequestCancellation(t *testing.T) {
	failure := errors.New("retrieval failed")
	child := &stubChildRetriever{err: failure}
	calls := 0
	llm := rerankModelFunc(func(ctx context.Context, _ []*schema.Message) (*schema.Message, error) {
		calls++
		<-ctx.Done()
		return nil, ctx.Err()
	})
	r, err := NewLLMRerankRetriever(child, llm, rerankTestConfig(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Retrieve(context.Background(), "query"); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.Retrieve(ctx, "query"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("model called before valid retrieval")
	}
	child.err, child.documents = nil, rerankTestDocuments()
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	got, err := r.Retrieve(ctx, "query")
	if !errors.Is(err, context.DeadlineExceeded) || got != nil {
		t.Fatalf("request cancellation fell back: %v %v", got, err)
	}
}

func TestLLMRerankConcurrencyIsNonblockingAndSlotReleasedOnCancellation(t *testing.T) {
	docs := rerankTestDocuments()
	entered := make(chan struct{}, 2)
	var calls atomic.Int32
	llm := rerankModelFunc(func(ctx context.Context, _ []*schema.Message) (*schema.Message, error) {
		calls.Add(1)
		entered <- struct{}{}
		<-ctx.Done()
		return nil, ctx.Err()
	})
	r, err := NewLLMRerankRetriever(&stubChildRetriever{documents: docs}, llm, rerankTestConfig(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { _, err := r.Retrieve(ctx, "first"); done <- err }()
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			cancel()
			t.Fatal("slot was not available")
		}
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				got, err := r.Retrieve(ctx, "concurrent")
				if err != nil || len(got) != 2 || got[0] != docs[0] {
					t.Error("concurrent request queued or changed input")
				}
			}()
		}
		wg.Wait()
		if calls.Load() != int32(attempt+1) {
			t.Fatal("exceeded concurrency limit")
		}
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}
}

func TestNewLLMRerankRetrieverValidation(t *testing.T) {
	child := &stubChildRetriever{}
	llm := rerankModelFunc(func(context.Context, []*schema.Message) (*schema.Message, error) { return nil, nil })
	if _, err := NewLLMRerankRetriever(nil, llm, rerankTestConfig(), nil); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	if _, err := NewLLMRerankRetriever(child, nil, rerankTestConfig(), nil); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	for _, cfg := range []RerankConfig{{}, {Timeout: -1, MaxInputChars: 1, MaxConcurrency: 1}, {Timeout: 1, MaxInputChars: 0, MaxConcurrency: 1}, {Timeout: 1, MaxInputChars: 1, MaxConcurrency: 0}} {
		if _, err := NewLLMRerankRetriever(child, llm, cfg, nil); !errors.Is(err, ErrInvalidConfig) {
			t.Fatal(err)
		}
	}
}
