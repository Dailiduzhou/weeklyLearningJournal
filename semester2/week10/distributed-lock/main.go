package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

const unlockScript = `
	if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
	else 
			return 0
	end`

const refreshScript = `
	if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("expire", KEYS[1], ARGV[2])
	else
			return 0
	end`

type RedisLock struct {
	client *redis.Client
	key    string
	value  string
	expire time.Duration
}

func NewRedisLock(client *redis.Client, key string, expire time.Duration) *RedisLock {
	return &RedisLock{
		client: client,
		key:    key,
		value:  uuid.New().String(),
		expire: expire,
	}
}

func (l *RedisLock) TryLock(ctx context.Context) (bool, error) {
	success, err := l.client.SetNX(ctx, l.key, l.value, l.expire).Result()
	if err != nil {
		return false, err
	}
	return success, nil
}

func (l *RedisLock) Unlock(ctx context.Context) (bool, error) {
	result, err := l.client.Eval(ctx, unlockScript, []string{l.key}, l.value).Result()
	if err != nil {
		return false, err
	}

	intResult, ok := result.(int64)
	if !ok {
		return false, nil
	}
	return intResult == 1, nil
}

func (l *RedisLock) Refresh(ctx context.Context) error {
	return l.client.Eval(ctx, refreshScript, []string{l.key}, l.value, int(l.expire.Seconds())).Err()
}

var productInventory = 5

func buyProduct(client *redis.Client, userID int, wg *sync.WaitGroup) {
	defer wg.Done()

	lockKey := "lock:product:1001"
	maxRetries := 30
	retryInterval := 10 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		lock := NewRedisLock(client, lockKey, 5*time.Second)

		acquired, err := lock.TryLock(ctx)
		if err != nil {
			log.Printf("User %d Sys failed, err: %v\n", userID, err)
			return
		}

		if !acquired {
			time.Sleep(retryInterval)
			continue
		}

		stopRefresh := make(chan struct{})
		go func() {
			ticker := time.NewTicker(lock.expire / 3)
			defer ticker.Stop()
			for {
				select {
				case <-stopRefresh:
					return
				case <-ticker.C:
					if err := lock.Refresh(ctx); err != nil {
						log.Printf("User %d refresh lock error: %v\n", userID, err)
					}
				}
			}
		}()

		time.Sleep(20 * time.Millisecond)

		if productInventory > 0 {
			productInventory--
			fmt.Printf("User %d buy successfully\n", userID)
		} else {
			fmt.Printf("User %d, product sold out\n", userID)
		}

		close(stopRefresh)
		ok, err := lock.Unlock(ctx)
		if err != nil {
			log.Printf("User %d unlock error: %v\n", userID, err)
		} else if !ok {
			log.Printf("User %d unlock failed, lock may have expired\n", userID)
		}
		return
	}

	fmt.Printf("User %d failed to acquire lock after %d retries\n", userID, maxRetries)
}

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("连接 Redis 失败: %v", err)
	}
	defer rdb.Close()

	fmt.Printf("当前总库存: %d\n", productInventory)

	var wg sync.WaitGroup
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go buyProduct(rdb, i, &wg)
	}

	wg.Wait()
	fmt.Print("Compeleted\n")
}
