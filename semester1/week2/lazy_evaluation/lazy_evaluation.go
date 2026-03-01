// the way to go
// lazy_evaluation.go

package main

import (
	"fmt"
)

var resume chan int

func integers() chan int {
	yield := make(chan int)
	count := 0
	go func() {
		for {
			yield <- count // 阻塞 channel
			count++
		}
	}()
	return yield
}

func generateInteger() int {
	return <-resume
}

func main() {
	resume = integers()
	const n = 30
	for range n {
		fmt.Println(generateInteger())
	}
}
