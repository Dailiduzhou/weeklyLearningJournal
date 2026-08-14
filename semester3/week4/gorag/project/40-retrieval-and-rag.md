# 40 向量检索与 RAG Chain

## 任务目标

实现问题向量化、pgvector 相似度检索、结果过滤、上下文构造，以及从问题到 ChatModel 输出的 Eino Chain。

主要代码区域：

```text
internal/retriever
internal/rag
```

## 检索要求

- 使用 Cosine Similarity。
- 知识库规模较小时优先使用精确检索。
- 只有当 Chunk 数量和查询延迟明显增加后，才引入 HNSW 索引。

查询过程：

1. 对问题进行 Embedding。
2. 从 pgvector 检索候选 Chunk。
3. 获取相似度最高的若干结果。
4. 过滤低于阈值的结果。
5. 选择最终上下文。
6. 按相关度排序。

初始参数：

| 参数 | 初始值 |
|---|---|
| 候选 Top K | 8–12 |
| 最终上下文 | 最多 5–6 个 Chunk |
| 相似度阈值 | 0.5 |

参数必须可配置；最终值通过 `60-testing-and-evaluation.md` 中的测试数据调整。

## `PgVectorRetriever`

需要自行实现符合 Eino `Retriever` 接口的 `PgVectorRetriever`，负责：

1. 调用 Embedding 组件生成问题向量。
2. 使用 pgx 查询 pgvector。
3. 将结果转换为 Eino `Document`。
4. 将源文件路径、标题层级、起止行号和相似度写入元数据。

数据库查询必须只返回未删除文档的当前有效版本。

## Eino Chain 顺序

```text
问题输入
→ 问题 Embedding
→ PgVectorRetriever
→ 相似度过滤
→ Context Builder
→ ChatTemplate
→ ChatModel
→ 引用校验
→ 响应输出
```

其中引用校验和最终 HTTP 响应由 [`50-answer-and-api.md`](50-answer-and-api.md) 定义。本工作包需为该阶段保留清晰接口。

## Context Builder

传入大模型的每个来源必须有本次请求内唯一且稳定的编号，例如 `S1`、`S2`、`S3`。

每个上下文来源包含：

- 来源编号。
- 源文件路径。
- 标题层级。
- 起止行号。
- Chunk 正文。

来源编号与检索结果元数据的映射必须保留，供输出和引用校验使用。

## 错误边界

- Embedding 或数据库检索失败时返回可识别错误，由回答层统一拒答。
- 没有候选结果、最高相似度低于阈值或过滤后为空时，不调用 ChatModel。
- 所有链路传播 Context 取消和超时。
- 不在该层静默使用模型自身知识或补造来源。

## 完成标准

- 使用固定输入向量的测试证明 Cosine 排序、Top K 和阈值过滤正确。
- `PgVectorRetriever` 返回 Eino Document 和完整来源元数据。
- Context Builder 的编号唯一、顺序稳定，且能反向映射到检索结果。
- 空结果和低相关结果在调用 ChatModel 前终止。
- 集成测试证明删除文档和旧版本不会出现在结果中。
- 检索参数可配置，并为评测调参提供明确入口。
