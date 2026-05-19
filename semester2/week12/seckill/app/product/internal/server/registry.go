package server

import (
	"seckill/app/product/internal/conf"

	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// NewEtcdClient 创建 etcd 客户端
func NewEtcdClient(c *conf.Registry) *clientv3.Client {
	client, err := clientv3.New(clientv3.Config{
		Endpoints: c.Etcd.Endpoints,
	})
	if err != nil {
		panic(err)
	}
	return client
}

// NewRegistrar 创建 Kratos 服务注册器
func NewRegistrar(client *clientv3.Client) registry.Registrar {
	return etcd.New(client)
}

// NewDiscovery 创建 Kratos 服务发现器 (如果你同时需要调用其他服务)
func NewDiscovery(client *clientv3.Client) registry.Discovery {
	return etcd.New(client)
}
