package svc

import (
	"svc-a/internal/config"
	"svc-b/b"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config
	// BRpc is the client for service B, used for service discovery and RPC calls
	BRpc b.B
}

func NewServiceContext(c config.Config) *ServiceContext {
	// Create zrpc client from config, which reads B service address from etcd
	bClient := zrpc.MustNewClient(c.BRpc)

	return &ServiceContext{
		Config: c,
		BRpc:   b.NewB(bClient),
	}
}
