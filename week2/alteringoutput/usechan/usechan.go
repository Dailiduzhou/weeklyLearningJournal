// Copilot assisted
package main

import "fmt"

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

		for {
			select {
			case <-chNums:
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
					for n <= maxn {
						fmt.Print(n)
						if n == maxn {
							close(Done)
							return
						}
						n++
					}
					close(Done)
					return
				default:
					chLetters <- struct{}{}
				}

			case <-lettersDone:
				for n <= maxn {
					fmt.Print(n)
					if n == maxn {
						close(Done)
						return
					}
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
