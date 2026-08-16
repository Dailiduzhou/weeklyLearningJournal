# GoRAG

GoRAG 是一个面向后端学习资料的 Go 知识库服务。Markdown/TXT 文档保存在 `docs/`；一次性 indexer 负责建立或更新索引，常驻 server 仅负责检索和回答。

## 本地依赖

- Docker Engine 与 Docker Compose v2
- 首次启动时可访问 Docker Hub 和 Ollama 模型仓库
- 至少约 2 GiB 可用磁盘空间（模型和 PostgreSQL 数据会写入命名卷）

Compose 固定使用 `pgvector/pgvector:0.8.0-pg17`、`migrate/migrate:v4.18.3` 和 `ollama/ollama:0.32.0`。应用镜像从当前源码构建，构建阶段和运行阶段分别固定为 `golang:1.26.3-alpine3.22`、`alpine:3.22.1`。

PostgreSQL 与 Ollama 默认仅发布到本机回环地址的 `5432` 和 `11434` 端口，便于在宿主机运行 indexer/server 联调；可分别通过 `GORAG_POSTGRES_PORT`、`GORAG_OLLAMA_PORT` 覆盖，不会暴露到外部网卡。

## 首次启动

1. 创建本地环境文件：

   ```powershell
   Copy-Item .env.example .env
   ```

2. 编辑 `.env`，至少将 `POSTGRES_PASSWORD` 改为随机密码，并把同一个密码写入 `GORAG_DATABASE_URL`。不要提交 `.env`。

3. 先检查 Compose 展开结果，再启动：

   ```shell
   docker compose config --quiet
   docker compose up --build -d
   docker compose ps
   ```

启动顺序由 Compose 管理：PostgreSQL 健康后运行版本化迁移；Ollama 健康后拉取 `qwen3-embedding:0.6b`（以及使用 Ollama provider 时的回答模型）；两项一次性任务成功后才启动 server。首次下载模型耗时取决于网络，使用 `docker compose logs -f ollama-model-init` 查看进度。

迁移失败或模型下载失败时 server 不会进入运行状态。修复原因后重新执行 `docker compose up -d`；迁移是版本化且可重复执行的，业务进程不会用临时 DDL 代替迁移。

## 索引与提问

同步 `docs/`：

```shell
docker compose --profile tools run --rm indexer sync
```

Indexer 是一次性任务，默认不随 `docker compose up` 常驻。其他文档操作使用 indexer CLI 的 `index`、`delete`、`reindex` 和 `reindex-all` 子命令。

服务检查与提问示例：

```shell
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl -X POST http://localhost:8080/api/v1/questions \
  -H "Content-Type: application/json" \
  -d '{"question":"知识库中的事务边界是什么？"}'
```

`/healthz` 只表示进程存活。`/readyz` 只有在 PostgreSQL 可用、`vector` 扩展存在、Ollama 可用、Embedding 模型与数据库向量维度一致，且回答 provider（Ollama 模型列表或 OpenAI-compatible `/models`）可验证时才成功。依赖不可用时问答接口不会回退到模型自身知识。

## 配置

配置优先级从高到低为：`GORAG_*` 环境变量、`config.yaml`、程序内默认值。环境变量使用下划线表示嵌套字段，例如 `database.url` 对应 `GORAG_DATABASE_URL`。未知 YAML 字段、非法 URL、非正超时、越界 Top K/阈值或非 1024 的向量维度会在启动时返回明确错误。

Embedding 模型固定为 `qwen3-embedding:0.6b`，不通过配置覆盖；`embedding.model` 会被视为未知配置并拒绝。更换模型需要同时修改代码、迁移和向量维度约定，并在全量重建索引后进行。

