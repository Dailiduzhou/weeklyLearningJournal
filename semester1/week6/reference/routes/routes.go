package routes

import (
	"gin-mvc-mysql/controllers"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	userController := controllers.NewUserController()

	// 用户路由组[citation:7]
	userGroup := router.Group("/api/users")
	{
		userGroup.GET("", userController.GetUsers)          // 获取所有用户
		userGroup.GET("/:id", userController.GetUser)       // 获取单个用户
		userGroup.POST("", userController.CreateUser)       // 创建用户
		userGroup.PUT("/:id", userController.UpdateUser)    // 更新用户
		userGroup.DELETE("/:id", userController.DeleteUser) // 删除用户
	}

	// 健康检查端点
	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"status":  "ok",
			"message": "服务器运行正常",
		})
	})

	// 首页重定向
	router.GET("/", func(ctx *gin.Context) {
		ctx.Redirect(302, "/api/users")
	})
}
