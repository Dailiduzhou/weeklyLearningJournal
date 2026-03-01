# Slice.go

## Code
``` go
package main

import "fmt"

func main() {
	slice := make([]byte, 5)
	fmt.Printf("slice, len: %d, cap: %d\n", len(slice), cap(slice))
	slice = slice[2:4]
	fmt.Printf("resized slice, len : %d, cap: %d\n", len(slice), cap(slice))

	s := "hello，世界"
	fmt.Printf("%d\n", len(s))
	for i, j := range s {
		fmt.Printf("%d : %c\n", i, j)
	}
}
```
## Output
```
slice, len: 5, cap: 5
resized slice, len : 2, cap: 3
14
0 : h
1 : e
2 : l
3 : l
4 : o
5 : ，
8 : 世
11 : 界
```