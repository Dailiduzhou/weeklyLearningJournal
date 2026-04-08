package svc

import (
	"github.com/Dailiduzhou/weeklyLearningJournal/semester2/week6/micro_svc/svc-a/internal/config"
	"github.com/Dailiduzhou/weeklyLearningJournal/semester2/week6/micro_svc/svc-b/b"
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
