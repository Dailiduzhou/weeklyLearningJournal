package main

import (
	"fmt"
	"log"
	"mvc/config"
	"mvc/controller"
	"mvc/middleware"
	"mvc/model"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n收到退出信号,正在保存数据...")
		// if err := persistUsers(); err != nil {
		// 	fmt.Printf("保存数据失败: %v\n", err)
		// }
		fmt.Println("数据已保存,程序退出")
		os.Exit(0)
	}()

	db := config.InitDB()

	err := db.AutoMigrate(&model.User{})
	if err != nil {
		log.Fatal("数据库迁移失败")
	}

	r := gin.Default()

	middleware.InitSession(r)

	config.InitHTML(r)

	userController := controller.NewUserController(db)
	userGroup := r.Group("/api/users")
	{
		userGroup.GET("", userController.PageCreateUser)
		userGroup.POST("", userController.CreateUser)
		userGroup.GET("/login", userController.PageLogin)
		userGroup.POST("/login", userController.Login)
		userGroup.GET("/me", userController.PageProfiles)
	}
}
