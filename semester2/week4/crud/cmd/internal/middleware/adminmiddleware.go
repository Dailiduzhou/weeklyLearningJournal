// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package middleware

import (
	"net/http"

	"crud/cmd/internal/utils"

	"github.com/zeromicro/go-zero/core/errorx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest/httpx"
)

type AdminMiddleware struct {
	Redis *redis.Redis
}

func NewAdminMiddleware(rds *redis.Redis) *AdminMiddleware {
	return &AdminMiddleware{Redis: rds}
}

func (m *AdminMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if jtiVal := ctx.Value("jti"); jtiVal != nil {
			jti, ok := jtiVal.(string)
			if !ok {
				httpx.Error(w, errorx.Wrapf(nil, "jti不是字符串"))
				return
			}

			if utils.IsBlacklisted(ctx, m.Redis, jti) {
				httpx.Error(w, errorx.Wrapf(nil, "Token 已失效"))
				return
			}
		}
		roleVal := ctx.Value("role")
		if roleVal == nil || roleVal.(string) != "admin" {
			httpx.Error(w, errorx.Wrapf(nil, "无权访问"))
			return
		}

		next(w, r)
	}
}
