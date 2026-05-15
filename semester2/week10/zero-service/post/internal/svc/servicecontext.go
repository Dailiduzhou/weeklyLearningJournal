package svc

import (
	"zero-service/post/internal/config"
	"zero-service/post/internal/model"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config    config.Config
	BizRedis  *redis.Redis
	PostModel model.PostModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.DB.DataSource)

	redis := redis.MustNewRedis(c.Redis.RedisConf)

	return &ServiceContext{
		Config:    c,
		BizRedis:  redis,
		PostModel: model.NewPostModel(conn, c.Cache),
	}
}
