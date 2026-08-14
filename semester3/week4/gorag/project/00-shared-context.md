# 00 共享上下文

> 所有 subagent 必读。本文件只描述共同契约；模块细节以对应工作包为准。

## 项目目标

使用 Go 实现一个后端学习知识库。知识库文档固定存放在项目的 `docs` 目录，仅支持 Markdown 和 TXT。

系统需要支持：

1. 文档加载与清洗。
2. 文档结构化切分。
3. 本地 Embedding。
4. 向量存储与相似度检索。
5. 根据检索结果生成回答。
6. 回答附带文档来源。
7. 资料不足时明确拒绝回答。
8. 文档新增、删除和重新索引。
9. 至少编写五个知识库问题进行测试。

## 整体架构

项目分为两个独立过程。

### 文档索引过程

```text
文档扫描
→ 文档加载
→ 文档清洗
→ 文档切分
→ 批量 Embedding
→ 向量入库
→ 切换文档有效版本
```

- `cmd/indexer` 负责文档同步、新增、删除和重新索引。
- 索引失败不能破坏当前可检索的有效版本。

### 在线问答过程

```text
接收用户问题
→ 生成问题向量
→ 执行向量检索
→ 过滤低相关结果
→ 构造上下文
→ 调用大模型生成回答
→ 校验引用
→ 返回答案和来源
```

- `cmd/server` 负责提供知识库问答 HTTP 接口。
- server 不直接扫描或修改 `docs`。

## 技术选型

| 类别 | 选择 |
|---|---|
| 开发语言 | Go |
| RAG 编排 | CloudWeGo Eino：Document、Embedding、Retriever、ChatTemplate、ChatModel、Chain |
| HTTP 服务 | Go 标准库 `net/http` |
| 数据库 | PostgreSQL |
| 向量扩展 | pgvector |
| 数据库驱动 | pgx v5 |
| Go 向量类型 | pgvector-go |
| 本地 Embedding 服务 | Ollama |
| Embedding 模型 | `qwen3-embedding:0.6b`，约 0.6B 参数、约 639 MB，支持中文、英文和代码内容 |
| 回答模型 | 通过 Eino `ChatModel` 接口接入远程 API 或 Ollama 本地模型 |
| 配置管理 | Viper |
| 数据库迁移 | go-migrations |
| 日志 | Go 标准库 `log/slog` |
| 测试 | Go testing、Testcontainers Go、PostgreSQL pgvector 测试容器 |
| 部署 | Docker Compose |

## 目标目录

```text
cmd/
  indexer/
  server/
docs/
  roadmap/
  sharing/
  projects/
  api/
  faq/
internal/
  config/
  document/
    loader/
    cleaner/
    splitter/
  embedding/
  indexer/
  retriever/
  rag/
  repository/
  transport/
migrations/
testdata/
  evaluation.jsonl
config.yaml
compose.yaml
go.mod
README.md
```

## 全局不变量

所有工作包都必须遵守：

1. 文档向量和问题向量使用同一个 Embedding 模型，维度固定为 1024。
2. 检索只能读取未删除文档的当前有效版本。
3. Embedding 等外部网络调用不能放在数据库长事务中。
4. 新版本完整生成并写入后，才能在短事务中切换为有效版本。
5. 大模型只能根据检索上下文回答，关键结论必须引用本次检索得到的来源。
6. 没有足够可靠依据或发生关键依赖错误时，系统明确拒答，不能退回模型自身知识。
7. 所有 I/O 和外部调用都接受 `context.Context`，取消和超时必须向下传递。
8. 所有依赖版本固定；配置项可通过环境变量覆盖。

## 跨模块数据契约

文档 Chunk 至少携带：

- 文档标识、源文件相对路径、文档标题。
- 完整标题层级路径、Chunk 序号、正文。
- 起始行号、结束行号、内容哈希、文档版本。
- Embedding 模型和维度。

检索结果在上述元数据基础上增加：

- 相似度。
- 本次上下文中的唯一来源编号，如 `S1`、`S2`、`S3`。

## 通用交付要求

每个工作包完成时：

- 代码保持模块边界，不让 transport、repository 或模型 SDK 细节泄漏到无关层。
- 为正常路径、边界条件和失败路径补充测试。
- 错误包含足够上下文，但不记录文档全文、凭据等敏感内容。
- 在交付说明中列出修改文件、已验证的完成标准、测试命令和遗留风险。
