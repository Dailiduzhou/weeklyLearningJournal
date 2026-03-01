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
