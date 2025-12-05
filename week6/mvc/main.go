// @title          Gin用户管理API
// @version        1.0
// @description    这是一个基于Gin+GORM的用户管理系统API文档，包含用户注册、登录、信息管理等功能。
// @host           localhost:8080
// @BasePath       /api
// @securityDefinitions.apikey SessionAuth
// @in header
// @name Cookie
// @description 请从登录响应中获取Session Cookie并在此处携带（基于gin-contrib/sessions）
package main

import (
	"context"
	"log"
	"mvc/config"
	"mvc/controller"
	"mvc/middleware"
	"mvc/model"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "mvc/docs"

	"github.com/gin-gonic/gin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	db := config.InitDB()

	err := db.AutoMigrate(&model.User{})
	if err != nil {
		log.Fatal("数据库迁移失败")
	}

	r := gin.Default()

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	if err := middleware.InitSession(r); err != nil {
		log.Printf("会话创建失败: %q", err)
	}

	config.InitHTML(r)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	userController := controller.NewUserController(db)
	userGroup := r.Group("/api/users")
	{
		userGroup.GET("", userController.PageCreateUser)
		userGroup.POST("", userController.CreateUser)
		userGroup.GET("/login", userController.PageLogin)
		userGroup.POST("/login", userController.Login)
		userGroup.GET("/me", userController.PageProfiles)
		userGroup.POST("/logout", userController.Logout)
		userGroup.GET("/profiles", userController.PageChangeProfile)
		userGroup.PUT("/profiles", userController.ChangeProfile)
		userGroup.GET("/password", userController.PageChangePassword)
		userGroup.PUT("/password", userController.ChangePassword)
	}

	adminGroup := r.Group("/api/admin")
	{
		adminGroup.GET("/users", userController.UsersData)
	}

	log.Println("服务器启动在 http://localhost:8080")
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("服务器强制关闭:", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.Close()
	log.Println("服务器已优雅退出")
}
