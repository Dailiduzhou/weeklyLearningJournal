//go:build wireinject
// +build wireinject

package di

import (
	"pg_forum/config"

	"github.com/google/wire"
	"gorm.io/gorm"
)

func InitializeApplication(cfg *config.Config, db *gorm.DB) (*App, error) {
	wire.Build(
		RepositorySet,
		ServiceSet,
		ControllerSet,
		provideJWTConfig,
		NewApp,
	)
	return nil, nil
}
