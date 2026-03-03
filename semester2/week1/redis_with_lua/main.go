package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// 修复: 补上了 "del" 后的逗号
var releaseScript = redis.NewScript(`
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("del", KEYS[1])
	else
		return 0
	end
`)

var refreshScript = redis.NewScript(`
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("pexpire", KEYS[1], ARGV[2])
	else
		return 0
	end
`)

func main() {
	client := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    "127.0.0.1:6379",
	})
	defer client.Close()

	ctx := context.Background()
	lockKey := "my_lock"

	lockToken := "e5141a4ca92eaf9fce9bb754c5f6c0c5"
	ttl := 5 * time.Second

	args := redis.SetArgs{
		Mode: "NX",
		TTL:  ttl,
	}

	fmt.Println("1. 尝试加锁...")
	status, err := client.SetArgs(ctx, lockKey, lockToken, args).Result()
	if err == redis.Nil {
		fmt.Println("锁已被占用，加锁失败")
		return
	} else if err != nil {
		fmt.Printf("加锁发生错误: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n", status)
	fmt.Println("已成功加锁")

	time.Sleep(2 * time.Second)

	fmt.Println("\n2. 尝试刷新锁的过期时间...")

	// 修复: 传入了 lockToken 作为 ARGV[1], ttl.Milliseconds() 作为 ARGV[2]
	refreshRes, err := refreshScript.Run(ctx, client, []string{lockKey}, lockToken, ttl.Milliseconds()).Int()
	if err != nil {
		fmt.Printf("刷新锁发生错误: %v\n", err)
	} else if refreshRes == 1 {
		fmt.Println("锁刷新成功")
	} else {
		fmt.Println("锁刷新失败 (Token不匹配或锁已过期)")
	}

	time.Sleep(1 * time.Second)

	fmt.Println("\n3. 尝试释放锁...")
	releaseRes, err := releaseScript.Run(ctx, client, []string{lockKey}, lockToken).Int()
	if err != nil {
		fmt.Printf("释放锁发生错误: %v\n", err)
	} else if releaseRes == 1 {
		fmt.Println("锁释放成功")
	} else {
		fmt.Println("锁释放失败 (Token不匹配或锁已过期)")
	}
}
