// Copilot assisted
package main

import (
	"fmt"
)

const demonum int64 = 100

func main() {
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	chLetters := make(chan struct{})
	chNums := make(chan struct{})
	Done := make(chan struct{})
	lettersDone := make(chan struct{})

	go func() {
		i := 0

		for i < len(letters) {
			<-chLetters
			for k := 0; k < 2 && i < len(letters); k++ {
				fmt.Print(string(letters[i]))
				i++
			}

			chNums <- struct{}{}
		}

		close(lettersDone)
	}()

	go func() {
		var n int64 = 0
		maxn := demonum

		alt := true

		for {
			if alt {
				select {
				case <-chNums:
				case <-lettersDone:
					alt = false
				}

				// 复用打印两个数字的部分
				for k := 0; k < 2 && n <= maxn; k++ {
					fmt.Print(n)
					if n == maxn {
						close(Done)
						return
					}
					n++
				}

				select {
				case <-lettersDone:
					alt = false
				default:
					chLetters <- struct{}{}
				}
			} else {
				for n <= maxn {
					fmt.Print(n)
					n++
				}
				close(Done)
				return
			}
		}

	}()

	chLetters <- struct{}{}
	<-Done
}
