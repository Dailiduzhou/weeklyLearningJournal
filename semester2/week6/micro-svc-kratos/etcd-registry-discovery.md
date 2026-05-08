# Kratos 微服务 etcd 服务注册与发现

## 整体架构

```
                    ┌─────────────────┐
                    │      etcd       │
                    │  (127.0.0.1:2379)│
                    └──┬───────────┬──┘
              注册      │           │  发现+注册
           ┌───────────┘           └───────────┐
           ▼                                    ▼
   ┌──────────────┐                    ┌──────────────┐
   │    svc-b     │◄──── gRPC ────────│    svc-a     │
   │  (提供者)     │   调用 Pong RPC    │  (消费者)     │
   └──────────────┘                    └──────────────┘
```

**调用链路**：`Client -> HTTP /ping (svc-a) -> gRPC Pong (svc-b, via etcd discovery) -> response`

---

## 1. 配置文件（etcd 地址的来源）

两个服务都在 `configs/config.yaml` 中配置 etcd 端点：

```yaml
# svc-a/configs/config.yaml / svc-b/configs/config.yaml
registry:
  etcd:
    endpoints:
      - "127.0.0.1:2379"
```

Docker 环境下使用 `config.docker.yaml`，etcd 端点改为 `etcd:2379`（容器服务名）。

---

## 2. 服务注册（svc-a 和 svc-b 相同逻辑）

核心文件：`internal/server/registry.go`（两个服务内容完全一致）

```go
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

// NewDiscovery 创建 Kratos 服务发现器
func NewDiscovery(client *clientv3.Client) registry.Discovery {
    return etcd.New(client)
}
```

### 三步注册流程

| 步骤 | 函数 | 作用 |
|------|------|------|
| ① | `NewEtcdClient(c *conf.Registry)` | 根据配置创建 etcd v3 原生客户端 |
| ② | `NewRegistrar(client) registry.Registrar` | 用 `contrib/registry/etcd/v2.New(client)` 将 etcd 客户端包装成 Kratos 的 `Registrar` 接口 |
| ③ | `kratos.Registrar(rr)` 传入 `newApp()` | Kratos 框架在 `app.Run()` 时自动以 `kratos.Name(Name)` 为 key 将服务地址注册到 etcd |

### main.go 中的注册入口

```go
func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server, rr registry.Registrar) *kratos.App {
    return kratos.New(
        kratos.ID(id),
        kratos.Name(Name),    // "svc-a" 或 "svc-b" — 这是 etcd 中注册的 key
        kratos.Version(Version),
        kratos.Metadata(map[string]string{}),
        kratos.Logger(logger),
        kratos.Server(gs, hs),
        kratos.Registrar(rr), // 传入 etcd registrar
    )
}
```

### Wire 注入链（svc-b 为例）

```
EtcdClient ──→ Registrar ──→ App
```

```go
// wire_gen.go
client := server.NewEtcdClient(registry)      // ① 创建 etcd 客户端
registrar := server.NewRegistrar(client)       // ② 包装为 Registrar
app := newApp(logger, grpcServer, httpServer, registrar) // ③ 传入 App
```

---

## 3. 服务发现（仅 svc-a 发现 svc-b）

svc-a 需要调用 svc-b 的 gRPC 接口，因此还需要服务发现能力。

### 3.1 发现器创建

```go
func NewDiscovery(client *clientv3.Client) registry.Discovery {
    return etcd.New(client)  // etcd.New() 同时实现了 Registrar 和 Discovery 接口
}
```

### 3.2 gRPC 客户端通过发现器解析目标服务

文件：`svc-a/internal/server/bclient.go`

```go
func NewBClient(dis registry.Discovery) bv1.BClient {
    conn, err := transgrpc.DialInsecure(
        nil,
        transgrpc.WithEndpoint("discovery:///svc-b"),  // "discovery:///" 前缀触发服务发现
        transgrpc.WithDiscovery(dis),                    // 注入 etcd 发现器
    )
    if err != nil {
        panic(err)
    }
    return bv1.NewBClient(conn)
}
```

