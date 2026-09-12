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

3. 模板已放在 `templates/` 下（登录/注册/我的信息/修改资料/修改密码），无需再复制：
   - `login.html`：登录页，同时是 **POST 请求演示页**（实时显示请求行/请求头/请求体 + 等价 curl，并打印真实响应状态码、响应头、JSON 响应体）
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

## 登录页演示说明

`templates/login.html` 用来观察「登录时到底发了什么」：

- 左侧填表单，右侧实时预览即将发出的请求：`POST /api/users/login`、`Content-Type: application/x-www-form-urlencoded`、`Content-Length`、`username=...&password=...` 请求体（密码可遮蔽）以及等价 `curl` 命令。
- 提交时使用 `fetch` 把请求“摊开”展示，`credentials: "same-origin"` 保证会话 Cookie 被保存/携带。
- 展示响应状态码（400/403/404/500/200）、响应头（含 `set-cookie`）与 JSON 响应体。
- **登录成功后不自动跳转**：服务端 `login` 处理器返回的是 JSON 而不是 `302 + Location`，页面停在原地并给出「我的信息 / 修改资料 / 修改密码 / 登出」等按钮，由你手动选择下一步。
- 表单里有一个「改用浏览器原生表单提交」开关，勾选后可对比原生提交的行为（会离开本页，直接显示服务端返回的 JSON）。
- 其它页面（注册、修改资料、修改密码）同样是“展示请求 + 展示响应 + 不自动跳转”，其中 PUT 请求由 `fetch` 改写方法发出。

