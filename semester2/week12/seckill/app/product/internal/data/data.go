package data

import (
	"context"
	"database/sql"

	uv1 "seckill/api/user/v1"
	"seckill/app/product/internal/conf"
	"seckill/app/product/internal/data/db"
	"seckill/app/product/internal/server"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"github.com/google/wire"
	"golang.org/x/sync/singleflight"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewProductRepo, server.NewUserClient)

// Data .
type Data struct {
	db         *sql.DB
	rdb        *redis.Client
	rs         *redsync.Redsync
	q          *db.Queries
	sg         *singleflight.Group
	userclient *uv1.UserClient
}

// NewData .
func NewData(c *conf.Data, userclient *uv1.UserClient) (*Data, func(), error) {
	sqldb, err := sql.Open("pgx", c.Database.Source)
	if err != nil {
		panic("error connecting db")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     c.Redis.Addr,
		Password: "",
		DB:       0,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		panic("error ping redis")
	}

	pool := goredis.NewPool(rdb)
	rs := redsync.New(pool)

	cleanup := func() {
		rdb.Close()
		sqldb.Close()

		log.Info("closing the data resources")
	}
	return &Data{
		db:         sqldb,
		rdb:        rdb,
		rs:         rs,
		q:          db.New(sqldb),
		sg:         &singleflight.Group{},
		userclient: userclient,
	}, cleanup, nil
}
