// Copilot assisted
package main

import (
	"fmt"
	"sync"
)

func main() {
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	demonum := 100

	type state struct {
		cond       *sync.Cond
		mu         sync.Mutex
		letterDone bool
		letteridx  int
		number     int
		turn       int
		maxnum     int
	}

	st := &state{
		turn:       0,
		letterDone: false,
		letteridx:  0,
		number:     0,
		maxnum:     demonum,
	}
	st.cond = sync.NewCond(&st.mu)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for {
			st.mu.Lock()

			if st.turn != 0 && !st.letterDone {
				st.cond.Wait()
			}

			if st.letteridx >= len(letters) {
				st.letterDone = true
				st.turn = 1
				st.cond.Broadcast()
				st.mu.Unlock()
				return
			}

			for k := 0; k < 2 && st.letteridx < len(letters); k++ {
				fmt.Print(string(letters[st.letteridx]))
				st.letteridx++
			}

			st.turn = 1
			st.cond.Broadcast()
			st.mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()

		for {
			st.mu.Lock()

			if st.turn != 1 && !st.letterDone {
				st.cond.Wait()
			}

			if st.number > st.maxnum {
				st.mu.Unlock()
				return
			}

			if st.letterDone {
				for st.number <= st.maxnum {
					fmt.Print(st.number)
					st.number++
				}
				st.mu.Unlock()
				return
			}

			for k := 0; k < 2 && st.number <= st.maxnum; k++ {
				fmt.Print(st.number)
				if st.number == st.maxnum {
					st.number++
					st.mu.Unlock()
					return
				}
				st.number++
			}

			st.turn = 0
			st.cond.Signal()
			st.mu.Unlock()
		}
	}()

	wg.Wait()

}
