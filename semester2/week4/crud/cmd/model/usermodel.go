package model

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserModel = (*customUserModel)(nil)

const cachePublicUserUsernamePrefix = "cache:public:user:username:"

type (
	UserModel interface {
		userModel
		FindOneByUsername(ctx context.Context, username string) (*User, error)
	}

	customUserModel struct {
		*defaultUserModel
	}
)

func NewUserModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UserModel {
	return &customUserModel{
		defaultUserModel: newUserModel(conn, c, opts...),
	}
}

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

	// 如果 username 发生变化，同时清理新 username 的缓存
	if oldUser != nil && oldUser.Username != data.Username {
		keys = append(keys, fmt.Sprintf("%s%s", cachePublicUserUsernamePrefix, data.Username))
	}

	_, err = m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("update %s set %s where id = $1", m.table, userRowsWithPlaceHolder)
		return conn.ExecCtx(ctx, query, data.Id, data.Username, data.Password, data.Role)
	}, keys...)

	return err
}
