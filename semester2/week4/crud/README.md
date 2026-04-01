# User Service - Go-Zero CRUD API

基于 go-zero 框架的用户管理微服务，支持 JWT 双令牌认证。

## 🚀 快速开始

### 方式一：Docker Compose（推荐）

```bash
# 构建并启动所有服务
make docker-up

# 查看服务状态
make docker-ps

# 查看应用日志
make docker-logs

# 停止服务
make docker-down
```

服务启动后：
- API 地址: http://localhost:8888
- PostgreSQL: localhost:5432
- Redis: localhost:6379

### 方式二：本地运行

前置条件：
- Go 1.23+
- PostgreSQL 16+
- Redis 7+

```bash
# 启动依赖服务
docker run -d --name postgres -p 5432:5432 \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=123456 \
  -e POSTGRES_DB=mydb \
  postgres:16-alpine

docker run -d --name redis -p 6379:6379 redis:7-alpine

# 初始化数据库
psql -h localhost -U postgres -d mydb -f user.sql

# 运行服务
make run
```

## 📡 API 接口

### 公开接口（无需认证）

#### 用户注册
```bash
POST /api/v1/user/register
Content-Type: application/json

{
  "username": "alice",
  "password": "password123"
}
```

#### 用户登录
```bash
POST /api/v1/user/login
Content-Type: application/json

{
  "username": "alice",
  "password": "password123"
}

# 响应
{
  "username": "alice",
  "access_token": "eyJhbG...",
  "refresh_token": "eyJhbG..."
}
```

#### 刷新令牌
```bash
POST /api/v1/user/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbG..."
}
```

### 受保护接口（需要 JWT）

#### 获取用户信息
```bash
GET /api/v1/user/:id
Authorization: Bearer <access_token>

# 响应
{
  "id": 1,
  "username": "alice",
  "role": "user"
}
```

#### 更新用户资料
```bash
POST /api/v1/user/update
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "id": 1,
  "username": "alice_new"
}
```

### 管理员接口（需要 admin 角色）

#### 删除用户
```bash
POST /api/v1/admin/user/:id
Authorization: Bearer <admin_access_token>
```

## 🔐 JWT 认证机制

### Token 结构

**Access Token（1小时有效期）**
```json
{
  "ID": 1,
  "Role": "user",
  "jti": "uuid-xxx",
  "iat": 1640000000,
  "exp": 1640003600
}
```

**Refresh Token（7天有效期）**
```json
{
  "ID": 1,
  "jti": "uuid-xxx",
  "iat": 1640000000,
  "exp": 1640604800
}
```

### 认证流程

1. 用户登录获取双令牌
2. 使用 Access Token 访问受保护接口
3. Access Token 过期前，使用 Refresh Token 刷新
4. Refresh Token 刷新后，旧 Token 自动加入黑名单

### 权限控制

- **公开接口**: 注册、登录、刷新令牌
- **认证接口**: 查看用户信息、更新资料（仅限本人）
- **管理员接口**: 删除用户（需要 admin 角色）

## 🛠️ 开发命令

```bash
# 编译
make build

# 运行
make run

# 代码格式化
make fmt

# 代码检查
make lint

# 运行测试
make test

# 清理
make clean
```

## 🐳 Docker 命令

```bash
# 构建镜像
make docker-build

# 启动服务
make docker-up

# 停止服务
make docker-down

# 查看日志
make docker-logs

# 查看容器状态
make docker-ps
```

## 📁 项目结构

```
.
├── cmd/
│   ├── desc/              # API 定义文件
│   ├── etc/               # 配置文件
│   ├── internal/
│   │   ├── config/        # 配置加载
│   │   ├── handler/       # HTTP 处理器
│   │   ├── logic/         # 业务逻辑
│   │   ├── middleware/    # 中间件
│   │   ├── svc/           # 服务上下文
│   │   ├── types/         # 类型定义
│   │   └── utils/         # 工具函数
│   ├── model/             # 数据模型
│   └── user.go            # 入口文件
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── user.sql               # 数据库初始化脚本
```

## ⚙️ 配置说明

### 环境变量

配置文件位于 `cmd/etc/user.yaml`，主要配置项：

```yaml
JwtAuth:
  AccessSecret: "your-access-secret"    # Access Token 签名密钥
  AccessExpire: 3600                     # Access Token 有效期（秒）
  RefreshSecret: "your-refresh-secret"  # Refresh Token 签名密钥
  RefreshExpire: 604800                  # Refresh Token 有效期（秒）

DB:
  DataSource: "postgres://..."           # 数据库连接

Cache:                                   # 数据库缓存配置
  - Host: 127.0.0.1:6379

BizRedis:                                # 业务 Redis（黑名单）
  Host: 127.0.0.1:6379
```

### Docker 环境配置

Docker 环境使用 `cmd/etc/user-docker.yaml`，服务地址改为容器服务名：
- PostgreSQL: `postgres:5432`
- Redis: `redis:6379`

## 🔧 数据库初始化

首次启动时，数据库会自动执行 `user.sql` 初始化脚本：

```sql
CREATE TABLE "user" (
  id bigserial PRIMARY KEY,
  username VARCHAR(255) NOT NULL,
  password VARCHAR(255) NOT NULL,
  role VARCHAR(255) NOT NULL
);

CREATE INDEX idx_user ON "user"(username);
```

## 📝 开发注意事项

1. **JWT 密钥安全**: 生产环境必须使用强密钥（256位）
2. **密钥分离**: Access 和 Refresh Token 必须使用不同密钥
3. **HTTPS**: 生产环境必须启用 HTTPS
4. **数据库迁移**: 使用数据库迁移工具管理 schema 变更
5. **日志监控**: 配置日志收集和监控系统

## 🚀 生产部署建议

1. 使用 Kubernetes 编排
2. 配置 CI/CD 自动化部署
3. 启用 Prometheus 监控
4. 使用 Redis Sentinel/Cluster 高可用
5. PostgreSQL 主从复制
6. API 网关（Kong/Traefik）

## 📄 License

MIT
