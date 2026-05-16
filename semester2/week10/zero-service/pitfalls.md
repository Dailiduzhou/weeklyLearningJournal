# 记录一些搞这个项目遇到的坑

## 创建微服务
```bash
goctl rpc new XXX
```
会自动查找父目录中的`go.mod`，并且根据那个`go.mod`进行import代码的生成。

注意要创建自己针对这个项目的`go.mod`。

## 更新根据`.proto`文件生成的代码

例如：
```bash
goctl rpc new user
cd user
# 更改了.proto
goctl rpc protoc -go_out=. -go-grpc_out=. -zrpc_out=.
```

## Redis字段冲突

`zrpc.RpcServerConf`中有`Redis`字段，所以在`/etc/XXX.yaml`中需要用`BizRedis`来避免冲突


## grpc互相调用
即使定义是是用**XXX**的，在`goctl rpc protoc`生成之后，包名可能会出现`client`后缀。

## 错误处理
官方认定的最佳实践[go-zero-looklook](https://github.com/Mikaelemmmm/go-zero-looklook)

其中使用的就是`github.com/pkg/errors`作为错误处理包。
internal/logic层返回的error一般都是使用`errors.Wrapf`包裹。

# `go-zero`使用要点

## /etc/XXX.yaml和internal/config/config.go结构体字段有严格的对应关系

例如
/etc/user.yaml
```yaml
Name: user.rpc
ListenOn: 0.0.0.0:8080
Mode: dev
Etcd:
  Hosts:
  - etcd:2379
  Key: user.rpc

BizRedis:
  Host: redis:6379
  Type: node
  Pass: ""

DB:
  DataSource: root:123456@tcp(mysql:3306)/user?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai
Cache:
  - Host: redis:6379
    Pass: ""

PostRpc:
  Etcd:
    Hosts:
    - etcd:2379
    Key: post.rpc

```
/internal/config/config.go
```go
package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf

	DB struct {
		DataSource string
	}

	BizRedis redis.RedisConf
	Cache cache.CacheConf

	PostRpc zrpc.RpcClientConf
}
```

## 连接数据库、Redis以及其他gRpc客户端都需要注册到svc中

```go
package svc

import (
	"zero-service/post/postclient"
	"zero-service/user/internal/config"
	"zero-service/user/internal/model"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config     config.Config
	BizRedis   *redis.Redis
	UserModel  model.UserModel
	PostClient postclient.Post
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.DB.DataSource)

	redis := redis.MustNewRedis(c.BizRedis)
	postClient := zrpc.MustNewClient(c.PostRpc)

	return &ServiceContext{
		Config:     c,
		BizRedis:   redis,
		UserModel:  model.NewUserModel(conn, c.Cache),
		PostClient: postclient.NewPost(postClient),
	}
}
```

## grpc互相调用
首先通过服务注册和发现，把需要暴露的grpc注册到`consul`，`etcd`等地方。

以`etcd`为例：
### 配置文件
/etc/user.yaml
```yaml
Name: user.rpc
ListenOn: 0.0.0.0:8080
Etcd:
  Hosts:
  - 127.0.0.1:2379 // <--- 或者在docker中设置为etcd:2379

# ...

PostRpc:
  Etcd:
    Hosts:
    - 127.0.0.1:2379 // <--- 或者在docker中设置为etcd:2379
```
首先把需要调用的grpc客户端放到yaml中。

### 修改Config字段
/inernal/config/config.go
```go
type Config struct {
	zrpc.RpcServerConf
    // ...
	PostRpc  zrpc.RpcClientConf
}
```

### 注册到服务ctx
/internal/svc/servicecontext.go
```go
package svc

import (
	"zero-service/post/postclient"
	"zero-service/user/internal/config"
	"zero-service/user/internal/model"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config     config.Config
	BizRedis   *redis.Redis
	UserModel  model.UserModel
	PostClient postclient.PostClient // <--- 注册到ServiceContext
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.DB.DataSource)

	redis := redis.MustNewRedis(c.BizRedis)
	postClient := zrpc.MustNewClient(c.PostRpc)

	return &ServiceContext{
		Config:     c,
		BizRedis:   redis,
		UserModel:  model.NewUserModel(conn, c.Cache),
		PostClient: post.NewPostClient(postClient), // <--- 初始化
	}
}
````
或者使用如下`one-liner` `post.NewPostClient(zrpc.MustNewClient(c.PostRpc))`

### 在纯gRPC服务中

为了将gRPC接口暴露出去，有两种方式：
#### 开启`grpc reflection`

在/etc/XXX.yaml中设置`Mode`
```yaml
Name: post.rpc
ListenOn: 0.0.0.0:9001
Mode: dev // or Mode: test
```

#### 使用`protoset`
使用`protoc`生成protoset文件
```bash
protoc --descriptor_set_out=user.protoset --include_imports user.proto
```
并使用`go-zero`自带的gateway中使用。
```yaml
Name: gateway-example
Host: 0.0.0.0
Port: 8888
Upstreams:
  - Name: User
    Grpc:
      Etcd:
        Hosts:
          - etcd:2379
        Key: user.rpc
    ProtoSets:
      - user.proto.protoset
    Mappings:
      - Method: POST
        Path: /api/user/getpostbyuser
        RpcPath: user.Post/Getpost // <-- [grpc包名].[service名]/[service中rpc接口名]

  - Name: Post
    Grpc:
      Etcd:
        Hosts:
          - etcd:2379
        Key: post.rpc
    ProtoSets:
      - post.proto.protoset
    Mappings:
      - Method: POST
        Path: /api/post/getpost
        RpcPath: post.Post/Getpost
```

#### 或者定义`.api`
在生成的http逻辑中，调用grpc微服务，相当于简单的wrapper。
但是要配置服务发现和注册到上下文，有点麻烦。