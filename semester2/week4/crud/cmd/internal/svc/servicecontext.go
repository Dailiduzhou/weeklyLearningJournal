// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"crud/cmd/internal/config"
	"crud/cmd/internal/middleware"
	"github.com/zeromicro/go-zero/rest"
)

type ServiceContext struct {
	Config          config.Config
	AdminMiddleware rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:          c,
		AdminMiddleware: middleware.NewAdminMiddleware().Handle,
	}
}
