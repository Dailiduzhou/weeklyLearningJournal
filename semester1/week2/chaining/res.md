## `Code`
``` go
// the way to go
// chaining.go
package main

import (
	"flag"
	"fmt"
)

var ngoroutine = flag.Int("n", 100000, "how many goroutines")

func f(left, right chan int) { left <- 1 + <-right }

func main() {
	flag.Parse()
	leftmost := make(chan int)
	var left, right chan int = nil, leftmost
	for i := 0; i < *ngoroutine; i++ {
		left, right = right, make(chan int)
		go f(left, right)
	}
	right <- 0      // bang!
	x := <-leftmost // wait for completion
	fmt.Println(x)  // 100000, about 1.5 s
}
```
## `Terminal` #1
```
go run chaining.go
```
## `output`
```
100000
```
## `Terminal` #2
```
go run chaining.go -n=100
```
## `output`
```
100
```