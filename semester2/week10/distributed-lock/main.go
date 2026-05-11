package main

import (
	"context"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

var unlockScript = redis.NewScript(`
	if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("delete", KEYS[1]
	else 
			return 0
`)
