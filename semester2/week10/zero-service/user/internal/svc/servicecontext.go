package svc

import (
	"zero-service/post/postclient"
	"zero-service/user/internal/config"
	"zero-service/user/internal/model"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config     config.Config
	BizRedis   *redis.Redis
	UserModel  model.UserModel
	PostClient postclient.Post
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.DB.DataSource)

	redis := redis.MustNewRedis(c.Redis.RedisConf)
	postClient := zrpc.MustNewClient(c.PostRpc)

	return &ServiceContext{
		Config:     c,
		BizRedis:   redis,
		UserModel:  model.NewUserModel(conn, c.Cache),
		PostClient: postclient.NewPost(postClient),
	}
}
