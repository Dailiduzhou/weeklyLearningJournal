# go-zero 自定义字段查询与缓存最佳实践

本文档基于本地 go-zero v1.10.1 源码，介绍如何在 go-zero 生成的 CRUD 之外使用其他字段查询，并充分利用缓存机制。

---

## 一、go-zero 提供的两种自定义查询方式

根据 `core/stores/sqlc/cachedsql.go` 源码，go-zero 提供了两种处理自定义字段查询的方法：

### 方式一：简单自定义查询（QueryRowCtx）

适用于：字段唯一但查询频率不高的场景

```go
func (m *customUserModel) FindOneByUsername(ctx context.Context, username string) (*User, error) {
    cacheKey := fmt.Sprintf("cache:user:username:%s", username)
    var resp User
    
    // QueryRowCtx 会自动处理缓存：先查缓存，未命中则执行 query 函数，并写入缓存
    err := m.QueryRowCtx(ctx, &resp, cacheKey, func(ctx context.Context, conn sqlx.SqlConn, v any) error {
        query := fmt.Sprintf("select %s from %s where username = $1 limit 1", userRows, m.table)
        return conn.QueryRowCtx(ctx, v, query, username)
    })
    
    switch err {
    case nil:
        return &resp, nil
    case sqlx.ErrNotFound:
        return nil, ErrNotFound
    default:
        return nil, err
    }
}
```

**特点：**
- ✅ 实现简单，直接缓存完整数据
- ❌ 数据更新时需要清理多个缓存键（id + username）

---

### 方式二：索引到主键的映射查询（QueryRowIndexCtx）⭐ 推荐

适用于：高频查询的唯一索引字段（如手机号、邮箱）

```go
func (m *customUserModel) FindOneByPhone(ctx context.Context, phone string) (*User, error) {
    var resp User
    indexKey := fmt.Sprintf("cache:user:phone:%s", phone)
    
    err := m.QueryRowIndexCtx(ctx, &resp, indexKey,
        // keyer: 根据主键生成主键缓存键
        func(primary any) string {
            return fmt.Sprintf("cache:user:id:%v", primary)
        },
        // indexQuery: 通过索引查主键（只返回主键，不返回完整数据）
        func(ctx context.Context, conn sqlx.SqlConn, v any) (any, error) {
            // 1. 必须查出全量数据并解析到外层传进来的 v 中！
            err := conn.QueryRowCtx(ctx, v, "select * from user where phone = $1 limit 1", phone)
            if err != nil {
              return nil, err
            }
            // 2. 将 v 断言为 User 类型，返回其主键 ID
            return v.(*User).Id, nil
        },
        // primaryQuery: 通过主键查完整数据
        func(ctx context.Context, conn sqlx.SqlConn, v, primary any) error {
            return conn.QueryRowCtx(ctx, v,
                "select * from user where id = $1 limit 1", primary)
        },
    )
    return &resp, err
}
```

**源码核心逻辑**（`cachedsql.go:152-180`）：

```go
func (cc CachedConn) QueryRowIndexCtx(ctx context.Context, v any, key string,
    keyer func(primary any) string, indexQuery IndexQueryCtxFn,
    primaryQuery PrimaryQueryCtxFn) error {
    
    var primaryKey any
    var found bool

    // 1. 先查索引缓存，获取主键
    if err := cc.cache.TakeWithExpireCtx(ctx, &primaryKey, key,
        func(val any, expire time.Duration) (err error) {
            // 2. 缓存未命中，查数据库获取主键
            primaryKey, err = indexQuery(ctx, cc.db, v)
            if err != nil {
                return
            }
            found = true
            // 3. 同时缓存主键对应的数据（增加5秒安全间隔）
            return cc.cache.SetWithExpireCtx(ctx, keyer(primaryKey), v,
                expire+cacheSafeGapBetweenIndexAndPrimary)
        }); err != nil {
        return err
    }

    if found {
        return nil
    }

    // 4. 有索引缓存，通过主键查数据
    return cc.cache.TakeCtx(ctx, v, keyer(primaryKey), func(v any) error {
        return primaryQuery(ctx, cc.db, v, primaryKey)
    })
}
```

**关键常量：**

```go
const cacheSafeGapBetweenIndexAndPrimary = time.Second * 5  // 索引与主键缓存过期时间差
```

**特点：**
- ✅ 只缓存主键数据，索引只存映射关系，节省内存
- ✅ 数据更新时只需清理主键缓存
- ✅ 索引缓存过期时间比主键缓存短5秒，避免并发问题

---

## 二、缓存键命名规范

| 查询类型 | 缓存键格式 | 示例 |
|---------|-----------|------|
| 主键查询 | `cache:{table}:id:{primary}` | `cache:user:id:123` |
| 唯一索引 | `cache:{table}:{field}:{value}` | `cache:user:username:alice` |
| 联合唯一 | `cache:{table}:{f1}:{v1}:{f2}:{v2}` | `cache:user:phone:138xxx:email:a@b.com` |