**关键点**：

- 使用 `discovery:///svc-b` 作为 endpoint，Kratos 识别 `discovery:///` 前缀后走服务发现流程
- 框架通过注入的 `Discovery` 去 etcd 查询 key 为 `svc-b` 的服务实例地址
- 返回的 `conn` 是一个自动负载均衡的 gRPC 连接（多个 svc-b 实例时自动轮询）

### 3.3 svc-a 引用 svc-b 的 proto 定义

`svc-a/go.mod` 中有本地 replace 指令，使得 svc-a 可以直接 import svc-b 的 API：

```
replace svc-b => ../svc-b
```

```go
import bv1 "svc-b/api/helloworld/v1"  // svc-a 调用 svc-b 的 Pong RPC
```

---

## 4. Wire 依赖注入对比

### svc-b（纯提供者 — 仅注册，不发现）

```
EtcdClient ──→ Registrar ──→ App
```

```go
// svc-b/cmd/svc-b/wire_gen.go
func wireApp(...) (*kratos.App, func(), error) {
    greeterService := service.NewGreeterService()
    grpcServer := server.NewGRPCServer(confServer, greeterService, logger)
    httpServer := server.NewHTTPServer(confServer, greeterService, logger)
    client := server.NewEtcdClient(registry)
    registrar := server.NewRegistrar(client)
    app := newApp(logger, grpcServer, httpServer, registrar)
    return app, func() {}, nil
}
```

### svc-a（消费者 + 提供者 — 注册 + 发现）

```
EtcdClient ──→ Registrar ──────────────────────────→ App
          └─→ Discovery ──→ BClient ──→ GreeterService ──→ gRPC/HTTP Server
```

```go
// svc-a/cmd/svc-a/wire_gen.go
func wireApp(...) (*kratos.App, func(), error) {
    etcdClient := server.NewEtcdClient(registry)
    discovery := server.NewDiscovery(etcdClient)           // ① 创建发现器
    bClient := server.NewBClient(discovery)                // ② 创建 svc-b 的 gRPC 客户端
    greeterService := service.NewGreeterService(bClient)   // ③ 注入到业务服务
    grpcServer := server.NewGRPCServer(confServer, greeterService, logger)
    httpServer := server.NewHTTPServer(confServer, greeterService, logger)
    registrar := server.NewRegistrar(etcdClient)           // ④ 创建注册器
    app := newApp(logger, grpcServer, httpServer, registrar) // ⑤ 传入 App
    return app, func() {}, nil
}
```

### Wire ProviderSet 定义

```go
// svc-a/internal/server/server.go
var ProviderSet = wire.NewSet(
    NewGRPCServer,
    NewHTTPServer,
    NewEtcdClient,
    NewRegistrar,
    NewDiscovery,
)
```

```go
// svc-a/internal/service/service.go
var ProviderSet = wire.NewSet(NewGreeterService)
```

---

## 5. 功能对比总结

| 能力 | svc-b | svc-a |
|------|-------|-------|
| 向 etcd 注册自己 | ✅ `NewRegistrar` | ✅ `NewRegistrar` |
| 从 etcd 发现其他服务 | ❌ | ✅ `NewDiscovery` → `NewBClient` |
| etcd 中的注册 key | `svc-b` | `svc-a` |
| 对外暴露的 API | `Ping`, `Pong` | `Ping`（内部调用 svc-b.Pong） |

---

## 6. 核心依赖包

| 包 | 作用 |
|----|------|
| `go.etcd.io/etcd/client/v3` | etcd 原生客户端 |
| `github.com/go-kratos/kratos/v2/registry` | Kratos 的 Registrar/Discovery 抽象接口 |
| `github.com/go-kratos/kratos/contrib/registry/etcd/v2` | etcd 对 Kratos registry 接口的实现 |

关键设计：`etcd.New(client)` 同时实现了 `registry.Registrar` 和 `registry.Discovery` 两个接口，同一个调用结果既可用于注册也可用于发现。
