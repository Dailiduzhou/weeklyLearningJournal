// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"crud/cmd/internal/config"
	"crud/cmd/internal/middleware"
	"crud/cmd/model"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/rest"
)

type ServiceContext struct {
	Config          config.Config
	AdminMiddleware rest.Middleware

	UserModel model.UserModel
	Redis     *redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewSqlConn("postgres", c.DB.DataSource)

	bizRedis := redis.MustNewRedis(c.BizRedis)

	return &ServiceContext{
		Config:          c,
		UserModel:       model.NewUserModel(conn, c.Cache),
		AdminMiddleware: middleware.NewAdminMiddleware(bizRedis).Handle,
		Redis:           bizRedis,
	}
}
