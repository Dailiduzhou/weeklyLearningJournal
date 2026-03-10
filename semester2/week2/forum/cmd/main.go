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

	"forum/config"
	"forum/di"
	"forum/middlewares"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	client, err := connectMongoDB(cfg.MongoDB.URI)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer disconnectMongoDB(client)

	db := client.Database(cfg.MongoDB.Database)

	app, err := di.InitializeApplication(cfg, db)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	gin.SetMode(cfg.Server.Mode)
	router := gin.Default()

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

func connectMongoDB(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	log.Println("Connected to MongoDB")
	return client, nil
}

func disconnectMongoDB(client *mongo.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Disconnect(ctx); err != nil {
		log.Printf("Error disconnecting from MongoDB: %v", err)
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
