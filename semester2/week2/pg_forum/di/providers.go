package di

import (
	"pg_forum/config"
	"pg_forum/controllers"
	"pg_forum/repositories"
	"pg_forum/services"

	"github.com/google/wire"
)

var RepositorySet = wire.NewSet(
	repositories.NewUserRepository,
	repositories.NewPostRepository,
)

var ServiceSet = wire.NewSet(
	services.NewJWTService,
	services.NewUserService,
	services.NewPostService,
)

var ControllerSet = wire.NewSet(
	controllers.NewAuthController,
	controllers.NewPostController,
)

func provideJWTConfig(cfg *config.Config) *config.JWTConfig {
	return &cfg.JWT
}

type App struct {
	AuthController *controllers.AuthController
	PostController *controllers.PostController
	JWTService     services.JWTService
}

func NewApp(
	authController *controllers.AuthController,
	postController *controllers.PostController,
	jwtService services.JWTService,
) *App {
	return &App{
		AuthController: authController,
		PostController: postController,
		JWTService:     jwtService,
	}
}