| 环境变量 | 默认值 | 说明 |
|---|---:|---|
| `GORAG_SERVER_ADDRESS` | `:8080` | HTTP 监听地址 |
| `GORAG_DATABASE_URL` | 本机 `gorag` 数据库 | PostgreSQL URL；可能含密码，禁止写入日志 |
| `GORAG_OLLAMA_BASE_URL` | `http://localhost:11434` | Ollama 地址 |
| `GORAG_OLLAMA_TIMEOUT` | `30s` | 单次 Ollama 调用超时 |
| `GORAG_EMBEDDING_DIMENSION` | `1024` | 固定向量维度 |
| `GORAG_DOCUMENTS_DIR` | `./docs` | 仅允许 Markdown/TXT 的资料目录 |
| `GORAG_RETRIEVAL_TOP_K` | `10` | 精确检索候选数，范围 1–100 |
| `GORAG_RETRIEVAL_MAX_CONTEXT` | `5` | 最终上下文 Chunk 上限，不得超过 Top K |
| `GORAG_RETRIEVAL_SIMILARITY_THRESHOLD` | `0.5` | 相似度阈值，范围 -1–1 |
| `GORAG_ANSWER_PROVIDER` | `ollama` | `ollama` 或 `openai-compatible` |
| `GORAG_ANSWER_BASE_URL` | `http://localhost:11434` | 回答模型 API 地址 |
| `GORAG_ANSWER_MODEL` | `qwen3:4b` | 回答模型名称 |
| `GORAG_ANSWER_API_KEY` | 空 | 远程模型密钥；只能通过秘密环境注入，禁止记录 |
| `GORAG_ANSWER_TIMEOUT` | `60s` | 回答调用超时 |
| `GORAG_STARTUP_CHECK_TIMEOUT` | `30s` | 启动依赖检查总时限 |
| `GORAG_STARTUP_RETRY_INTERVAL` | `1s` | 启动检查重试间隔 |

`server.read_header_timeout` 和 `server.shutdown_timeout` 也可分别由 `GORAG_SERVER_READ_HEADER_TIMEOUT`、`GORAG_SERVER_SHUTDOWN_TIMEOUT` 覆盖。不要把生产数据库密码或模型 API 密钥写入 `config.yaml`、命令行参数或日志。

## 端到端评测

`testdata/evaluation.jsonl` 包含知识库内、同义改写、跨章节、知识库外、模糊、删除和重建场景。服务启动并准备好对应索引状态后，可运行：

```shell
GORAG_EVALUATION_ENDPOINT=http://localhost:8080/api/v1/questions go test -run TestLiveEvaluation -v .
```

删除场景与正常场景的数据库状态互斥，可分阶段执行。正常状态排除删除场景时设置 `GORAG_EVALUATION_EXCLUDE_SCENARIOS=deleted_document`；执行 `indexer delete` 后仅验证删除场景时设置 `GORAG_EVALUATION_INCLUDE_SCENARIOS=deleted_document`，完成后使用 `indexer reindex` 恢复文档。

## 停止、数据卷与排障

正常停止保留索引和模型数据：

```shell
docker compose down
```

`postgres-data` 保存数据库，`ollama-data` 保存已拉取模型，因此下次启动不必重新下载。只有确认要清空数据库、索引和模型缓存时才运行 `docker compose down --volumes`；该操作不可恢复。

常用排障命令：

```shell
docker compose ps --all
docker compose logs migrate
docker compose logs ollama-model-init
docker compose logs server
```

- PostgreSQL 不健康：确认 `.env` 的用户名、数据库名和密码一致；已有卷不会因修改环境变量自动更改原数据库密码。
- 迁移失败：检查 `migrate` 日志和迁移版本，不要直接在容器里创建业务表。
- 模型任务失败：确认磁盘空间和模型仓库网络，然后重新运行 `docker compose up -d ollama-model-init`。
- `/healthz` 成功但 `/readyz` 失败：依次检查 PostgreSQL、`vector` 扩展、Ollama、Embedding 模型和 1024 维约束。

server 收到终止信号后停止接收新请求，并在配置的关闭超时内完成优雅关闭；Compose 预留 20 秒 `stop_grace_period`。
