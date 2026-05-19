package server

import (
	"context"

	uv1 "seckill/api/user/v1"

	"github.com/go-kratos/kratos/v2/registry"
	transgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
)

func NewUserClient(dis registry.Discovery) uv1.UserClient {
	conn, err := transgrpc.DialInsecure(
		context.Background(),
		transgrpc.WithEndpoint("discovery:///user"),
		transgrpc.WithDiscovery(dis),
	)
	if err != nil {
		panic(err)
	}
	return uv1.NewUserClient(conn)
}
