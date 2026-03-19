package main

import (
	"fmt"
	"mymarshal/marshal"
	"mymarshal/unmarshal"
)

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

	bytes, err := marshal.Marshal(data)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(string(bytes))

	type User1 struct {
		Name    string   `json:"name"`
		Age     int      `json:"age"`
		Emails  []string `json:"emails"`
		Ignored string   `json:"-"`
	}

	type Response struct {
		Code int     `json:"code"`
		Msg  string  `json:"msg"`
		Data []User1 `json:"data"`
	}

	jsonStr := `
	{
		"code": 200,
		"msg": "success",
		"data": [
			{"name": "Alice", "age": 25, "emails": ["alice@work.com", "alice@home.com"]},
			{"name": "Bob", "age": 30, "emails": []}
		]
	}`

	var resp Response
	// 泛型调用，T 被推导为 Response，传入指针 &resp
	err = unmarshal.Unmarshal([]byte(jsonStr), &resp)
	if err != nil {
		fmt.Println("Unmarshal error:", err)
		return
	}

	fmt.Printf("解析结果: %+v\n", resp)
}
