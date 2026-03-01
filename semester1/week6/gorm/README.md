// AI
# GORM + MySQL MVC 示例项目

这是一个使用 GORM、Gin 和 MySQL 构建的简单 MVC 架构项目。

## 项目结构

```
gorm/
├── main.go              # 程序入口
├── config/              # 配置层
│   └── database.go      # 数据库配置
├── model/               # 模型层（Model）
│   └── user.go          # 用户模型
├── service/             # 服务层（业务逻辑）
│   └── user_service.go  # 用户服务
├── controller/          # 控制器层（Controller）
│   └── user_controller.go # 用户控制器
├── go.mod               # Go 模块文件
└── README.md            # 项目说明
```

## MVC 架构说明

- **Model（模型层）**: 定义数据结构和数据库表映射
- **View（视图层）**: 本项目使用 JSON API，视图层由前端处理
- **Controller（控制器层）**: 处理 HTTP 请求，调用 Service 层

## 前置准备

1. 安装 MySQL 数据库
2. 创建数据库：
```sql
CREATE DATABASE gorm_demo CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

3. 修改 `config/database.go` 中的数据库配置（用户名、密码等）

## 安装依赖

```bash
cd /root/weeklyLearningJournal/weeklyLearningJournal/week6/gorm
go mod tidy
```

## 运行项目

```bash
go run main.go
```

服务器将在 `http://localhost:8080` 启动

## API 接口

### 1. 创建用户
```bash
POST /api/users
Content-Type: application/json

{
  "name": "张三",
  "email": "zhangsan@example.com",
  "age": 25
}
```

### 2. 获取所有用户
```bash
GET /api/users
```

### 3. 获取单个用户
```bash
GET /api/users/:id
```

### 4. 更新用户
```bash
PUT /api/users/:id
Content-Type: application/json

{
  "name": "李四",
  "age": 30
}
```

### 5. 删除用户
```bash
DELETE /api/users/:id
```

## 测试示例

使用 curl 测试：

```bash
# 创建用户
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"name":"张三","email":"zhangsan@example.com","age":25}'

# 获取所有用户
curl http://localhost:8080/api/users

# 获取单个用户
curl http://localhost:8080/api/users/1

# 更新用户
curl -X PUT http://localhost:8080/api/users/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"李四","age":30}'

# 删除用户
curl -X DELETE http://localhost:8080/api/users/1
```

## 技术栈

- **Gin**: Web 框架
- **GORM**: ORM 框架
- **MySQL**: 数据库
- **Go**: 编程语言

## 特性

- ✅ RESTful API 设计
- ✅ MVC 架构模式
- ✅ GORM 数据库操作
- ✅ 自动数据库迁移
- ✅ JSON 数据交互
- ✅ 错误处理
