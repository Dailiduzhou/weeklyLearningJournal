// package main

// import "fmt"

// func main() {
// 	var n, E, maxE, r, cnt int
// 	fmt.Scanln(&n, &E, &r)
// 	dp := make([][]int, n)
// 	a := make([]int, n)
// 	var str string

// 	fmt.Scanf("%s", &str)
// 	for i := 0; i < n; i++ {
// 		if str[i] == '0' {
// 			a[i] = 0
// 		}
// 		if str[i] == '-' {
// 			a[i] = 1
// 		}
// 		if str[i] == '+' {
// 			a[i] = 2
// 			cnt++
// 		}
// 	}
// 	maxE = cnt*r + E

// 	for i := 0; i < n; i++ {
// 		dp[i] = make([]int, maxE+1)
// 		for j := 1; j <= maxE; j++ {
// 			dp[i][j] = 2147483647
// 		}
// 	}

// 	dp[0][E] = 0
// 	for i := 0; i < n; i++ {
// 		for j := 1; j <= maxE; j++ {
// 			if dp[i][j] != 2147483647 {
// 				for k := 1; k <= j && i+k < n; k++ {
// 					if a[i+k] == 0 {
// 						dp[i+k][j] = min(dp[i+k][j], dp[i][j]+1)
// 					}
// 					if a[i+k] == 2 {
// 						dp[i+k][j+r] = min(dp[i+k][j+r], dp[i][j]+1)
// 					}
// 				}
// 			}
// 		}
// 	}
// 	minJump := n
// 	for i := 1; i <= maxE; i++ {
// 		minJump = min(minJump, dp[n-1][i])
// 	}
// 	fmt.Println(minJump)
// }

package main

import (
	"fmt"
	"math"
)

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func main() {
	var n, E, r int
	fmt.Scanln(&n, &E, &r)

	var road string
	fmt.Scanln(&road)

	// 计算最大可能能量（保守估计）
	maxE := E
	for i := 0; i < n; i++ {
		if road[i] == '+' {
			maxE += r
		}
	}

	// 初始化DP数组
	dp := make([][]int, n)
	for i := range dp {
		dp[i] = make([]int, maxE+1)
		for j := range dp[i] {
			dp[i][j] = math.MaxInt32
		}
	}

	// 起点初始化（假设起点0是安全的）
	if road[0] != '-' { // 起点不能是陷阱
		if road[0] == '+' {
			dp[0][min(E+r, maxE)] = 0
		} else {
			dp[0][E] = 0
		}
	}

	// 动态规划
	for i := 0; i < n; i++ {
		for energy := 1; energy <= maxE; energy++ {
			if dp[i][energy] == math.MaxInt32 {
				continue
			}

			// 尝试所有可能的跳跃距离
			for jump := 1; jump <= energy; jump++ {
				nextPos := i + jump
				if nextPos >= n {
					continue
				}

				// 检查目标格子类型
				switch road[nextPos] {
				case '-': // 陷阱，不能落地
					continue
				case '0': // 普通格子
					dp[nextPos][energy] = min(dp[nextPos][energy], dp[i][energy]+1)
				case '+': // 能量板
					newEnergy := min(energy+r, maxE)
					dp[nextPos][newEnergy] = min(dp[nextPos][newEnergy], dp[i][energy]+1)
				}
			}
		}
	}

	// 查找到达终点的最小跳跃次数
	result := math.MaxInt32
	for energy := 1; energy <= maxE; energy++ {
		result = min(result, dp[n-1][energy])
	}

	if result == math.MaxInt32 {
		fmt.Println(-1)
	} else {
		fmt.Println(result)
	}
}
