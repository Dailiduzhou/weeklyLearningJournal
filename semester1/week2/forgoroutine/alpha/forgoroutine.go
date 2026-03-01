package main

import (
	"fmt"
	"math/rand"
	"sync"
)

type msg struct {
	Rand int
	Id   int
}

func main() {
	const n = 20

	input := make([]chan msg, n+1)
	for i := 1; i <= n; i++ {
		input[i] = make(chan msg, 1)
	}
	var wg sync.WaitGroup

	for i := 1; i <= n; i++ {
		wg.Add(1)
		go func(id int) {
			random := rand.Intn(100)
			defer wg.Done()

			input[id] <- msg{Rand: random, Id: id}
		}(i)
	}

	wg.Wait()

	fmt.Printf("排序前结果：\n")
	for i := 1; i <= 20; i++ {
		tmsg := <-input[i]
		fmt.Printf("Go routine ID: %d, random num: %d\n", tmsg.Id, tmsg.Rand)
		input[i] <- tmsg
	}

	fmt.Printf("\n排序后结果: \n")
	for i := 1; i <= n; i++ {
		tmsg := <-input[i] // 按 ID 顺序读取
		fmt.Printf("Go routine ID: %d, random num: %d\n", tmsg.Id, tmsg.Rand)
	}
	// for i := 1; i <= n; i++ {
	// 	for it := range input[i] {
	// 		if i == it.Id {
	// 			fmt.Printf("Go routine ID: %d, random num: %d\n", it.Id, it.Rand)
	// 		}
	// 	}
	// }
}
