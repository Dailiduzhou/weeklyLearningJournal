# Week3
## 试试template包
这玩意在写简易wiki有点用，拿`the way to go`里的代码试试。
## 带文件系统的简单网页
[elaborated_webserver.go](./elaborated_webserver/elaborated_webserver.go)
带文件系统的Handle常见写法：
``` go
package main

import (
	"net/http"
)

func main() {
	// To serve a directory on disk (/tmp) under an alternate URL
	// path (/tmpfiles/), use StripPrefix to modify the request
	// URL's path before the FileServer sees it:
	http.Handle("/tmpfiles/", http.StripPrefix("/tmpfiles/", http.FileServer(http.Dir("/tmp"))))
}
```
在绝对路径/temp下寻找目标文件。

---

也可通过命令行传入路径：
### `Code`
``` go
// example.go
package main

import (
    "net/http"
    "log"
)
// flags:
var webroot = flag.String("root", "./", "web root directory")

func main(){
    http.Handle("/go/", http.StripPrefix("/go/", http.FilServer(http.Dir(webroot))))
    err := http.ListenAndServe(":12345", nil)
	if err != nil {
		log.Panicln("ListenAndServe:", err)
	}
}
```
默认在相对路径./下寻找文件。
通过`go run example.go -root=xxx`传入路径。

## 木犀骇客吐槽
```
还剩最后一道门了！
我们需要银行结构图碎片，这些碎片就隐藏在前面某四个路由的响应头中，位于 map-fragments 字段。
将它们用"/"拼起来就是最后一道门的所在位置！注意response的信息。
```
这玩意要整死我了。
我选择看网站源码的`swagger.yaml`来跳关。

---

``` go
// SendRequest ... 发送消息，根据状态码输出提示
func (r *HttpRequest) SendRequest() (*HttpResponse, error) {
	response := new(HttpResponse)
	var err error
	client := &http.Client{}
	response.raw, err = client.Do(r.Req)
	if err != nil {
		return nil, err
	}

	if response.raw.StatusCode == 200 {
		err = resolveResponse(response)
		if err != nil {
			return nil, err
		}
		fmt.Println("Send request successfully! Please check your response body.")
		//fmt.Println("request success the data is: ")
		//fmt.Println(response.Body.Data.Text)
		//fmt.Println("the Extra info is: ")
		//fmt.Println(response.Body.Data.ExtraInfo)
	} else {
		body, err := ioutil.ReadAll(response.raw.Body)
		if err != nil {
			fmt.Println("read body error" + err.Error())
			return nil, err
		}

		if response.raw.StatusCode == 400 {
			fmt.Println("http 400 failed! the wrong data is: ")
			fmt.Println(string(body))
		} else if response.raw.StatusCode == 404 {
			fmt.Println("http 404. we can not find the path, did you input the right information? the wrong message is: ")
			fmt.Println(string(body))
		} else if response.raw.StatusCode == 500 {
			fmt.Println("http 500. server error, message: ")
			fmt.Println(string(body))
		}
		response.Raw = string(body)

		return response, nil
	}

	return response, nil
}

func handleFile(url, path string) (*HttpRequest, error) {
	req := new(HttpRequest)
	req.BodyType = FILE

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	file, errFile := os.Open(path)
	if errFile != nil {
		return nil, errFile
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}(file)

	part1, errFile := writer.CreateFormFile("file", filepath.Base(path))
	if errFile != nil {
		return nil, errFile
	}

	_, errFile = io.Copy(part1, file)
	if errFile != nil {
		return nil, errFile
	}

	err := writer.Close()
	if err != nil {
		return nil, errFile
	}

	req.Req, err = http.NewRequest("POST", url, payload)
	if err != nil {
		return nil, err
	}

	req.SetHeader("Content-Type", writer.FormDataContentType())

	return req, nil
}
```
`handleFile`不检查请求方式吗？

## 论证qzh其实是**地雷妹**
| 地雷妹          | qzh        |
| ------------ | ---------- |
| 精神药物overdose | sleep od   |
| 改花刀          | 运动受伤       |
| 自残           | 满课周二跑步     |
| 精神不稳定        | 精神疑似**正常** |
## `strings.FieldsFunc()`用法
函数签名`func FieldsFunc(s string, f func(rune) bool) []string`
示例:
``` go
package main

import (
	"fmt"
	"strings"
	"unicode"
)

func main() {
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}
	fieldsstr := strings.FieldsFunc("  foo1;bar2,baz3...", f)
	for _, v := range fieldsstr {
		fmt.Printf("%v\n", v)
	}
}
```
`Output`
```
foo1
bar2
baz3
```
将`func f(c rune)`返回值为`true`的字符作为分隔符。
## 研究`hacker-support`包
大概看了各函数的实现，并搜索学习了各调用函数的签名、功能。
现在对AES的一般用法不太了解，在深入学习。