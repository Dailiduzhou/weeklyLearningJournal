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