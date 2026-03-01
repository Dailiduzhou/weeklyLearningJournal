## Code
``` go
// the way to go
// panic_package.go
package main

import (
	"fmt"

	"github.com/Dailiduzhou/weeklyLearningJournal/week1/panic/parse"
)

func main() {
	var examples = []string{
		"1 2 3 4 5",
		"100 50 25 12.5 6.25",
		"2 + 2 = 4",
		"1st class",
		"",
	}

	for _, ex := range examples {
		fmt.Printf("Parsing %q:\n  ", ex)
		nums, err := parse.Parse(ex)
		if err != nil {
			fmt.Println(err) // here String() method from ParseError is used
			continue
		}
		fmt.Println(nums)
	}
}
```
## package
``` go
// the way to go
// parse.go
package parse

import (
	"fmt"
	"strconv"
	"strings"
)

// A ParseError indicates an error in converting a word into an integer.
type ParseError struct {
	Index int    // The index into the space-separated list of words.
	Word  string // The word that generated the parse error.
	Err   error  // The raw error that precipitated this error, if any.
}

// String returns a human-readable error message.
func (e *ParseError) String() string {
	return fmt.Sprintf("pkg parse: error parsing %q as int", e.Word)
}

// Parse parses the space-separated words in in put as integers.
func Parse(input string) (numbers []int, err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
			}
		}
	}()

	fields := strings.Fields(input)
	numbers = fields2numbers(fields)
	return
}

func fields2numbers(fields []string) (numbers []int) {
	if len(fields) == 0 {
		panic("no words to parse")
	}
	for idx, field := range fields {
		num, err := strconv.Atoi(field)
		if err != nil {
			panic(&ParseError{idx, field, err})
		}
		numbers = append(numbers, num)
	}
	return
}
```
## `example`\#1
``` go
var examples = []string{
		"1 2 3 4 5",
		"100 50 25 12.5 6.25",
		"2 + 2 = 4",
		"1st class",
		"",
	}
```
## `output`\#1
```
Parsing "1 2 3 4 5":
  [1 2 3 4 5]
Parsing "100 50 25 12.5 6.25":
  pkg: pkg parse: error parsing "12.5" as int
Parsing "2 + 2 = 4":
  pkg: pkg parse: error parsing "+" as int
Parsing "1st class":
  pkg: pkg parse: error parsing "1st" as int
Parsing "":
  pkg: no words to parse
```
## `example`\#2
``` go
var examples = []string{
		"25时、nightcord见",
		"100 50 6.22 12.5 6.25",
		"mygo!!!!!",
		"5人乐队",
		"组",
		"1辈子",
		"band",
	}
```
## `output`\#2
```
Parsing "100 50 6.22 12.5 6.25":
  pkg: pkg parse: error parsing "6.22" as int
Parsing "mygo!!!!!":
  pkg: pkg parse: error parsing "mygo!!!!!" as int
Parsing "5人乐队":
  pkg: pkg parse: error parsing "5人乐队" as int
Parsing "组":
  pkg: pkg parse: error parsing "组" as int
Parsing "1辈子":
  pkg: pkg parse: error parsing "1辈子" as int
Parsing "band":
  pkg: pkg parse: error parsing "band" as int
```