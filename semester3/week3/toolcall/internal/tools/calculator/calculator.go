package calculator

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"toolcall/internal/tool"
)

const (
	maxExpressionLength = 256
	maxParseDepth       = 32
	maxAbsValue         = 1e15
)

type Tool struct{}

func New() *Tool { return &Tool{} }

func (*Tool) Definition() tool.Definition {
	return tool.Definition{
		Name:        "calculator",
		Description: "Evaluate a safe arithmetic expression. Supports numbers, parentheses, +, -, *, / and ^ only.",
		Type:        tool.TypeRead,
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"expression": map[string]any{"type": "string", "minLength": 1, "maxLength": maxExpressionLength},
			},
			"required":             []string{"expression"},
			"additionalProperties": false,
		},
	}
}

func (*Tool) Execute(ctx context.Context, raw json.RawMessage) tool.Result {
	if err := ctx.Err(); err != nil {
		return contextFailure(err)
	}
	var args struct {
		Expression string `json:"expression"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return tool.Failure("invalid_arguments", err.Error(), false)
	}
	p := parser{input: args.Expression, ctx: ctx}
	value, err := p.parse()
	if err != nil {
		if ctx.Err() != nil {
			return contextFailure(ctx.Err())
		}
		return tool.Failure("invalid_expression", err.Error(), false)
	}
	return tool.Success(map[string]any{"value": value}, strconv.FormatFloat(value, 'g', -1, 64))
}

func contextFailure(err error) tool.Result {
	code := "canceled"
	if err == context.DeadlineExceeded {
		code = "timeout"
	}
	return tool.Failure(code, err.Error(), true)
}

type parser struct {
	input string
	pos   int
	depth int
	ctx   context.Context
}

func (p *parser) parse() (float64, error) {
	if len(p.input) == 0 || len(p.input) > maxExpressionLength {
		return 0, fmt.Errorf("expression length must be between 1 and %d", maxExpressionLength)
	}
	v, err := p.expression()
	if err != nil {
		return 0, err
	}
	p.space()
	if p.pos != len(p.input) {
		return 0, fmt.Errorf("unexpected character %q at offset %d", p.input[p.pos], p.pos)
	}
	return checked(v)
}

func (p *parser) expression() (float64, error) {
	left, err := p.term()
	if err != nil {
		return 0, err
	}
	for {
		op := p.take("+-")
		if op == 0 {
			return checked(left)
		}
		right, err := p.term()
		if err != nil {
			return 0, err
		}
		if op == '+' {
			left += right
		} else {
			left -= right
		}
		if left, err = checked(left); err != nil {
			return 0, err
		}
	}
}

func (p *parser) term() (float64, error) {
	left, err := p.power()
	if err != nil {
		return 0, err
	}
	for {
		op := p.take("*/")
		if op == 0 {
			return checked(left)
		}
		right, err := p.power()
		if err != nil {
			return 0, err
		}
		if op == '*' {
			left *= right
		} else {
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			left /= right
		}
		if left, err = checked(left); err != nil {
			return 0, err
		}
	}
}

func (p *parser) power() (float64, error) {
	left, err := p.unary()
	if err != nil {
		return 0, err
	}
	if p.take("^") == '^' {
		right, err := p.power()
		if err != nil {
			return 0, err
		}
		return checked(math.Pow(left, right))
	}
	return left, nil
}

func (p *parser) unary() (float64, error) {
	if err := p.ctx.Err(); err != nil {
		return 0, err
	}
	if op := p.take("+-"); op != 0 {
		v, err := p.unary()
		if err != nil {
			return 0, err
		}
		if op == '-' {
			v = -v
		}
		return checked(v)
	}
	return p.primary()
}

func (p *parser) primary() (float64, error) {
	p.space()
	if p.pos < len(p.input) && p.input[p.pos] == '(' {
		p.pos++
		p.depth++
		if p.depth > maxParseDepth {
			return 0, fmt.Errorf("expression nesting exceeds %d", maxParseDepth)
		}
		v, err := p.expression()
		p.depth--
		if err != nil {
			return 0, err
		}
		p.space()
		if p.pos >= len(p.input) || p.input[p.pos] != ')' {
			return 0, fmt.Errorf("missing closing parenthesis")
		}
		p.pos++
		return v, nil
	}
	start := p.pos
	seenDigit := false
	seenDot := false
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		if c >= '0' && c <= '9' {
			seenDigit = true
			p.pos++
			continue
		}
		if c == '.' && !seenDot {
			seenDot = true
			p.pos++
			continue
		}
		break
	}
	if !seenDigit {
		return 0, fmt.Errorf("expected number at offset %d", start)
	}
	v, err := strconv.ParseFloat(p.input[start:p.pos], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %w", err)
	}
	return checked(v)
}

func (p *parser) take(chars string) byte {
	p.space()
	if p.pos < len(p.input) && strings.ContainsRune(chars, rune(p.input[p.pos])) {
		c := p.input[p.pos]
		p.pos++
		return c
	}
	return 0
}

func (p *parser) space() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

func checked(v float64) (float64, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > maxAbsValue {
		return 0, fmt.Errorf("numeric result is outside the allowed range")
	}
	return v, nil
}
