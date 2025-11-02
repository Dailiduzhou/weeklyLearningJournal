## `Code`
``` go
// the way to go
// channel_idiom2.go modified ver.

package main

import (
	"fmt"
	"time"
)

func main() {
	suck(pump())
	time.Sleep(1e5) // modified
}

func pump() chan int {
	ch := make(chan int)
	go func() {
		for i := 0; ; i++ {
			ch <- i
		}
	}()
	return ch
}

func suck(ch chan int) {
	go func() {
		for v := range ch {
			fmt.Println(v)
		}
	}()
}
```
## `Output`
```
0
1
2
3
4
5
6
7
8
9
10
11
12
13
```