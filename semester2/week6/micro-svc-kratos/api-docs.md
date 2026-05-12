# API 接口文档

本文档描述了微服务项目中所有可用的 API 接口，包括直接访问各服务的接口和通过网关访问的接口。

## 目录

- [服务架构](#服务架构)
- [直连接口](#直连接口)
  - [svc-a 服务](#svc-a-服务)
  - [svc-b 服务](#svc-b-服务)
- [网关接口](#网关接口)
- [请求/响应格式](#请求响应格式)
- [错误码](#错误码)

---

## 服务架构

```
┌─────────────┐     ┌─────────────┐
│   客户端     │────▶│   Gateway   │
└─────────────┘     └──────┬──────┘
                           │
              ┌────────────┴────────────┐
              ▼                         ▼
       ┌──────────┐              ┌──────────┐
       │  svc-a   │              │  svc-b   │
       │ :8000/g  │              │ :8000/g  │
       │ :9000/g  │              │ :9000/g  │
       └──────────┘              └──────────┘
              │                         │
              └────────────┬────────────┘
                           ▼
                    ┌──────────────┐
                    │     etcd     │
                    │   :2379      │
                    └──────────────┘
```

## 直连接口

### svc-a 服务

**服务地址**: `http://localhost:8001` (HTTP) / `localhost:9001` (gRPC)

#### Ping

检查 svc-a 服务是否正常运行。

- **URL**: `/ping`
- **Method**: `GET`
- **协议**: HTTP / gRPC

**请求参数**: 无

**响应示例**:

```json
{
  "msg": "pong from svc-a"
}
```

---

### svc-b 服务

**服务地址**: `http://localhost:8000` (HTTP) / `localhost:9000` (gRPC)

#### Ping

检查 svc-b 服务是否正常运行。

- **URL**: `/ping`
- **Method**: `GET`
- **协议**: HTTP / gRPC

**请求参数**: 无

**响应示例**:

```json
{
  "msg": "pong from svc-b"
}
```

#### Pong

svc-b 的另一个测试接口。

- **URL**: `/pong`
- **Method**: `GET`
- **协议**: HTTP / gRPC

**请求参数**: 无

**响应示例**:

```json
{
  "msg": "pong"
}
```

---

## 网关接口

**网关地址**: `http://localhost:8080`

网关统一使用 `/api/{service}/{action}` 的路径格式对外暴露接口。

#### svc-a Ping

- **URL**: `/api/a/ping`
- **Method**: `GET`
- **后端**: `svc-a /ping`
- **超时**: 1s

**请求示例**:

```bash
curl http://localhost:8080/api/a/ping
```

**响应示例**:

```json
{
  "msg": "pong from svc-a"
}
```

---

#### svc-b Ping

- **URL**: `/api/b/ping`
- **Method**: `GET`
- **后端**: `svc-b /ping`
- **超时**: 1s

**请求示例**:

```bash
curl http://localhost:8080/api/b/ping
```

**响应示例**:

```json
{
  "msg": "pong from svc-b"
}
```

---

#### svc-b Pong

- **URL**: `/api/b/pong`
- **Method**: `GET`
- **后端**: `svc-b /pong`
- **超时**: 1s

**请求示例**:

```bash
curl http://localhost:8080/api/b/pong
```

**响应示例**:

```json
{
  "msg": "pong"
}
```

---

## 请求/响应格式

### 请求格式

- Content-Type: `application/json` (POST/PUT 请求)
- GET 请求参数通过 URL Query 传递

### 响应格式

所有接口返回 JSON 格式数据。

**成功响应**:

```json
{
  "msg": "响应消息"
}
```

**错误响应**:

```json
{
  "code": 500,
  "reason": "INTERNAL_ERROR",
  "message": "服务器内部错误",
  "metadata": {}
}
```

---

## 错误码

| HTTP 状态码 | 说明 |
|------------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 404 | 接口不存在 |
| 500 | 服务器内部错误 |
| 503 | 服务不可用 |

---

## 启动服务

### 使用 Docker Compose

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f gateway
```

### 本地开发

```bash
# 1. 启动 etcd
docker run -d --name etcd -p 2379:2379 bitnami/etcd:3.5

# 2. 启动 svc-b
cd svc-b
go run ./cmd/svc-b/ -conf ./configs

# 3. 启动 svc-a
cd svc-a
go run ./cmd/svc-a/ -conf ./configs

# 4. 启动网关
cd gateway
go run ./cmd/gateway/ -conf ./configs -addr :8080 -etcd 127.0.0.1:2379
```

---

## Proto 定义

### svc-a (helloworld.v1.A)

```protobuf
service A {
  rpc Ping(PingReq) returns (PingResp) {
    option (google.api.http) = {get: "/ping"};
  }
}
```

### svc-b (helloworld.v1.B)

```protobuf
service B {
  rpc Ping(PingReq) returns (PingResp) {
    option (google.api.http) = {get: "/ping"};
  }

  rpc Pong(PongReq) returns (PongResp) {
    option (google.api.http) = {get: "/pong"};
  }
}
```
