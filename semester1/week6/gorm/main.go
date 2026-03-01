package main

import (
	"gorm-mvc-demo/config"
	"gorm-mvc-demo/controller"
	"gorm-mvc-demo/model"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化数据库
	db := config.InitDB()

	// 自动迁移数据库表
	err := db.AutoMigrate(&model.User{})
	if err != nil {
		log.Fatal("数据库迁移失败:", err)
	}

	// 创建路由
	r := gin.Default()

	// 用户路由
	userController := controller.NewUserController(db)
	userGroup := r.Group("/api/users")
	{
		userGroup.POST("", userController.CreateUser)
		userGroup.GET("", userController.GetAllUsers)
		userGroup.GET("/:id", userController.GetUserByID)
		userGroup.PUT("/:id", userController.UpdateUser)
		userGroup.DELETE("/:id", userController.DeleteUser)
	}

	// 启动服务器
	log.Println("服务器启动在 http://localhost:8080")
	r.Run(":8080")
}
