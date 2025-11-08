package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

type UserInfo struct {
	Name    string
	Gender  string
	Age     int
	EnName  string
	IsAdmin bool
}

func main() {
	user := UserInfo{
		Name:    "枯藤",
		Gender:  "男",
		Age:     18,
		EnName:  "Fuji",
		IsAdmin: true,
	}

	user1 := UserInfo{
		Name:    "藤",
		Gender:  "女",
		Age:     18,
		EnName:  "female",
		IsAdmin: false,
	}

	// {{.}}
	tmpl1 := `hello {{.}}!`
	t1 := template.Must(template.New("tmpl1").Parse(tmpl1))
	t1.Execute(os.Stdout, user)
	fmt.Println()

	// {{.Name}}
	tmpl2 := `hello {{.Name}}!`
	t2 := template.Must(template.New("tmpl2").Parse(tmpl2))
	t2.Execute(os.Stdout, user)
	fmt.Println()

	// if-else, if-elif-else and pipeline
	tmpl3 := `{{ if .IsAdmin}}{{.Name | printf "%s , you're admin, congratulations!\n"}}{{else}}{{.Name | printf "%s, you're not admin. That doesn't matter.\n"}} {{ end }} `
	t3 := template.Must((template.New("tmpl3").Parse(tmpl3)))
	t3.Execute(os.Stdout, user)
	t3.Execute(os.Stdout, user1)

	// pipeline and Func
	funcmap := template.FuncMap{
		"upper": strings.ToUpper,
	}
	tmpl4 := `{{.EnName | upper | printf "%s, upper case.\n"}}`
	t4 := template.Must(template.New("tmpl4").Funcs(funcmap).Parse(tmpl4))
	t4.Execute(os.Stdout, user)

	// FuncMap
	tmpl5 := `{{ .EnName | upper}}, upper case.`
	t5 := template.Must(template.New("tmpl5").Funcs(funcmap).Parse(tmpl5))
	t5.Execute(os.Stdout, user)

	// range
	// with
	// omits
}
