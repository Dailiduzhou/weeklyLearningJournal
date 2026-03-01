# mvc_seaorm

基于 SeaORM 的 Rust 用户管理 MVC 系统，完整实现 week6/mvc (Go版) 的所有功能。

## 技术栈
- **Web 框架**: actix-web 4
- **会话管理**: actix-session (Cookie)
- **ORM**: SeaORM 0.12
- **数据库**: MySQL
- **密码加密**: bcrypt
- **模板引擎**: Tera

## 功能特性
- ✅ 用户注册（表单验证 + bcrypt 密码加密）
- ✅ 用户登录（会话管理）
- ✅ 个人资料查看与修改
- ✅ 密码修改（需验证原密码）
- ✅ 登出
- ✅ 管理员用户列表（需 admin 权限）

## SeaORM 核心特性展示

### 实体定义
```rust
#[derive(Clone, Debug, PartialEq, DeriveEntityModel, Serialize)]
#[sea_orm(table_name = "users")]
pub struct Model {
    #[sea_orm(primary_key)]
    pub id: i64,
    #[sea_orm(unique)]
    pub username: String,
    pub password: String,
    pub name: String,
    pub created_at: Option<chrono::NaiveDateTime>,
    pub updated_at: Option<chrono::NaiveDateTime>,
}
```

### CRUD 操作
```rust
// 创建 (Create)
let new_user = UserActiveModel {
    username: Set("zhangsan".to_string()),
    password: Set(hashed_password),
    name: Set("张三".to_string()),
    ..Default::default()
};
new_user.insert(&db).await?;

// 查询 (Read)
let user = UserEntity::find()
    .filter(UserColumn::Username.eq("zhangsan"))
    .one(&db)
    .await?;

// 更新 (Update)
let mut active_user: UserActiveModel = user.into();
active_user.name = Set("新名字".to_string());
active_user.updated_at = Set(Some(chrono::Local::now().naive_local()));
active_user.update(&db).await?;

// 查询所有
let all_users = UserEntity::find().all(&db).await?;
```

## 环境准备

### 1. 数据库设置
确保 MySQL 运行在 `localhost:3307`，创建数据库和表：

```sql
CREATE DATABASE IF NOT EXISTS ginserver;
USE ginserver;

CREATE TABLE IF NOT EXISTS users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  username VARCHAR(100) NOT NULL UNIQUE,
  password VARCHAR(100) NOT NULL,
  name VARCHAR(100) NOT NULL,
  created_at DATETIME,
  updated_at DATETIME
);
```

### 2. 运行项目
```bash
cd week6/mvc_seaorm
cargo run
```

服务器将在 `http://localhost:8080` 启动。

## API 端点

| 方法 | 路径 | 功能 | 需登录 |
|------|------|------|--------|
| GET | `/api/users` | 注册页面 | ❌ |
| POST | `/api/users` | 提交注册 | ❌ |
| GET | `/api/users/login` | 登录页面 | ❌ |
| POST | `/api/users/login` | 提交登录 | ❌ |
| GET | `/api/users/me` | 个人信息页 | ✅ |
| POST | `/api/users/logout` | 登出 | ✅ |
| GET | `/api/users/profiles` | 修改资料页 | ✅ |
| PUT | `/api/users/profiles` | 提交资料修改 | ✅ |
| GET | `/api/users/password` | 修改密码页 | ✅ |
| PUT | `/api/users/password` | 提交密码修改 | ✅ |
| GET | `/api/admin/users` | 用户列表(admin) | ✅ |

## SeaORM vs sqlx 对比

| 特性 | SeaORM | sqlx (原 mvc_rust) |
|------|--------|-------------------|
| 类型 | ORM | SQL-first |
| 查询方式 | 链式 API + 类型安全 | 原始 SQL + 宏 |
| 迁移 | 内置迁移工具 | 手动 SQL 文件 |
| 关系处理 | 声明式关系 | 手动 JOIN |
| 学习曲线 | 中等 | 低（熟悉 SQL 即可）|
| 性能 | 略有开销 | 接近原生 |

### 代码对比示例

**sqlx (原版)**:
```rust
sqlx::query_as::<_, User>(
    "SELECT id, username, password, name, created_at, updated_at FROM users WHERE username = ?"
)
.bind(&username)
.fetch_one(&db).await
```

**SeaORM (本版)**:
```rust
UserEntity::find()
    .filter(UserColumn::Username.eq(&username))
    .one(&db)
    .await
```

## 测试建议

### 1. 注册用户
```bash
curl -X POST http://localhost:8080/api/users \
  -d "username=testuser&password=pass123&name=测试用户"
```

### 2. 登录
```bash
curl -X POST http://localhost:8080/api/users/login \
  -c cookies.txt \
  -d "username=testuser&password=pass123"
```

### 3. 查看个人信息（需登录）
```bash
curl http://localhost:8080/api/users/me \
  -b cookies.txt
```

### 4. 修改资料
```bash
curl -X PUT http://localhost:8080/api/users/profiles \
  -b cookies.txt \
  -d "newname=新名字"
```

## 项目结构
```
mvc_seaorm/
├── Cargo.toml          # 依赖配置
├── README.md           # 本文档
├── src/
│   ├── main.rs         # 主程序（路由 + 处理器）
│   └── entity.rs       # SeaORM 实体定义
├── templates/          # Tera 模板
│   ├── register.html
│   ├── login.html
│   ├── profiles.html
│   ├── changeprofiles.html
│   └── changepassword.html
└── migration/          # 数据库迁移（可选）
    └── init.sql
```

## 扩展建议

1. **添加迁移工具**：使用 `sea-orm-cli` 生成迁移文件
   ```bash
   cargo install sea-orm-cli
   sea-orm-cli migrate init
   ```

2. **关系映射**：如果需要用户-文章等关联，在 `entity.rs` 添加：
   ```rust
   #[derive(Copy, Clone, Debug, EnumIter, DeriveRelation)]
   pub enum Relation {
       #[sea_orm(has_many = "super::post::Entity")]
       Post,
   }
   ```

3. **Redis 会话**：替换 `CookieSessionStore` 为 `RedisSessionStore`

4. **API 文档**：集成 `utoipa` 生成 OpenAPI 规范

## 对比原项目

- **Go 版 (mvc)**: Gin + GORM + gin-contrib/sessions
- **Rust sqlx 版 (mvc_rust)**: Actix + sqlx + 原生 SQL
- **Rust SeaORM 版 (本项目)**: Actix + SeaORM + 类型安全查询

本项目展示了在 Rust 生态中使用成熟 ORM 的最佳实践，适合：
- 需要类型安全和自动补全的团队
- 复杂关系模型的应用
- 希望减少 SQL 样板代码的项目
