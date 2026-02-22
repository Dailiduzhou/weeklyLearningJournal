package routes

import (
	controller "github.com/Dailiduzhou/library_manage_sys/controllers"
	"github.com/Dailiduzhou/library_manage_sys/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterBookRouters(r *gin.Engine) {
	api := r.Group("/api")

	authGroup := api.Group("/")
	authGroup.Use(middleware.AuthRequired())
	{
		authGroup.POST("/records/:id", controller.BorrowRecords)
		borrows := authGroup.Group("/borrows")
		{
			// 创建借阅记录 (借书)
			borrows.POST("", controller.BorrowBook)
			borrows.POST("/return", controller.ReturnBook)
		}

		authGroup.GET("/books", controller.GetBooks)

		adminGroup := authGroup.Group("/admin")
		adminGroup.Use(middleware.AdminRequired())
		{
			// POST /books 创建
			adminGroup.POST("/books", controller.CreateBook)
			// PUT /books/:id 更新
			adminGroup.PUT("/books/:id", controller.UpdateBook)
			// DELETE /books/:id 删除
			adminGroup.DELETE("/books/:id", controller.DeleteBooks)

			adminGroup.GET("/records", controller.GetAllBorrowRecords)

			adminGroup.POST("/records/:id", controller.BorrowRecordsByID)
		}
	}
}
