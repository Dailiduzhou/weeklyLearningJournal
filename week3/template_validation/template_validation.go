package main

import (
	"fmt"
	"text/template"
)

func main() {
	tOk := template.New("ok")
	//a valid template, so no panic with Must:
	template.Must(tOk.Parse("/* and a comment */ some static text: {{ .Name }}"))
	fmt.Println("The first one parsed OK.")
	fmt.Println("The next one ought to fail.")
	tErr := template.New("error_template")
	template.Must(tErr.Parse(" some static text {{ .Name }"))
}

/*
func Must
func Must(t *Template, err error) *Template

Must is a helper that wraps a call to a function returning (*Template, error) and panics if the error is non-nil. It is intended for use in variable initializations such as
e.g.
	var t = template.Must(template.New("name").Parse("text"))
*/
