package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	// BRpc is the client config for service B, loaded from etcd
	BRpc zrpc.RpcClientConf
}
