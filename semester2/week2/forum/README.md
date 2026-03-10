# Forum API

基于 Go 语言的高性能社区论坛后端服务，采用整洁架构（Clean Architecture）和依赖注入设计。

## 技术栈

- **语言**: Go 1.25
- **Web框架**: Gin
- **数据库**: MongoDB
- **认证**: JWT (golang-jwt/jwt/v5)
- **依赖注入**: Google Wire
- **配置管理**: Viper
- **密码加密**: bcrypt

## 项目结构

```
forum/
├── cmd/
│   └── main.go              # 应用入口
├── config/
│   ├── config.go            # 配置结构体
│   └── loader.go            # Viper配置加载
├── models/
│   ├── models.go            # Post/Comment模型
│   ├── user.go              # User模型
│   └── dto.go               # 数据传输对象
├── repositories/
│   ├── interfaces.go        # Repository接口
│   ├── user_repository.go   # 用户Repository实现
│   └── post_repository.go   # 帖子Repository实现
├── services/
│   ├── interfaces.go        # Service接口
│   ├── user_service.go      # 用户Service实现
│   ├── jwt_service.go       # JWT Service实现
│   └── post_service.go      # 帖子Service实现
├── controllers/
│   └── controllers.go       # HTTP Handler
├── middlewares/
│   └── jwt_auth.go          # JWT认证中间件
├── errors/
│   └── domain_errors.go     # 领域错误定义
├── di/
│   ├── wire.go              # Wire配置
│   ├── providers.go         # Provider集合
│   └── wire_gen.go          # Wire生成代码
├── config.yaml              # 配置文件
└── go.mod
```

## 核心功能

### 1. 用户认证
- 用户注册（用户名、邮箱、密码）
- 用户登录（JWT token生成）
- JWT认证中间件

### 2. 帖子管理
- 创建帖子（支持自定义字段）
- 查询帖子详情
- 分页查询帖子列表
- 更新帖子（仅作者）
- 删除帖子（仅作者）

### 3. 评论系统
- 添加评论（支持嵌套回复）
- 删除评论（仅作者）

### 4. 投票系统
- 帖子点赞/踩（原子操作）
- 评论点赞/踩（原子操作）
- 防止重复投票

## API 接口

### 公开接口（无需认证）

```
POST   /api/auth/register    # 用户注册
POST   /api/auth/login       # 用户登录
```

### 受保护接口（需要JWT认证）

```
GET    /api/posts            # 获取帖子列表（分页）
POST   /api/posts            # 创建帖子
GET    /api/posts/:id        # 获取帖子详情
PUT    /api/posts/:id        # 更新帖子
DELETE /api/posts/:id        # 删除帖子

POST   /api/posts/:id/comments           # 添加评论
DELETE /api/posts/:id/comments/:commentId # 删除评论

POST   /api/posts/:id/vote               # 帖子投票
POST   /api/posts/:id/comments/:commentId/vote # 评论投票
```

## 配置说明

编辑 `config.yaml` 文件：

```yaml
server:
  port: 8080              # 服务端口
  mode: debug             # 运行模式: debug/release

mongodb:
  uri: "mongodb://localhost:27017"  # MongoDB连接URI
  database: "forum_db"              # 数据库名称

jwt:
  secret: "your-secret-key"  # JWT密钥（生产环境请修改）
  expires_in: "24h"          # Token有效期
  issuer: "forum-api"        # 签发者
```

## Docker 部署（推荐）

### 使用 Docker Compose 一键部署

项目提供了完整的 Docker 部署方案，包含应用和 MongoDB 数据库。

#### 1. 启动服务

```bash
docker-compose up -d
```

这将自动：
- 构建 Go 应用镜像
- 启动 MongoDB 容器（带认证）
- 启动应用容器
- 配置网络和数据卷

#### 2. 查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 仅查看应用日志
docker-compose logs -f app

# 仅查看 MongoDB 日志
docker-compose logs -f mongodb
```

#### 3. 停止服务

```bash
docker-compose down
```

#### 4. 停止并删除数据卷

```bash
docker-compose down -v
```

### Docker 配置说明

#### MongoDB 认证信息

Docker Compose 中的 MongoDB 使用以下认证信息：

- **用户名**: `admin`
- **密码**: `admin123`
- **数据库**: `forum_db`

#### 数据持久化

MongoDB 数据存储在 Docker 卷中：

- `mongodb_data`: 数据库文件
- `mongodb_config`: 配置文件

#### 健康检查

应用容器会等待 MongoDB 健康检查通过后启动，确保数据库就绪。

### 单独构建镜像

如果只想构建应用镜像：

```bash
docker build -t forum-api:latest .
```

### 环境变量配置

可以通过环境变量覆盖配置：

```bash
docker-compose run -e GIN_MODE=debug app
```

## 快速开始

### 方式一：Docker 部署（推荐）

参见上方 [Docker 部署](#docker-部署推荐) 章节。

### 方式二：本地开发部署

#### 1. 安装依赖

```bash
go mod download
```

#### 2. 启动MongoDB

确保MongoDB服务正在运行。

#### 3. 运行应用

```bash
go run ./cmd/main.go
```

或编译后运行：

```bash
go build -o forum ./cmd
./forum
```

#### 4. 测试API

#### 注册用户

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123"
  }'
```

#### 登录

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

返回的token用于后续请求认证。

#### 创建帖子

```bash
curl -X POST http://localhost:8080/api/posts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-token>" \
  -d '{
    "title": "My First Post",
    "content": "This is the content",
    "custom": {"tags": ["golang", "mongodb"]}
  }'
```

#### 添加评论

```bash
curl -X POST http://localhost:8080/api/posts/<post-id>/comments \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-token>" \
  -d '{
    "content": "Great post!",
    "replyToId": ""
  }'
```

#### 投票

```bash
curl -X POST http://localhost:8080/api/posts/<post-id>/vote \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-token>" \
  -d '{
    "action": "up"
  }'
```

## 架构特点

### 1. 分层架构

- **Controller层**: 处理HTTP请求，参数校验，调用Service
- **Service层**: 业务逻辑，接口定义，错误处理
- **Repository层**: 数据访问，MongoDB操作，接口定义

### 2. 依赖注入

使用Google Wire进行编译时依赖注入，避免运行时反射开销。

### 3. 接口隔离

所有Service和Repository都定义为接口，便于测试和扩展。

### 4. 原子操作

投票功能使用MongoDB的`$inc`、`$addToSet`等原子操作，保证并发安全。

### 5. 嵌套文档

评论作为数组嵌套在帖子文档中，使用`arrayFilters`实现评论级别的原子更新。

## 开发说明

### 重新生成Wire代码

修改`di/wire.go`或`di/providers.go`后，运行：

```bash
cd di && wire
```

或：

```bash
go generate ./di/...
```

### 添加新功能

1. 在`models/`中定义数据模型和DTO
2. 在`repositories/interfaces.go`中定义Repository接口
3. 在`repositories/`中实现Repository
4. 在`services/interfaces.go`中定义Service接口
5. 在`services/`中实现Service
6. 在`controllers/`中添加Handler
7. 在`di/providers.go`中添加Provider
8. 重新生成Wire代码

## 安全建议

1. **生产环境**请修改`config.yaml`中的JWT密钥
2. 使用环境变量管理敏感配置
3. 启用HTTPS
4. 添加速率限制
5. 实现输入验证和XSS防护

## License

MIT
