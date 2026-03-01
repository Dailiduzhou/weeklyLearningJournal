package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type msg struct {
	Rand int
	Id   int
}

func main() {
	const n = 20
	input := make(chan msg, 20)
	var wg sync.WaitGroup

	for i := range n {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// 宝宝、宝宝，胖手套，爱睡觉
			randomsleep := rand.Intn(1000)
			randnum := rand.Intn(100)
			time.Sleep(time.Duration(randomsleep) * time.Millisecond)
			input <- msg{Rand: randnum, Id: id}
		}(i)
	}

	wg.Wait()

	fmt.Printf("排序前的结果:\n")
	for range n {
		temp := <-input
		fmt.Printf("Go routine Id: %d, random number: %d\n", temp.Id, temp.Rand)
		input <- temp
	}

	flag := make([]bool, 20)
	fmt.Printf("排序后的结果:\n")
	for i := range n {
		for range n {
			temp := <-input
			if temp.Id == i && !flag[i] {
				flag[i] = true
				fmt.Printf("Go routine Id: %d, random number: %d\n", temp.Id, temp.Rand)
			}
			input <- temp
		}
	}
}
