package server

import (
	"github.com/go-kratos/kratos/v2/registry"
	transgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	bv1 "svc-b/api/helloworld/v1"
)

func NewBClient(dis registry.Discovery) bv1.BClient {
	conn, err := transgrpc.DialInsecure(
		nil,
		transgrpc.WithEndpoint("discovery:///svc-b"),
		transgrpc.WithDiscovery(dis),
	)
	if err != nil {
		panic(err)
	}
	return bv1.NewBClient(conn)
}
