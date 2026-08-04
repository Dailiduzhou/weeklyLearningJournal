package tool

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"toolcall/internal/llm"
)

var validName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

type registered struct {
	tool   Tool
	schema *jsonschema.Schema
}

type Registry struct {
	tools map[string]registered
}

func NewRegistry(tools ...Tool) (*Registry, error) {
	r := &Registry{tools: make(map[string]registered, len(tools))}
	for _, t := range tools {
		if err := r.Register(t); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Registry) Register(t Tool) error {
	if t == nil {
		return errors.New("register nil tool")
	}
	d := t.Definition()
	if !validName.MatchString(d.Name) || d.Description == "" || d.Schema == nil {
		return fmt.Errorf("invalid tool definition %q", d.Name)
	}
	if d.Type != TypeRead && d.Type != TypeWrite {
		return fmt.Errorf("invalid type for tool %q", d.Name)
	}
	if _, exists := r.tools[d.Name]; exists {
		return fmt.Errorf("duplicate tool %q", d.Name)
	}
	c := jsonschema.NewCompiler()
	url := "urn:tool:" + d.Name
	// Normalize Go-specific slices (for example []string in "required") into
	// JSON values before handing the document to the schema compiler.
	schemaJSON, err := json.Marshal(d.Schema)
	if err != nil {
		return fmt.Errorf("encode schema for %q: %w", d.Name, err)
	}
	var schemaDocument any
	if err := json.Unmarshal(schemaJSON, &schemaDocument); err != nil {
		return fmt.Errorf("decode schema for %q: %w", d.Name, err)
	}
	if err := c.AddResource(url, schemaDocument); err != nil {
		return fmt.Errorf("add schema for %q: %w", d.Name, err)
	}
	schema, err := c.Compile(url)
	if err != nil {
		return fmt.Errorf("compile schema for %q: %w", d.Name, err)
	}
	r.tools[d.Name] = registered{tool: t, schema: schema}
	return nil
}

func (r *Registry) Lookup(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t.tool, ok
}

func (r *Registry) Validate(name string, raw json.RawMessage) error {
	t, ok := r.tools[name]
	if !ok {
		return fmt.Errorf("unknown tool %q", name)
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if err := t.schema.Validate(value); err != nil {
		return fmt.Errorf("schema validation: %w", err)
	}
	return nil
}

func (r *Registry) Specs() []llm.ToolSpec {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]llm.ToolSpec, 0, len(names))
	for _, name := range names {
		d := r.tools[name].tool.Definition()
		result = append(result, llm.ToolSpec{Name: d.Name, Description: d.Description, Schema: d.Schema})
	}
	return result
}
