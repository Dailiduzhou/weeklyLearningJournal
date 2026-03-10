//go:build wireinject
// +build wireinject

package di

import (
	"forum/config"

	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/mongo"
)

func InitializeApplication(cfg *config.Config, db *mongo.Database) (*App, error) {
	wire.Build(
		RepositorySet,
		ServiceSet,
		ControllerSet,
		provideJWTConfig,
		NewApp,
	)
	return nil, nil
}
