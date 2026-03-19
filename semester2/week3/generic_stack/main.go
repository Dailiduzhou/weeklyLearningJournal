package main

import "fmt"

type Stack[T any] []T

func (s *Stack[T]) push(v T) {
	*s = append(*s, v)
}

func (s *Stack[T]) IsEmpty() bool {
	return len(*s) == 0
}

func (s *Stack[T]) pop() (T, bool) {
	if s.IsEmpty() {
		var zero T
		return zero, false
	}

	index := len(*s) - 1
	top := (*s)[index]

	var zero T
	(*s)[index] = zero

	*s = (*s)[:index]
	return top, true
}

func (s *Stack[T]) peek() (T, bool) {
	if s.IsEmpty() {
		var zero T
		return zero, false
	}

	return (*s)[len(*s)-1], true
}

func (s *Stack[T]) size() int {
	return len(*s)
}

func main() {
	var s Stack[int]
	s.push(1)
	s.push(2)
	s.push(3)
	fmt.Println(s.peek())

	fmt.Println("size:", s.size())

	if t, ok := s.pop(); ok {
		fmt.Println(t)
		fmt.Println("After pop, size :", s.size())
	}
}
