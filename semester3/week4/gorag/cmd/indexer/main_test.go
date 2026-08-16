package main

import (
	"context"
	"testing"

	indexerusecase "gorag/internal/indexer"
)

func TestParseArgsSupportsEveryCommand(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		args       []string
		command    string
		target     string
		configPath string
	}{
		{args: []string{"sync"}, command: "sync", configPath: "config.yaml"},
		{args: []string{"index", "api/auth.md"}, command: "index", target: "api/auth.md", configPath: "config.yaml"},
		{args: []string{"delete", "api/auth.md"}, command: "delete", target: "api/auth.md", configPath: "config.yaml"},
		{args: []string{"reindex", "api/auth.md"}, command: "reindex", target: "api/auth.md", configPath: "config.yaml"},
		{args: []string{"reindex-all", "--config", "local.yaml"}, command: "reindex-all", configPath: "local.yaml"},
	} {
		t.Run(testCase.command, func(t *testing.T) {
			options, err := parseArgs(testCase.args)
			if err != nil {
				t.Fatalf("parseArgs() error = %v", err)
			}
			if options.command != testCase.command || options.target != testCase.target || options.configPath != testCase.configPath {
				t.Fatalf("parseArgs() = %#v", options)
			}
		})
	}
}

func TestDispatchSupportsEveryCommand(t *testing.T) {
	t.Parallel()
	for _, command := range []string{"sync", "index", "delete", "reindex", "reindex-all"} {
		t.Run(command, func(t *testing.T) {
			fake := &fakeLifecycle{}
			result, err := dispatch(context.Background(), fake, cliOptions{command: command, target: "doc.md"})
			if err != nil || result.Operation != command || fake.called != command {
				t.Fatalf("dispatch(%q) = %#v, called %q, error %v", command, result, fake.called, err)
			}
		})
	}
}

func TestParseArgsRejectsInvalidArity(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{}, {"index"}, {"sync", "doc.md"}, {"unknown"}, {"--config"}} {
		if _, err := parseArgs(args); err == nil {
			t.Fatalf("parseArgs(%q) error = nil", args)
		}
	}
}

type fakeLifecycle struct{ called string }

func (f *fakeLifecycle) result(command string) (indexerusecase.Result, error) {
	f.called = command
	return indexerusecase.Result{Operation: command}, nil
}

func (f *fakeLifecycle) Sync(context.Context) (indexerusecase.Result, error) {
	return f.result("sync")
}
func (f *fakeLifecycle) IndexFile(context.Context, string) (indexerusecase.Result, error) {
	return f.result("index")
}
func (f *fakeLifecycle) DeleteFile(context.Context, string) (indexerusecase.Result, error) {
	return f.result("delete")
}
func (f *fakeLifecycle) ReindexFile(context.Context, string) (indexerusecase.Result, error) {
	return f.result("reindex")
}
func (f *fakeLifecycle) ReindexAll(context.Context) (indexerusecase.Result, error) {
	return f.result("reindex-all")
}
