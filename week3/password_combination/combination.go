package main

import (
	"fmt"
)

func permute(nums []int) (res [][]int) {
	// insert your code
	var dfs func(path []int, used []bool)
	dfs = func(path []int, used []bool) {
		if len(path) == len(nums) {
			temp := make([]int, len(nums))
			copy(temp, path)
			res = append(res, temp)
			return
		}
		for i := 0; i < len(nums); i++ {
			if used[i] {
				continue
			}
			used[i] = true
			path = append(path, nums[i])
			// 递归
			dfs(path, used)
			path = path[:len(path)-1]
			used[i] = false
		}
	}
	dfs([]int{}, make([]bool, len(nums)))
	return
}
func main() {
	var n int
	fmt.Scanf("%d", &n)

	testSlice := make([]int, n)

	// 标准输入n个不重复的数字
	for i := 0; i < n; i++ {
		var a int
		fmt.Scanf("%d", &a)
		testSlice[i] = a
	}

	res := permute(testSlice)
	fmt.Println(res)
}
