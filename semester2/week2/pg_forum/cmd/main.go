package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pg_forum/config"
	"pg_forum/di"
	"pg_forum/middlewares"
	"pg_forum/models"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := connectPostgres(cfg.Postgres)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}

	defer disconnectPostgres(db)

	app, err := di.InitializeApplication(cfg, db)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	gin.SetMode(cfg.Server.Mode)
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	setupRoutes(router, app)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	go func() {
		log.Printf("Server starting on port %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func connectPostgres(cfg config.PostgresConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.ConnMaxLifetime != "" {
		if duration, err := time.ParseDuration(cfg.ConnMaxLifetime); err == nil {
			sqlDB.SetConnMaxLifetime(duration)
		}
	}

	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto").Error; err != nil {
		return nil, err
	}
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS ltree").Error; err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Post{},
		&models.Comment{},
		&models.PostVote{},
		&models.CommentVote{},
	); err != nil {
		return nil, err
	}

	log.Println("Connected to Postgres")
	return db, nil
}

func disconnectPostgres(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Error getting database instance: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.Printf("Error disconnecting Postgres: %v", err)
	}
}

func setupRoutes(router *gin.Engine, app *di.App) {
	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", app.AuthController.Register)
			auth.POST("/login", app.AuthController.Login)
		}

		posts := api.Group("/posts")
		posts.Use(middlewares.JWTAuthMiddleware(app.JWTService))
		{
			posts.GET("", app.PostController.GetPosts)
			posts.POST("", app.PostController.CreatePost)
			posts.GET("/author/:authorId", app.PostController.GetPostsByAuthor)
			posts.GET("/:id", app.PostController.GetPost)
			posts.PUT("/:id", app.PostController.UpdatePost)
			posts.DELETE("/:id", app.PostController.DeletePost)

			posts.POST("/:id/comments", app.PostController.AddComment)
			posts.DELETE("/:id/comments/:commentId", app.PostController.DeleteComment)

			posts.POST("/:id/vote", app.PostController.VotePost)
			posts.POST("/:id/comments/:commentId/vote", app.PostController.VoteComment)
		}
	}
}