---

## 三、充分利用缓存的最佳实践

### 3.1 完整示例：使用 QueryRowIndexCtx 实现自定义查询

在你的项目中，如果要添加更多字段查询（如邮箱），建议使用 QueryRowIndexCtx：

```go
type (
    UserModel interface {
        userModel
        FindOneByUsername(ctx context.Context, username string) (*User, error)
        FindOneByEmail(ctx context.Context, email string) (*User, error)  // 新增
    }
)

// 使用 QueryRowIndexCtx 实现 - 更高效的缓存利用
func (m *customUserModel) FindOneByEmail(ctx context.Context, email string) (*User, error) {
    var resp User
    indexKey := fmt.Sprintf("cache:user:email:%s", email)
    
    err := m.QueryRowIndexCtx(ctx, &resp, indexKey,
        func(primary any) string {
            return fmt.Sprintf("%s%v", cachePublicUserIdPrefix, primary)
        },
        func(ctx context.Context, conn sqlx.SqlConn, v any) (any, error) {
            query := fmt.Sprintf("select id from %s where email = $1 limit 1", m.table)
            var result struct{ Id int64 }
            if err := conn.QueryRowCtx(ctx, &result, query, email); err != nil {
                return nil, err
            }
            return result.Id, nil
        },
        func(ctx context.Context, conn sqlx.SqlConn, v, primary any) error {
            query := fmt.Sprintf("select %s from %s where id = $1 limit 1", userRows, m.table)
            return conn.QueryRowCtx(ctx, v, query, primary)
        },
    )
    
    switch err {
    case nil:
        return &resp, nil
    case sqlc.ErrNotFound:
        return nil, ErrNotFound
    default:
        return nil, err
    }
}
```

---

### 3.2 数据更新时同步清理索引缓存

使用 `QueryRowIndexCtx` 时，索引缓存和主键缓存是分开的。当数据更新时，需要同时清理这两部分缓存。

**完整实现示例（含 Update 重写）：**

```go
const cachePublicUserUsernamePrefix = "cache:public:user:username:"

func (m *customUserModel) FindOneByUsername(ctx context.Context, username string) (*User, error) {
    var resp User
    indexKey := fmt.Sprintf("%s%s", cachePublicUserUsernamePrefix, username)

    err := m.QueryRowIndexCtx(ctx, &resp, indexKey,
        // keyer: 根据主键生成主键缓存键
        func(primary any) string {
            return fmt.Sprintf("%s%v", cachePublicUserIdPrefix, primary)
        },
        // indexQuery: 通过索引查主键（只返回主键，不返回完整数据）
        func(ctx context.Context, conn sqlx.SqlConn, v any) (any, error) {
            query := fmt.Sprintf("select id from %s where username = $1 limit 1", m.table)
            var result struct{ Id int64 }
            if err := conn.QueryRowCtx(ctx, &result, query, username); err != nil {
                return nil, err
            }
            return result.Id, nil
        },
        // primaryQuery: 通过主键查完整数据
        func(ctx context.Context, conn sqlx.SqlConn, v, primary any) error {
            query := fmt.Sprintf("select %s from %s where id = $1 limit 1", userRows, m.table)
            return conn.QueryRowCtx(ctx, v, query, primary)
        },
    )

    switch err {
    case nil:
        return &resp, nil
    case sqlc.ErrNotFound:
        return nil, ErrNotFound
    default:
        return nil, err
    }
}

// Update 重写 Update 方法，同时清理 username 索引缓存
func (m *customUserModel) Update(ctx context.Context, data *User) error {
    // 先查询旧数据，获取旧的 username
    oldUser, err := m.FindOne(ctx, data.Id)
    if err != nil && err != ErrNotFound {
        return err
    }

    publicUserIdKey := fmt.Sprintf("%s%v", cachePublicUserIdPrefix, data.Id)
    keys := []string{publicUserIdKey}

    // 如果旧数据存在，添加旧 username 的缓存键
    if oldUser != nil {
        keys = append(keys, fmt.Sprintf("%s%s", cachePublicUserUsernamePrefix, oldUser.Username))
    }

    // 如果 username 发生变化，同时清理新 username 的缓存（防止脏数据）
    if oldUser != nil && oldUser.Username != data.Username {
        keys = append(keys, fmt.Sprintf("%s%s", cachePublicUserUsernamePrefix, data.Username))
    }

    _, err = m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
        query := fmt.Sprintf("update %s set %s where id = $1", m.table, userRowsWithPlaceHolder)
        return conn.ExecCtx(ctx, query, data.Id, data.Username, data.Password, data.Role)
    }, keys...)

    return err
}
```

**关键点：**
- 使用 `ExecCtx` 执行更新，它会自动清理指定的缓存键
- 清理顺序：主键缓存 + 旧 username 索引缓存 + 新 username 索引缓存（如有变更）
- 如果 username 可能改变，需要同时清理新旧两个 username 对应的缓存

