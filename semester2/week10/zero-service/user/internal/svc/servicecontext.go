package svc

import (
	"gihub.com/Dailiduzhou/weeklyLearningJournal/semester2/week10/zero-service/post"
	"github.com/Dailiduzhou/weeklyLearningJournal/semester2/week10/zero-service/user/internal/config"
	"github.com/Dailiduzhou/weeklyLearningJournal/semester2/week10/zero-service/user/internal/model"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config    config.Config
	UserModel model.UserModel
	BizRedis  *redis.Redis
	PostRpc   post.Post
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.DB.DataSource)

	redis := redis.MustNewRedis(c.BizRedis)

	postClient := zrpc.MustNewClient(c.PostRpc)
	return &ServiceContext{
		Config:    c,
		UserModel: model.NewUserModel(conn, c.Cache),
		BizRedis:  redis,
	}
}
