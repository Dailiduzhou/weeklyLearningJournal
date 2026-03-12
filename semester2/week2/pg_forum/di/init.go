package di

import (
	"pg_forum/config"
	"pg_forum/controllers"
	"pg_forum/repositories"
	"pg_forum/services"

	"gorm.io/gorm"
)

func InitializeApplication(cfg *config.Config, db *gorm.DB) (*App, error) {
	userRepo := repositories.NewUserRepository(db)
	postRepo := repositories.NewPostRepository(db)

	jwtConfig := provideJWTConfig(cfg)
	jwtService := services.NewJWTService(jwtConfig)

	userService := services.NewUserService(userRepo, jwtService)
	postService := services.NewPostService(postRepo)

	authController := controllers.NewAuthController(userService)
	postController := controllers.NewPostController(postService)

	app := NewApp(authController, postController, jwtService)
	return app, nil
}
