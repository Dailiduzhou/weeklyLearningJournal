# mvc_rust

Rust 版用户管理 MVC，等价于 week6/mvc 的功能：注册、登录、会话、资料修改、密码修改、管理员用户列表，以及 HTML 页面渲染。

## 技术栈
- actix-web + actix-session（Cookie 会话）
- sqlx (MySQL)
- bcrypt
- tera 模板
- docker compose（MySQL）；应用可跑宿主机也可跑容器

## 文件一览
```
mvc_rust/
├── run.sh                 # 一条命令起数据库 + 应用（见下）
├── docker-compose.yml     # mysql 服务（+ full profile 下的 app 服务）
├── Dockerfile             # 应用镜像（多阶段，release 构建）
├── scripts/db/init.sql    # 建库建表 + 预置 admin 账号（幂等）
├── src/main.rs            # 所有路由/处理器
└── templates/             # 5 个页面（login 是 POST 请求演示页）
```

## 快速开始（推荐）

```bash
./run.sh
```

它会依次做这几件事，然后把演示页地址打出来：

1. 检查 `3307` 端口：有 MySQL 就直接复用，没有就用 `docker compose up -d mysql` 起一个
2. 等数据库就绪 → 执行 `scripts/db/init.sql`（建库建表 + 补 `admin` 账号，幂等，跑几次都没事）
3. `cargo run` 启动应用，等 `http://127.0.0.1:8080/api/users/login` 返回 200
4. 用真实 HTTP 请求验证种子账号能登录（顺带把「HTTP → 会话 → 数据库」整条链路走一遍）
5. Ctrl-C 退出（应用会停，数据库容器留着）

其它子命令：

| 命令 | 作用 |
| --- | --- |
| `./run.sh` | 数据库 + 应用（宿主机 `cargo run`，方便看编译和日志） |
| `./run.sh full` | 全部跑在容器里：`docker compose --profile full up --build` |
| `./run.sh db` | 只起 MySQL 并建好表，然后退出 |
| `./run.sh stop` | 停掉容器（保留数据卷）；`./run.sh stop --volumes` 连数据一起删 |
| `./run.sh reset` | 删掉数据卷重新初始化（数据库清空重建，种子账号补回） |
| `./run.sh status` | 看端口 / 容器 / 应用健康 / 表里有哪些用户 |
| `./run.sh logs [服务]` | 跟踪容器日志，服务默认 `app` |
| `./run.sh sql` | 打开 MySQL 交互式客户端 |

可用环境变量覆盖默认值：

```bash
DB_PORT=3307 DB_PASSWORD=123456 APP_PORT=8080 ./run.sh
CARGO_PROFILE=release ./run.sh        # 用 release 跑，登录快很多（见下面「注意」）
ADMIN_USER=root ADMIN_PASSWORD=xxx ./run.sh
```

## docker compose

两个服务，端口故意和 Go 版保持一致（`localhost:3307`）：

| 服务 | 容器名 | 端口 | 说明 |
| --- | --- | --- | --- |
| `mysql` | `mvc_rust-mysql` | `3307:3306` | root/123456，库 `ginserver`，带 healthcheck |
| `app` | `mvc_rust-app` | `8080:8080` | 只在 `--profile full` 下启动 |

```bash
docker compose up -d mysql                            # 只起数据库（默认行为）
docker compose --profile full up --build              # 连应用一起（首次构建要编译 actix 全家桶，比较慢）
docker compose --profile full logs -f app             # 看应用日志
docker compose --profile full down                    # 停掉
docker compose --profile full down --volumes          # 停掉并清空数据库
docker compose exec -T mysql mysql -uroot -p123456    # 进容器里的 MySQL 客户端
```

容器里的应用连的是 `mysql:3306`（服务名），通过环境变量覆盖，不依赖宿主机端口：

```
DATABASE_URL=mysql://root:123456@mysql:3306/ginserver
```

`src/main.rs` 里是 `std::env::var("DATABASE_URL")` 读不到才回落到 `mysql://root:123456@localhost:3307/ginserver`，所以两种跑法都不用改代码。

## 手动运行（不用脚本）

