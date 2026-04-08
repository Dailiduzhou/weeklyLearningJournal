package model

import (
	"context"
	"testing"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// TestFindOneByUsernameWithIndex 测试 QueryRowIndexCtx 实现的 FindOneByUsername
// 注意：这是一个示例测试，需要在有真实数据库和 Redis 环境时运行
func TestFindOneByUsernameWithIndex(t *testing.T) {
	// 配置数据库连接
	dataSource := "postgres://user:password@localhost:5432/dbname?sslmode=disable"
	conn := sqlx.NewSqlConn("postgres", dataSource)

	// 配置缓存
	cacheConf := cache.CacheConf{
		{
			RedisConf: redis.RedisConf{
				Host: "localhost:6379",
				Type: redis.NodeType,
			},
			Weight: 100,
		},
	}

	// 创建模型
	model := NewUserModel(conn, cacheConf)

	ctx := context.Background()
	username := "testuser"

	// 测试查询
	user, err := model.FindOneByUsername(ctx, username)
	if err != nil && err != ErrNotFound {
		t.Errorf("FindOneByUsername failed: %v", err)
	}

	if user != nil {
		t.Logf("Found user: %+v", user)
	} else {
		t.Log("User not found")
	}
}

// TestCacheKeyFormat 测试缓存键格式
func TestCacheKeyFormat(t *testing.T) {
	username := "alice"
	expectedIndexKey := "cache:public:user:username:alice"
	actualIndexKey := cachePublicUserUsernamePrefix + username

	if actualIndexKey != expectedIndexKey {
		t.Errorf("Cache key format mismatch: expected %s, got %s", expectedIndexKey, actualIndexKey)
	}
}