---

### 3.3 缓存防护机制（go-zero 内置）

go-zero 缓存节点 `core/stores/cache/cachenode.go` 内置了三重防护：

| 问题 | 机制 | 实现 |
|------|------|------|
| **缓存击穿** | SingleFlight | 相同 key 的并发查询只执行一次 |
| **缓存穿透** | 空值缓存 | 数据库查询不到时缓存 `*` 占位符（默认1分钟） |
| **缓存雪崩** | 过期时间抖动 | 过期时间 = [0.95, 1.05] × 设定值 |

**源码参考：**

```go
// 空值缓存
const notFoundPlaceholder = "*"

func (c cacheNode) setCacheWithNotFound(ctx context.Context, key string) error {
    seconds := int(math.Ceil(c.aroundDuration(c.notFoundExpiry).Seconds()))
    _, err := c.rds.SetnxExCtx(ctx, key, notFoundPlaceholder, seconds)
    return err
}

// 过期时间抖动
const expiryDeviation = 0.05  // [0.95, 1.05] * seconds

func (c cacheNode) aroundDuration(duration time.Duration) time.Duration {
    return c.unstableExpiry.AroundDuration(duration)
}
```

---

### 3.4 缓存配置优化

```go
// 单节点配置
cacheConf := cache.CacheConf{
    {
        RedisConf: redis.RedisConf{
            Host: "localhost:6379",
            Type: redis.NodeType,  // 或 redis.ClusterType
        },
        Weight: 100,
    },
}

// 集群配置（一致性哈希）
cacheConf := cache.CacheConf{
    {RedisConf: redis.RedisConf{Host: "node1:6379", Type: redis.NodeType}, Weight: 100},
    {RedisConf: redis.RedisConf{Host: "node2:6379", Type: redis.NodeType}, Weight: 100},
    {RedisConf: redis.RedisConf{Host: "node3:6379", Type: redis.NodeType}, Weight: 100},
}

// 自定义缓存选项
model := NewUserModel(conn, cacheConf, 
    cache.WithExpiry(time.Hour*24),           // 数据缓存24小时（默认7天）
    cache.WithNotFoundExpiry(time.Minute*5),  // 空值缓存5分钟（默认1分钟）
)
```

---

### 3.5 缓存失效重试机制

根据 `core/stores/cache/cleaner.go`，go-zero 提供了异步延迟重试清理机制：

```go
// 重试间隔：1s -> 5s -> 1min -> 5min -> 1h
func nextDelay(delay time.Duration) (time.Duration, bool) {
    switch delay {
    case time.Second:
        return time.Second * 5, true
    case time.Second * 5:
        return time.Minute, true
    case time.Minute:
        return time.Minute * 5, true
    case time.Minute * 5:
        return time.Hour, true
    default:
        return 0, false
    }
}
```

---

### 3.6 缓存统计监控

go-zero 内置缓存统计，每分钟输出日志：

```
dbcache(sqlc) - qpm: 1000, hit_ratio: 85.0%, hit: 850, miss: 150, db_fails: 0
```

**监控指标：**
- `Total`：总请求数
- `Hit`：命中数
- `Miss`：未命中数
- `DbFails`：数据库失败数
- `hit_ratio`：命中率

---

## 四、总结对比

| 特性 | QueryRowCtx（方式一） | QueryRowIndexCtx（方式二） |
|------|----------------------|---------------------------|
| **缓存内容** | 直接缓存完整数据 | 缓存"索引→主键"映射 + 主键→数据 |
| **内存占用** | 高（每条记录多份缓存） | 低（只有主键数据 + 映射） |
| **更新复杂度** | 高（需清理多个缓存键） | 低（只需清理主键缓存） |
| **适用场景** | 低频查询字段 | 高频查询的唯一索引（手机号/邮箱） |
| **推荐程度** | ⭐⭐ | ⭐⭐⭐⭐⭐ |

---

## 五、建议

1. **对于手机号、邮箱等高频查询的唯一索引字段**，使用 `QueryRowIndexCtx`
2. **对于低频查询字段**，可以使用简单的 `QueryRowCtx`
3. **列表查询不要使用缓存**（go-zero 明确建议使用 `QueryRowsNoCache`）
4. **缓存键设计**：使用 `cache:{table}:{field}:{value}` 的层级结构，便于管理和清理
5. **更新数据时**：使用 `ExecCtx` 会自动清理相关缓存

---

## 参考源码位置

| 文件路径 | 说明 |
|---------|------|
| `core/stores/sqlc/cachedsql.go` | SQL 缓存连接层，包含 QueryRowIndexCtx 实现 |
| `core/stores/cache/cachenode.go` | 缓存节点核心实现，包含防护机制 |
| `core/stores/cache/cleaner.go` | 缓存清理和重试机制 |
| `core/stores/cache/cachestat.go` | 缓存统计监控 |

---

*文档基于 go-zero v1.10.1 源码整理*
