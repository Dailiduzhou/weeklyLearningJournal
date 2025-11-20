package main

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

var (
	ctx = context.Background()
	rdb *redis.Client
)

func initRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis地址
		Password: "",               // 无密码
		DB:       0,                // 使用默认DB
		// 其他可选参数：
		// PoolSize:     10,       // 连接池大小
		// MinIdleConns: 5,        // 最小空闲连接
		// DialTimeout:  5 * time.Second, // 连接超时
	})

	// 测试连接
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		panic(err)
	}
	fmt.Println("Redis连接成功:", pong) // 应该输出 "PONG"
}

func main() {
	initRedis()
}
