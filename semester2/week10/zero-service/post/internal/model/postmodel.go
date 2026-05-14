package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ PostModel = (*customPostModel)(nil)

type (
	// PostModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPostModel.
	PostModel interface {
		postModel
		FindMany(ctx context.Context) ([]*Post, error)
		FindManyByUserID(ctx context.Context, userID int64) ([]*Post, error)
	}

	customPostModel struct {
		*defaultPostModel
	}
)

var cachePostUserIdPrefix = "cache:post:userid"

// NewPostModel returns a model for the database table.
func NewPostModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) PostModel {
	return &customPostModel{
		defaultPostModel: newPostModel(conn, c, opts...),
	}
}

func (m *defaultPostModel) FindMany(ctx context.Context) ([]*Post, error) {
	var resp []*Post
	query := fmt.Sprintf("select %s from %s", postRows, m.table)
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query)
	switch err {
	case nil:
		return resp, nil
	case sqlc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (m *defaultPostModel) FindManyByUserID(ctx context.Context, userID int64) ([]*Post, error) {
	var resp []*Post
	query := fmt.Sprintf("select %s from %s where `userid` = ?", postRows, m.table)
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, userID)
	switch err {
	case nil:
		return resp, nil
	case sqlc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}
