package main

import "fmt"

type Counter struct {
	value int
}

func (c *Counter) initialize() {
	c.value = 0
}

func (c *Counter) increment(a ...int) {
	switch len(a) {
	case 0:
		c.value += 1
	case 1:
		c.value += a[0]
	}
}

func (c *Counter) decreament(a ...int) {
	switch len(a) {
	case 0:
		c.value -= 1
	case 1:
		c.value -= a[0]
	}
}

// 好的，我意识到了“脱裤子放屁”的意义
func (c *Counter) getValue() int {
	return c.value
}

// 依旧
func (c *Counter) reset() func() {
	return func() {
		c.value = 0
	}
}

func main() {
	var cnt Counter

	cnt.initialize()
	fmt.Println("getValue(): ", cnt.getValue())

	cnt.increment()
	fmt.Println("increment():", cnt.getValue())

	cnt.increment(3)
	fmt.Println("increment(3): ", cnt.getValue())

	cnt.decreament()
	fmt.Println("decrement(): ", cnt.getValue())

	cnt.decreament(2)
	fmt.Println("decrement(): ", cnt.getValue())

	cnt.reset()
	fmt.Println("reset(): ", cnt.getValue())
}
