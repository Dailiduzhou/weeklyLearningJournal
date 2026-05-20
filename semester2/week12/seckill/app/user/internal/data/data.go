package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"seckill/app/user/internal/biz"
	"seckill/app/user/internal/conf"
	"seckill/app/user/internal/data/db"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"github.com/google/wire"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/sync/singleflight"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewUserRepo, wire.Bind(new(biz.UserRepo), new(*UserRepo)))

// Data .
type Data struct {
	db  *sql.DB
	rdb *redis.Client
	rs  *redsync.Redsync
	q   *db.Queries
	sg  *singleflight.Group
}

// NewData .
func NewData(c *conf.Data) (*Data, func(), error) {
	sqldb, err := sql.Open("pgx", c.Database.Source)
	if err != nil {
		return nil, nil, fmt.Errorf("open postgres: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := sqldb.PingContext(pingCtx); err != nil {
		sqldb.Close()
		return nil, nil, fmt.Errorf("ping postgres: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     c.Redis.Addr,
		Password: "",
		DB:       0,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		rdb.Close()
		sqldb.Close()
		return nil, nil, fmt.Errorf("ping redis: %w", err)
	}

	pool := goredis.NewPool(rdb)
	rs := redsync.New(pool)

	cleanup := func() {
		rdb.Close()
		sqldb.Close()

		log.Info("closing the data resources")
	}
	return &Data{
		db:  sqldb,
		rdb: rdb,
		rs:  rs,
		q:   db.New(sqldb),
		sg:  &singleflight.Group{},
	}, cleanup, nil
}
