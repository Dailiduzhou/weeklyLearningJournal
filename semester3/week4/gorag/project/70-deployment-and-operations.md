# 70 部署与运维

## 任务目标

使用固定版本的 Docker Compose 运行 PostgreSQL/pgvector、Ollama 和 RAG Server，并提供可重复的配置、迁移与启动检查。

主要区域：

```text
compose.yaml
config.yaml
internal/config
cmd/server
cmd/indexer
README.md
```

## Compose 服务

- PostgreSQL，并启用 pgvector。
- Ollama。
- RAG Server。
- Indexer 作为按需运行的一次性任务，而非常驻服务。

启动前拉取 `qwen3-embedding:0.6b`。所有镜像和 Go 依赖版本需要固定。

## 配置项

以下配置可通过环境变量设置：

- PostgreSQL 连接地址。
- Ollama 服务地址。
- Embedding 模型名称。
- Embedding 维度。
- 文档目录。
- Top K。
- 相似度阈值。
- 回答模型配置。

配置由 Viper 加载。默认值、配置文件和环境变量的优先级应明确；日志不能输出数据库密码或模型 API 密钥。

## 数据库初始化

- PostgreSQL 启动后自动执行 go-migrations 管理的迁移。
- 迁移负责启用 `vector` 扩展及创建业务表、约束和索引。
- server 与 indexer 不以临时 DDL 代替版本化迁移。

## Server 启动检查

server 启动或进入 ready 状态前检查：

1. PostgreSQL 是否可用。
2. `vector` 扩展是否存在。
3. Ollama 是否可用。
4. Embedding 模型和数据库向量维度是否一致。

`/healthz` 表示进程存活；`/readyz` 反映关键依赖是否满足服务条件。

## 运行与关闭

- 为外部依赖设置启动等待、健康检查和明确超时。
- server 接收终止信号后停止接收新请求，并在超时内优雅关闭。
- indexer 支持作为 Compose 一次性任务运行各类索引命令。
- 使用 `log/slog` 输出可检索的结构化日志。

## 依赖

- 数据库结构来自 [`20-embedding-and-storage.md`](20-embedding-and-storage.md)。
- indexer 行为来自 [`30-index-lifecycle.md`](30-index-lifecycle.md)。
- HTTP 健康/就绪接口来自 [`50-answer-and-api.md`](50-answer-and-api.md)。

## 完成标准

- 干净环境可通过文档化命令启动所有常驻服务、迁移数据库并拉取模型。
- 可按需执行一次性 indexer，同步 `docs` 后使用 API 提问。
- 配置可由环境变量覆盖，缺失或冲突配置在启动时给出清晰错误。
- PostgreSQL、vector 扩展、Ollama、模型或维度不满足时，ready 检查失败。
- 停止 Compose 不遗留无法解释的运行任务；持久数据卷行为在 README 中说明。
