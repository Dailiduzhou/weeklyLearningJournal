package postgres

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	runtimeTool "toolcall/internal/tool"
)

func TestWhitelistAndParameterValidationBeforeDatabase(t *testing.T) {
	query := QueryDefinition{Name: "by_id", SQL: "SELECT id FROM items WHERE id=$1", Params: []ParamDefinition{{Name: "id", Type: "integer", Required: true, Minimum: floatPtr(1)}}}
	db := New(nil, []QueryDefinition{query}, time.Second, 10, 1024)
	registry, err := runtimeTool.NewRegistry(db)
	if err != nil {
		t.Fatalf("compile generated query schema: %v", err)
	}
	if err := registry.Validate("database_query", json.RawMessage(`{"query":"by_id","params":{"id":1}}`)); err != nil {
		t.Fatalf("valid generated schema rejected: %v", err)
	}
	if err := registry.Validate("database_query", json.RawMessage(`{"query":"by_id","params":{"id":0}}`)); err == nil {
		t.Fatal("generated schema accepted an out-of-range parameter")
	}

	denied := db.Execute(context.Background(), json.RawMessage(`{"query":"DROP TABLE items","params":{}}`))
	if denied.OK || denied.Error.Code != "query_not_allowed" {
		t.Fatalf("non-whitelisted query was not denied: %+v", denied)
	}
	invalid := db.Execute(context.Background(), json.RawMessage(`{"query":"by_id","params":{"id":0}}`))
	if invalid.OK || invalid.Error.Code != "invalid_arguments" {
		t.Fatalf("invalid parameter was not denied: %+v", invalid)
	}
	unavailable := db.Execute(context.Background(), json.RawMessage(`{"query":"by_id","params":{"id":1}}`))
	if unavailable.OK || unavailable.Error.Code != "database_unavailable" {
		t.Fatalf("expected database unavailable: %+v", unavailable)
	}
}

func TestTypedNilPoolIsTreatedAsUnavailable(t *testing.T) {
	var pool *pgxpool.Pool
	query := QueryDefinition{Name: "by_id", SQL: "SELECT id FROM items WHERE id=$1", Params: []ParamDefinition{{Name: "id", Type: "integer", Required: true, Minimum: floatPtr(1)}}}
	db := New(pool, []QueryDefinition{query}, time.Second, 10, 1024)
	result := db.Execute(context.Background(), json.RawMessage(`{"query":"by_id","params":{"id":1}}`))
	if result.OK || result.Error == nil || result.Error.Code != "database_unavailable" {
		t.Fatalf("expected database_unavailable instead of panic, got %+v", result)
	}
}

func floatPtr(value float64) *float64 { return &value }

func TestIntegrationQueryTimeout(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	db := New(pool, []QueryDefinition{{Name: "slow", SQL: "SELECT pg_sleep(0.2)"}}, 10*time.Millisecond, 10, 1024)
	result := db.Execute(ctx, json.RawMessage(`{"query":"slow","params":{}}`))
	if result.OK || result.Error == nil || result.Error.Code != "timeout" {
		t.Fatalf("expected timeout, got %+v", result)
	}
}
