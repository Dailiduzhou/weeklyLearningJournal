package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	RefreshInterval time.Duration
}

func DefaultConfig() *Config {
	return &Config{
		MaxRetries:      3,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        10 * time.Second,
		RefreshInterval: 2 * time.Second,
	}
}

func exponentialBackoff(tries int, config *Config) time.Duration {
	if tries <= 0 {
		return config.InitialDelay
	}

	delay := float64(config.InitialDelay) * math.Pow(2, float64(tries-1))

	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	return time.Duration(delay)
}

func startLockRefresh(ctx context.Context, mutex *redsync.Mutex, interval time.Duration) context.CancelFunc {
	refreshCtx, cancel := context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-refreshCtx.Done():
				log.Printf("Lock refresh stopped for %s", mutex.Name())
				return
			case <-ticker.C:
				ok, err := mutex.ExtendContext(refreshCtx)
				if err != nil {
					log.Printf("Failed to extend lock %s: %v", mutex.Name(), err)
					continue
				}
				if !ok {
					log.Printf("Lock %s no longer held", mutex.Name())
					return
				}
				log.Printf("Lock %s extended successfully", mutex.Name())
			}
		}
	}()

	return cancel
}

var ctx = context.Background()

var productInventory = 5

func buyProduct(ctx context.Context, rs *redsync.Redsync, userID int, wg *sync.WaitGroup, config *Config) {
	defer wg.Done()

	lockKey := "buy_product:1001"
	mutex := rs.NewMutex(lockKey,
		redsync.WithExpiry(8*time.Second),
		redsync.WithTries(1),
	)

	var lastErr error

	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := exponentialBackoff(attempt, config)
			log.Printf("User %d retrying in %v (attempt %d/%d)", userID, delay, attempt+1, config.MaxRetries)

			select {
			case <-ctx.Done():
				log.Printf("User %d cancelled during retry wait", userID)
				return
			case <-time.After(delay):
			}
		}

		err := mutex.LockContext(ctx)
		if err == nil {
			log.Printf("User %d acquired lock successfully (attempt %d)", userID, attempt+1)

			stopRefresh := startLockRefresh(ctx, mutex, config.RefreshInterval)
			defer stopRefresh()

			if productInventory > 0 {
				productInventory--
				log.Printf("User %d buy successfully\n", userID)
			} else {
				log.Printf("User %d failed to buy, product sold out\n", userID)
			}

			if ok, err := mutex.UnlockContext(ctx); !ok || err != nil {
				log.Printf("User %d failed to unlock: ok=%v, err=%v", userID, ok, err)
			} else {
				log.Printf("User %d unlocked successfully", userID)
			}

			return
		}

		lastErr = err
		log.Printf("User %d failed to acquire lock (attempt %d/%d): %v", userID, attempt+1, config.MaxRetries, err)
	}

	log.Printf("User %d failed to acquire lock after %d attempts: %v", userID, config.MaxRetries, lastErr)
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

	pool := goredis.NewPool(rdb)
	rs := redsync.New(pool)

	// Use default configuration
	config := DefaultConfig()

	var wg sync.WaitGroup
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go buyProduct(ctx, rs, i, &wg, config)
	}

	wg.Wait()
	fmt.Print("Completed\n")
}
