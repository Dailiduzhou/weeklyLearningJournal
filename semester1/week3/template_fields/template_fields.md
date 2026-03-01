# 导入和未导入字段
## 未导入字段
###  `Code`
``` go
package main

import (
	"fmt"
	"os"
	"text/template"
)

type Person struct {
	Name                string
	nonExportedAgeField string
}

func main() {
	t := template.New("hello")

	// t, _ = t.Parse("hello {{.Name}}!")

	t, _ = t.Parse("hello {{.nonExportedAgeField}}!")

	p := Person{Name: "Mary", nonExportedAgeField: "31"}
	if err := t.Execute(os.Stdout, p); err != nil {
		fmt.Println("There was an error:", err.Error())
	}
}
```
### `Output`
```
hello There was an error: template: hello:1:8: executing "hello" at <.nonExportedAgeField>: nonExportedAgeField is an unexported field of struct type main.Person
```
## 导入字段
### `Code`
``` go
package main

import (
	"fmt"
	"os"
	"text/template"
)

type Person struct {
	Name                string
	nonExportedAgeField string
}

func main() {
	t := template.New("hello")

	t, _ = t.Parse("hello {{.Name}}!")

	// t, _ = t.Parse("hello {{.nonExportedAgeField}}!")

	p := Person{Name: "Mary", nonExportedAgeField: "31"}
	if err := t.Execute(os.Stdout, p); err != nil {
		fmt.Println("There was an error:", err.Error())
	}
}
```
### `Output`
```
hello Mary!
```