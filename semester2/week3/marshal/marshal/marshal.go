package marshal

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func Marshal[T any](v T) ([]byte, error) {
	str, err := encode(v)
	if err != nil {
		return nil, err
	}
	return []byte(str), nil
}

func encode(v any) (string, error) {
	if v == nil {
		return "null", nil
	}

	val := reflect.ValueOf(v)

	switch val.Kind() {
	case reflect.String:
		return fmt.Sprintf(`"%s"`, val.String()), nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(val.Uint(), 10), nil

	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(val.Float(), 'f', -1, 64), nil

	case reflect.Bool:
		return strconv.FormatBool(val.Bool()), nil

	case reflect.Slice, reflect.Array:
		var parts []string
		for i := 0; i < val.Len(); i++ {
			elemStr, err := encode(val.Index(i).Interface())
			if err != nil {
				return "", err
			}
			parts = append(parts, elemStr)
		}
		return "[" + strings.Join(parts, ",") + "]", nil

	case reflect.Map:
		var parts []string
		for _, key := range val.MapKeys() {
			keyStr := fmt.Sprintf(`"%v"`, key.Interface())
			valStr, err := encode(val.MapIndex(key).Interface())
			if err != nil {
				return "", err
			}
			parts = append(parts, keyStr+":"+valStr)
		}
		return "{" + strings.Join(parts, ",") + "}", nil

	case reflect.Struct:
		var parts []string
		typ := val.Type()
		for i := 0; i < val.NumField(); i++ {
			field := typ.Field(i)

			if !field.IsExported() {
				continue
			}

			keyName := field.Name
			tag := field.Tag.Get("json")
			if tag == "-" {
				continue
			}
			if tag != "" {
				keyName = strings.Split(tag, ",")[0]
			}

			valStr, err := encode(val.Field(i).Interface())
			if err != nil {
				return "", err
			}
			parts = append(parts, fmt.Sprintf(`"%s":%s`, keyName, valStr))
		}
		return "{" + strings.Join(parts, ",") + "}", nil

	case reflect.Pointer, reflect.Interface:
		if val.IsNil() {
			return "null", nil
		}
		return encode(val.Elem().Interface())

	default:
		return "", fmt.Errorf("unsupported type: %s", val.Kind())
	}
}

func main() {
	type User struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Email string `json:"-"`
	}

	data := map[string]any{
		"code": 200,
		"msg":  "success",
		"data": []User{
			{"Alice", 25, "alice@test.com"},
			{"Bob", 30, "bob@test.com"},
		},
		"is_active": true,
	}

	bytes, err := Marshal(data)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(string(bytes))
}
