// beta ver.
/*
	感觉不太能不使用其他空间实现功能。
	笨蛋大学生尽力了。
*/
package main

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

type msg struct {
	Rand int
	Id   int
}

func main() {
	rand.Seed(time.Now().UnixNano())
	const n = 20

	input := make(chan msg, n)
	var wg sync.WaitGroup

	for i := 1; i <= n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			random := rand.Intn(100)
			input <- msg{Rand: random, Id: id}
		}(i)
	}

	wg.Wait()
	close(input)

	data := make([]msg, 0, n)
	for it := range input {
		data = append(data, it)
	}

	fmt.Println("排序前结果：")
	for _, it := range data {
		fmt.Printf("Go routine ID: %d, random num: %d\n", it.Id, it.Rand)
	}

	sort.Slice(data, func(i, j int) bool {
		return data[i].Id < data[j].Id
	})

	fmt.Println("\n排序后结果:")
	for _, it := range data {
		fmt.Printf("Go routine ID: %d, random num: %d\n", it.Id, it.Rand)
	}
}
