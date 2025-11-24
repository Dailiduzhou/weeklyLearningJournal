package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 书籍数据结构
type Book struct {
	ID     int
	Title  string
	Author string
	Price  float64
	Rating float64
}

func main() {
	// 创建不包含日志中间件的 Gin 路由（减少干扰）
	r := gin.New()

	// 静态文件服务（可选）
	r.Static("/static", "./static")

	// 创建书籍数据
	books := []Book{
		{1, "Go 语言实战", "William Kennedy", 69.5, 4.8},
		{2, "Web 开发实战", "Brian Sletten", 89.0, 4.5},
		{3, "数据结构与算法", "Robert Sedgewick", 108.0, 4.9},
		{4, "网络编程精解", "Michael Kerrisk", 128.0, 4.7},
		{5, "分布式系统设计", "Brendan Burns", 95.5, 4.6},
	}

	// 首页路由 - 专为解析优化
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Title":       "书籍目录 - Soup 解析示例",
			"CurrentTime": time.Now().Format("2006-01-02 15:04:05"),
			"Books":       books,
		})
	})

	// 详情页路由（带参数）
	r.GET("/book/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.HTML(http.StatusOK, "detail.html", gin.H{
			"BookID":      id,
			"Title":       "书籍详情 - Soup 解析示例",
			"CurrentTime": time.Now().Format("2006-01-02 15:04:05"),
		})
	})

	// 启动服务器
	addr := ":8080"
	println("服务已启动，请访问 http://localhost" + addr)
	r.Run(addr)
}
