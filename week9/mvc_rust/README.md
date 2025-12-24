# mvc_rust

Rust 版用户管理 MVC，等价于 week6/mvc 的功能：注册、登录、会话、资料修改、密码修改、管理员用户列表，以及 HTML 页面渲染。

## 技术栈
- actix-web + actix-session（Cookie 会话）
- sqlx (MySQL)
- bcrypt
- tera 模板

## 运行
1. 保证 MySQL 可用，和 Go 版同样连接：`mysql://root:123456@localhost:3307/ginserver`
2. 创建 `users` 表（与 Go 版一致）：

```sql
CREATE TABLE IF NOT EXISTS users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  username VARCHAR(100) NOT NULL UNIQUE,
  password VARCHAR(100) NOT NULL,
  name VARCHAR(100) NOT NULL,
  created_at DATETIME,
  updated_at DATETIME
);
```

3. 复制模板：从 `week6/mvc/template/*.html` 到 `week6/mvc_rust/templates/`
4. 构建并运行：

```bash
cd week6/mvc_rust
cargo run
```

启动后访问：
- 注册页：/api/users (GET)
- 注册提交：/api/users (POST)
- 登录页：/api/users/login (GET)
- 登录提交：/api/users/login (POST)
- 我的信息页：/api/users/me (GET)
- 修改资料页：/api/users/profiles (GET)
- 修改资料提交：/api/users/profiles (PUT)
- 修改密码页：/api/users/password (GET)
- 修改密码提交：/api/users/password (PUT)
- 管理员用户列表：/api/admin/users (GET)

