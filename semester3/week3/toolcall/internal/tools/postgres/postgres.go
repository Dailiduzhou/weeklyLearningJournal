package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"toolcall/internal/tool"

	"github.com/jackc/pgx/v5"
)

type DB interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type ParamDefinition struct {
	Name      string
	Type      string
	Required  bool
	MaxLength int
	Minimum   *float64
	Maximum   *float64
}

type QueryDefinition struct {
	Name        string
	Description string
	SQL         string
	Params      []ParamDefinition
}

type Tool struct {
	db           DB
	queries      map[string]QueryDefinition
	queryTimeout time.Duration
	maxRows      int
	maxBytes     int
	schema       map[string]any
}

func New(db DB, queries []QueryDefinition, queryTimeout time.Duration, maxRows, maxBytes int) *Tool {
	byName := make(map[string]QueryDefinition, len(queries))
	for _, q := range queries {
		byName[q.Name] = q
	}
	return &Tool{
		db: db, queries: byName, queryTimeout: queryTimeout, maxRows: maxRows, maxBytes: maxBytes,
		schema: buildSchema(queries),
	}
}

func (t *Tool) Definition() tool.Definition {
	return tool.Definition{
		Name:        "database_query",
		Description: t.description(),
		Schema:      t.schema,
		Type:        tool.TypeRead,
	}
}

func (t *Tool) description() string {
	var b strings.Builder
	b.WriteString("Run one predefined read-only PostgreSQL query. SQL text is never accepted. Available queries: ")
	names := make([]string, 0, len(t.queries))
	for name := range t.queries {
		names = append(names, name)
	}
	sort.Strings(names)
	first := true
	for _, name := range names {
		q := t.queries[name]
		if !first {
			b.WriteString("; ")
		}
		first = false
		fmt.Fprintf(&b, "%s (%s)", q.Name, q.Description)
	}
	if first {
		b.WriteString("none configured")
	}
	return b.String()
}

func buildSchema(queries []QueryDefinition) map[string]any {
	if len(queries) == 0 {
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query":  map[string]any{"type": "string", "minLength": 1, "maxLength": 64},
				"params": map[string]any{"type": "object"},
			},
			"required": []string{"query", "params"}, "additionalProperties": false,
		}
	}
	variants := make([]any, 0, len(queries))
	for _, query := range queries {
		properties := make(map[string]any, len(query.Params))
		required := make([]string, 0, len(query.Params))
		for _, param := range query.Params {
			property := map[string]any{"type": param.Type}
			if param.MaxLength > 0 {
				property["maxLength"] = param.MaxLength
			}
			if param.Minimum != nil {
				property["minimum"] = *param.Minimum
			}
			if param.Maximum != nil {
				property["maximum"] = *param.Maximum
			}
			properties[param.Name] = property
			if param.Required {
				required = append(required, param.Name)
			}
		}
		paramSchema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
		if len(required) > 0 {
			paramSchema["required"] = required
		}
		variants = append(variants, map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query":  map[string]any{"const": query.Name},
				"params": paramSchema,
			},
			"required": []string{"query", "params"}, "additionalProperties": false,
		})
	}
	return map[string]any{"oneOf": variants}
}

func (t *Tool) Execute(ctx context.Context, raw json.RawMessage) tool.Result {
	var args struct {
		Query  string                     `json:"query"`
		Params map[string]json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return tool.Failure("invalid_arguments", err.Error(), false)
	}
	query, ok := t.queries[args.Query]
	if !ok {
		return tool.Failure("query_not_allowed", "query is not in the configured whitelist", false)
	}
	values, err := validateParams(query.Params, args.Params)
	if err != nil {
		return tool.Failure("invalid_arguments", err.Error(), false)
	}
	if t.db == nil {
		return tool.Failure("database_unavailable", "database is not enabled", true)
	}
	queryCtx, cancel := context.WithTimeout(ctx, t.queryTimeout)
	defer cancel()
	tx, err := t.db.BeginTx(queryCtx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return databaseFailure(queryCtx, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(queryCtx, query.SQL, values...)
	if err != nil {
		return databaseFailure(queryCtx, err)
	}
	defer rows.Close()
	fields := rows.FieldDescriptions()
	columns := make([]string, len(fields))
	for i, field := range fields {
		columns[i] = field.Name
	}

	resultRows := make([]map[string]any, 0, min(t.maxRows, 16))
	resultBytes := 0
	truncated := false
	for rows.Next() {
		if len(resultRows) >= t.maxRows {
			truncated = true
			break
		}
		values, err := rows.Values()
		if err != nil {
			return tool.Failure("database_error", "could not decode database row", false)
		}
		row := make(map[string]any, len(columns))
		for i, name := range columns {
			row[name] = values[i]
		}
		encoded, err := json.Marshal(row)
		if err != nil {
			return tool.Failure("database_error", "database row is not JSON encodable", false)
		}
		if resultBytes+len(encoded) > t.maxBytes {
			truncated = true
			break
		}
		resultBytes += len(encoded)
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		return databaseFailure(queryCtx, err)
	}
	result := tool.Success(map[string]any{"columns": columns, "rows": resultRows, "row_count": len(resultRows)}, fmt.Sprintf("returned %d rows", len(resultRows)))
	result.Truncated = truncated
	return result
}

func validateParams(defs []ParamDefinition, raw map[string]json.RawMessage) ([]any, error) {
	known := make(map[string]struct{}, len(defs))
	values := make([]any, 0, len(defs))
	for _, def := range defs {
		known[def.Name] = struct{}{}
		valueRaw, exists := raw[def.Name]
		if !exists {
			if def.Required {
				return nil, fmt.Errorf("missing required parameter %q", def.Name)
			}
			values = append(values, nil)
			continue
		}
		value, err := decodeParam(def, valueRaw)
		if err != nil {
			return nil, fmt.Errorf("parameter %q: %w", def.Name, err)
		}
		values = append(values, value)
	}
	for name := range raw {
		if _, ok := known[name]; !ok {
			return nil, fmt.Errorf("unknown parameter %q", name)
		}
	}
	return values, nil
}

func decodeParam(def ParamDefinition, raw json.RawMessage) (any, error) {
	switch def.Type {
	case "string":
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("must be a string")
		}
		if def.MaxLength > 0 && len([]rune(value)) > def.MaxLength {
			return nil, fmt.Errorf("exceeds maximum length %d", def.MaxLength)
		}
		return value, nil
	case "integer":
		var value int64
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("must be an integer")
		}
		if err := numericRange(float64(value), def); err != nil {
			return nil, err
		}
		return value, nil
	case "number":
		var value float64
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("must be a number")
		}
		if err := numericRange(value, def); err != nil {
			return nil, err
		}
		return value, nil
	case "boolean":
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("must be a boolean")
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported configured type")
	}
}

func numericRange(value float64, def ParamDefinition) error {
	if def.Minimum != nil && value < *def.Minimum {
		return fmt.Errorf("must be at least %v", *def.Minimum)
	}
	if def.Maximum != nil && value > *def.Maximum {
		return fmt.Errorf("must be at most %v", *def.Maximum)
	}
	return nil
}

func databaseFailure(ctx context.Context, err error) tool.Result {
	if ctx.Err() != nil {
		code := "canceled"
		if ctx.Err() == context.DeadlineExceeded {
			code = "timeout"
		}
		return tool.Failure(code, ctx.Err().Error(), true)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return tool.Success("No rows", "no expected records")
	}
	return tool.Failure("database_error", err.Error(), true)
}
