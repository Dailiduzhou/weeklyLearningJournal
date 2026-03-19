package unmarshal

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// Unmarshal 是泛型入口。强制要求传入指针类型 *T
func Unmarshal[T any](data []byte, v *T) error {
	// 1. 将 JSON 字符串解析为中间态的 interface{} (AST)
	p := &parser{s: string(data), i: 0}
	parsedData, err := p.parseValue()
	if err != nil {
		return err
	}

	// 2. 利用反射，将中间态数据填充到目标结构体中
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("v must be a non-nil pointer")
	}

	return populate(rv.Elem(), parsedData)
}

func populate(v reflect.Value, data any) error {
	if data == nil {
		return nil // null 值跳过
	}

	// 处理指针：如果目标是指针，解引用；如果是 nil 指针，则初始化它
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return populate(v.Elem(), data)
	}

	switch v.Kind() {
	case reflect.String:
		if s, ok := data.(string); ok {
			v.SetString(s)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// JSON 中的数字在解析时统一当作 float64 处理，这里需要强转
		if f, ok := data.(float64); ok {
			v.SetInt(int64(f))
		}
	case reflect.Float32, reflect.Float64:
		if f, ok := data.(float64); ok {
			v.SetFloat(f)
		}
	case reflect.Bool:
		if b, ok := data.(bool); ok {
			v.SetBool(b)
		}
	case reflect.Slice:
		list, ok := data.([]any)
		if !ok {
			return fmt.Errorf("expected JSON array")
		}
		// 动态创建一个指定类型的切片
		slice := reflect.MakeSlice(v.Type(), len(list), len(list))
		for i, elem := range list {
			if err := populate(slice.Index(i), elem); err != nil {
				return err
			}
		}
		v.Set(slice)
	case reflect.Struct:
		m, ok := data.(map[string]any)
		if !ok {
			return fmt.Errorf("expected JSON object")
		}
		typ := v.Type()
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if !field.IsExported() {
				continue
			}

			// 解析 Tag
			keyName := field.Name
			tag := field.Tag.Get("json")
			if tag == "-" {
				continue
			}
			if tag != "" {
				keyName = strings.Split(tag, ",")[0]
			}

			// 如果 JSON 中存在该字段的值，则递归赋值
			if val, exists := m[keyName]; exists {
				if err := populate(v.Field(i), val); err != nil {
					return err
				}
			}
		}
	default:
		return fmt.Errorf("unsupported target type: %s", v.Kind())
	}
	return nil
}

type parser struct {
	s string
	i int
}

func (p *parser) skip() {
	for p.i < len(p.s) && (p.s[p.i] == ' ' || p.s[p.i] == '\n' || p.s[p.i] == '\r' || p.s[p.i] == '\t') {
		p.i++
	}
}

func (p *parser) parseValue() (any, error) {
	p.skip()
	if p.i >= len(p.s) {
		return nil, io.EOF
	}
	c := p.s[p.i]
	switch c {
	case '{':
		return p.parseObject()
	case '[':
		return p.parseArray()
	case '"':
		return p.parseString()
	case 't', 'f':
		return p.parseBool()
	case 'n':
		return p.parseNull()
	default:
		return p.parseNumber()
	}
}

func (p *parser) parseObject() (map[string]any, error) {
	p.i++ // skip '{'
	p.skip()
	if p.i < len(p.s) && p.s[p.i] == '}' {
		p.i++ // 空对象 {}
		return map[string]any{}, nil
	}

	obj := make(map[string]any)
	for {
		p.skip()
		key, err := p.parseString()
		if err != nil {
			return nil, err
		}
		p.skip()
		if p.i >= len(p.s) || p.s[p.i] != ':' {
			return nil, fmt.Errorf("expected ':'")
		}
		p.i++ // skip ':'

		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		obj[key] = val

		p.skip()
		if p.i >= len(p.s) {
			return nil, io.EOF
		}
		if p.s[p.i] == '}' {
			p.i++
			break
		}
		if p.s[p.i] == ',' {
			p.i++
		} else {
			return nil, fmt.Errorf("expected ',' or '}'")
		}
	}
	return obj, nil
}

func (p *parser) parseArray() ([]any, error) {
	p.i++ // skip '['
	p.skip()
	if p.i < len(p.s) && p.s[p.i] == ']' {
		p.i++
		return []any{}, nil
	}

	var arr []any
	for {
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		arr = append(arr, val)
		p.skip()
		if p.i >= len(p.s) {
			return nil, io.EOF
		}
		if p.s[p.i] == ']' {
			p.i++
			break
		}
		if p.s[p.i] == ',' {
			p.i++
		} else {
			return nil, fmt.Errorf("expected ',' or ']'")
		}
	}
	return arr, nil
}

func (p *parser) parseString() (string, error) {
	p.i++ // skip starting '"'
	start := p.i
	// 极简处理：不考虑 \", \n 等转义符的复杂情况
	for p.i < len(p.s) && p.s[p.i] != '"' {
		p.i++
	}
	if p.i >= len(p.s) {
		return "", fmt.Errorf("unclosed string")
	}
	res := p.s[start:p.i]
	p.i++ // skip ending '"'
	return res, nil
}

func (p *parser) parseNumber() (float64, error) {
	start := p.i
	for p.i < len(p.s) && ((p.s[p.i] >= '0' && p.s[p.i] <= '9') || p.s[p.i] == '.' || p.s[p.i] == '-') {
		p.i++
	}
	return strconv.ParseFloat(p.s[start:p.i], 64)
}

func (p *parser) parseBool() (bool, error) {
	if strings.HasPrefix(p.s[p.i:], "true") {
		p.i += 4
		return true, nil
	}
	if strings.HasPrefix(p.s[p.i:], "false") {
		p.i += 5
		return false, nil
	}
	return false, fmt.Errorf("invalid boolean")
}

func (p *parser) parseNull() (any, error) {
	if strings.HasPrefix(p.s[p.i:], "null") {
		p.i += 4
		return nil, nil
	}
	return nil, fmt.Errorf("invalid null")
}
