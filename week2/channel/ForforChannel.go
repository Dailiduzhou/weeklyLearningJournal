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