```bash
docker compose up -d mysql                                     # 1. 数据库
docker compose exec -T mysql mysql -uroot -p123456 < scripts/db/init.sql   # 2. 建表
cargo run                                                      # 3. 应用
```

自己已经有 MySQL 也行，只要它监听 `3307`、账号 `root/123456`、库名 `ginserver`；不一致就用 `DATABASE_URL` 覆盖，例如：

```bash
DATABASE_URL='mysql://root:123456@127.0.0.1:3306/ginserver' cargo run
```

## 接口一览

| 页面 | 方法 | 说明 |
| --- | --- | --- |
| `/api/users` | GET | 注册页 |
| `/api/users` | POST | 注册提交 → `201` JSON |
| `/api/users/login` | GET | 登录页（POST 请求演示页） |
| `/api/users/login` | POST | 登录提交 → `200` JSON + 会话 Cookie（**不返回 302，所以页面不跳转**） |
| `/api/users/me` | GET | 我的信息页（无会话则 302 到登录页） |
| `/api/users/profiles` | GET / PUT | 修改资料页 / 提交 |
| `/api/users/password` | GET / PUT | 修改密码页 / 提交 |
| `/api/users/logout` | POST | 登出（`session.purge()`） |
| `/api/admin/users` | GET | 用户列表，仅 `admin` |

内置账号：`admin / admin123`（`scripts/db/init.sql` 里存的是 bcrypt 哈希，不是明文）。

## 登录页演示说明

`templates/login.html` 用来观察「登录时到底发了什么」：

- 左侧填表单，右侧实时预览即将发出的请求：`POST /api/users/login`、`Content-Type: application/x-www-form-urlencoded`、`Content-Length`、`username=...&password=...` 请求体（密码可遮蔽）以及等价 `curl` 命令。
- 提交时使用 `fetch` 把请求“摊开”展示，`credentials: "same-origin"` 保证会话 Cookie 被保存/携带。
- 展示响应状态码（400/403/404/500/200）、响应头与 JSON 响应体。注意 `set-cookie` 被浏览器安全策略过滤掉了，`fetch` 读不到（Cookie 照样会存），想看得去 DevTools → Network。
- **登录成功后不自动跳转**：服务端 `login` 处理器返回的是 JSON 而不是 `302 + Location`，页面停在原地并给出「我的信息 / 修改资料 / 修改密码 / 登出」等按钮，由你手动选择下一步。
- 表单里有一个「改用浏览器原生表单提交」开关，勾选后可对比原生提交的行为（会离开本页，直接显示服务端返回的 JSON）。
- 其它页面（注册、修改资料、修改密码）同样是“展示请求 + 展示响应 + 不自动跳转”，其中 PUT 请求由 `fetch` 改写方法发出。

## 注意

- **debug 构建下登录要等约 0.7 秒**：`bcrypt::verify(DEFAULT_COST=12)` 在未优化编译下慢几十倍（release 下约 0.25s）。演示时嫌慢就 `CARGO_PROFILE=release ./run.sh`，容器里镜像本来就是 release 构建。
- 仓库根 `.gitignore` 里有 `*.html`，现有这 5 个模板已经用 `git add -f` 提交过了（`2b5c903 feat:HTML`），改它们、提交它们都正常；但**以后新增**的 `.html` 默认会被忽略，要么 `git add -f`，要么在 `templates/` 下加个 `.gitignore` 写 `!*.html`。
- 会话 Cookie 由 actix-session 生成，默认带 `HttpOnly`（JS 读不到 `document.cookie`），因此演示页只能“说明”Cookie 的存在，无法在页面上打印它的值。
- **工具链版本有下限：Cargo >= 1.85**。`Cargo.toml` 用的是 `edition = "2024"`（`Cargo.lock` 里 `base64ct`/`globset`/`home`/`ignore` 也是 edition 2024），旧 Cargo 会报 `feature edition2024 is required`。本机跑 `rustc --version` 看一眼，太低就 `rustup update stable`。
- `docker compose` 首次 `--build` 会从 `rust:1.98-bookworm` 开始编译所有依赖（约几分钟），网络慢的话请耐心等；之后再改代码只会重编 `src/` 那一个 crate。想一直跟最新稳定版可以把这个 tag 改成 `rust:1-bookworm`。
