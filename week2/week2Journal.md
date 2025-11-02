# 哈嗨嗨，第二周开始了
## `并行` 和 `并发` 的差别
> Copilot大人，太伟大了

| **特性**             | **并行 (Parallelism)**                          | **并发 (Concurrency)**       |
|----------------------|------------------------------------------------|------------------------------------------------|
| **定义** | 同一时间点执行多个任务    | 任务在同一时间段交替执行，未必同时运行   |
| **关注点**| 同时执行多个任务的能力    | 高效地管理多个任务的能力  |
| **核心思想** | 做更多的事（同时执行） | 看起来在同时做很多事（交替切换） |
| **依赖**  | 需要多核/多处理器支持   | 不需要多核，单核即可实现    |
| **实现方式**         | 任务独立运行在不同的处理器或核心上             | 通过时间片轮转或异步方式交替执行任务           |
| **示例**| 启动多个线程同时对数据进行计算  | 单线程中通过I/O多路复用处理多个网络连接  |
| **适用场景**| 计算密集型任务，如矩阵运算或图像处理 | I/O密集型任务，如服务器处理多个客户端请求      |
| **硬件需求**  | 依赖多核CPU或多处理器         | 不依赖特定硬件，单核也能实现                   |
| **同步性**     | 各任务通常独立运行，无需频繁交互     | 各任务可能需要频繁交互或同步    |

> 才发现看云上的《the way to go》比Github上少了好多章节

## **埃式筛**的go routine实现
``` go
// the way to go
// sieve.go

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.package main
package main

import "fmt"

// Send the sequence 2, 3, 4, ... to channel 'ch'.
func generate(ch chan int) {
	for i := 2; ; i++ {
		ch <- i // Send 'i' to channel 'ch'.
	}
}

// Copy the values from channel 'in' to channel 'out',
// removing those divisible by 'prime'.
func filter(in, out chan int, prime int) {
	for {
		i := <-in // Receive value of new variable 'i' from 'in'.
		if i%prime != 0 {
			out <- i // Send 'i' to channel 'out'.
		}
	}
}

// The prime sieve: Daisy-chain filter processes together.
func main() {
	ch := make(chan int) // Create a new channel.
	go generate(ch)      // Start generate() as a goroutine.
	for {
		prime := <-ch
		fmt.Print(prime, " ")
		ch1 := make(chan int)
		go filter(ch, ch1, prime)
		ch = ch1
	}
}
```
并发开启多个`filter goroutine`，每个数要经过`filter`链才能被确定为**质数**。


| 时间点| filter链长度| 能正确筛除的合数范围 |
| --- | :-: | :--|
开始时 |0  | 只能识别2
发现2后| 1  | 能筛2的倍数（4,6,8...）
发现3后|2 | 能筛2,3的倍数（4,6,8,9,12...）  
发现5后| 3 | 能筛2,3,5的倍数（最大到25）
发现7后| 4 | 能筛2,3,5,7的倍数（最大到49）

`Alternative`
``` go
// the way to go
// sieve2.go

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
)

// Send the sequence 2, 3, 4, ... to returned channel
func generate() chan int {
	ch := make(chan int)
	go func() {
		for i := 2; ; i++ {
			ch <- i
		}
	}()
	return ch
}

// Filter out input values divisible by 'prime', send rest to returned channel
func filter(in chan int, prime int) chan int {
	out := make(chan int)
	go func() {
		for {
			if i := <-in; i%prime != 0 {
				out <- i
			}
		}
	}()
	return out
}

func sieve() chan int {
	out := make(chan int)
	go func() {
		ch := generate()
		for {
			prime := <-ch
			ch = filter(ch, prime)
			out <- prime
		}
	}()
	return out
}

func main() {
	primes := sieve()
	for {
		fmt.Println(<-primes)
	}
}
```