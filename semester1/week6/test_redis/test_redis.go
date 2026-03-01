package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

func main() {
	ctx := context.Background()

	// 测试连接配置
	configs := []struct {
		name     string
		addr     string
		password string
	}{
		{"localhost无密码", "localhost:6379", ""},
		{"127.0.0.1无密码", "127.0.0.1:6379", ""},
		{"docker内部IP", "172.17.0.2:6379", ""}, // 需要先获取实际IP
	}

	for _, cfg := range configs {
		log.Printf("测试连接: %s", cfg.name)
		if testConnection(ctx, cfg.addr, cfg.password) {
			log.Printf("✓ %s 连接成功", cfg.name)
			return
		}
	}

	log.Fatal("所有连接测试都失败")
}

func testConnection(ctx context.Context, addr, password string) bool {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// 测试连接
	_, err := client.Ping(ctx).Result()
	if err != nil {
		log.Printf("  × %s 连接失败: %v", addr, err)
		return false
	}

	// 测试读写
	key := fmt.Sprintf("test:%d", time.Now().Unix())
	value := "hello_redis"

	err = client.Set(ctx, key, value, 5*time.Second).Err()
	if err != nil {
		log.Printf("  × %s 写入失败: %v", addr, err)
		return false
	}

	gotValue, err := client.Get(ctx, key).Result()
	if err != nil {
		log.Printf("  × %s 读取失败: %v", addr, err)
		return false
	}

	if gotValue != value {
		log.Printf("  × %s 数据不匹配", addr)
		return false
	}

	return true
}
