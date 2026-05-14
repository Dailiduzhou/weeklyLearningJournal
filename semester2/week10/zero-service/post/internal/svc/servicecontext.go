package svc

import (
	"github.com/Dailiduzhou/weeklyLearningJournal/semester2/week10/zero-service/post/internal/config"
	"github.com/Dailiduzhou/weeklyLearningJournal/semester2/week10/zero-service/post/internal/model"
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

	redis := redis.MustNewRedis(c.BizRedis)

	return &ServiceContext{
		Config:    c,
		BizRedis:  redis,
		PostModel: model.NewPostModel(conn, c.Cache),
	}
}
