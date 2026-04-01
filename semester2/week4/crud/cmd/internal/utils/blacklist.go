package utils

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

const BlacklistPrefix = "jwt:bl:"

func AddToBlacklist(ctx context.Context, rds *redis.Redis, jti string, exp int64) error {
	now := time.Now().Unix()
	ttl := exp - now
	if ttl <= 0 {
		return nil // 已经自然过期，无需加入黑名单
	}
	// 将 jti 存入 Redis，并设置过期时间为 Token 的剩余存活时间
	return rds.SetexCtx(ctx, BlacklistPrefix+jti, "1", int(ttl))
}

// IsBlacklisted 检查 Token 是否在黑名单中
func IsBlacklisted(ctx context.Context, rds *redis.Redis, jti string) bool {
	exists, err := rds.ExistsCtx(ctx, BlacklistPrefix+jti)
	if err != nil {
		return false // 降级处理：如果 Redis 挂了，默认放行（或根据业务要求直接阻断）
	}
	return exists
}
