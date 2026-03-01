// from DeepSeek
package main

import (
	"gin-mvc-mysql/config"
	"gin-mvc-mysql/routes"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化数据库连接[citation:7]
	config.InitDB()
	defer config.CloseDB()

	// 创建Gin实例[citation:7]
	router := gin.Default()

	// 设置HTML模板（如果使用模板引擎）
	// router.LoadHTMLGlob("views/*")

	// 设置静态文件服务
	// router.Static("/static", "./static")

	// 注册路由[citation:7]
	routes.RegisterRoutes(router)

	// 启动服务器
	log.Println("🚀 服务器启动在 http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}
