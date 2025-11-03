// counter_closure.go

package main

import "fmt"

type Counter struct {
	Increment func(a ...int) int // 递增
	Decrement func(a ...int) int // 递减
	GetValue  func() int         // 获取当前值
	Reset     func()             // 重置
}

// NewCounter 创建新的计数器
func NewCounter(initialValue int) Counter {
	// 使用闭包存储计数器的值
	value := initialValue

	return Counter{
		Increment: func(a ...int) int {
			switch len(a) {
			case 0:
				value++
				return value
			default:
				value += a[0]
				return value
			}
		},
		Decrement: func(a ...int) int {
			switch len(a) {
			case 0:
				value++
				return value
			default:
				value += a[0]
				return value
			}
		},
		GetValue: func() int {
			return value
		},
		Reset: func() {
			value = initialValue
		},
	}
}

func main() {
	counter := NewCounter(10)

	fmt.Println("初始值：", counter.GetValue())

	fmt.Println("递增：", counter.Increment())
	fmt.Println("递增：", counter.Increment(3))

	fmt.Println("递减：", counter.Decrement())
	fmt.Println("递减：", counter.Decrement(2))

	fmt.Println("当前值：", counter.GetValue())

	counter.Reset()
	fmt.Println("重置后的值：", counter.GetValue())
}
