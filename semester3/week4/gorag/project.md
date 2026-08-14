# Go RAG 知识库项目说明

本文件是项目需求的导航入口。详细要求已按可独立分配给 subagent 的工作包拆分到 [`project/`](project/) 目录。

## Subagent 读取规则

1. 所有 subagent 先读取 [`project/00-shared-context.md`](project/00-shared-context.md)，了解共同架构、技术约束和跨模块不变量。
2. 然后只读取自己被分配的工作包；不要默认加载其他工作包。
3. 如果任务涉及其他工作包的接口，只读取对应文件中的“边界与依赖”部分，避免扩大上下文。
4. 实现时以工作包中的“完成标准”为验收依据；全局冲突以共享上下文为准。
5. 一个 subagent 的交付说明应列出：修改文件、完成的验收项、运行的测试、遗留风险。

## 工作包索引

| 工作包 | 负责内容 | 主要代码区域 | 依赖 |
|---|---|---|---|
| [`00-shared-context.md`](project/00-shared-context.md) | 项目目标、架构、技术栈、目录和全局不变量 | 全局 | 无；所有任务必读 |
| [`10-document-pipeline.md`](project/10-document-pipeline.md) | 文档扫描、加载、清洗和结构化切分 | `internal/document` | 共享上下文 |
| [`20-embedding-and-storage.md`](project/20-embedding-and-storage.md) | Ollama Embedding、数据库模型、迁移和持久化 | `internal/embedding`、`internal/repository`、`migrations` | 文档数据结构 |
| [`30-index-lifecycle.md`](project/30-index-lifecycle.md) | 增量同步、删除、版本切换和 indexer 命令 | `internal/indexer`、`cmd/indexer` | 文档管线、Embedding、存储 |
| [`40-retrieval-and-rag.md`](project/40-retrieval-and-rag.md) | pgvector 检索、上下文构造和 Eino Chain | `internal/retriever`、`internal/rag` | Embedding、存储 |
| [`50-answer-and-api.md`](project/50-answer-and-api.md) | 回答约束、引用校验、拒答和 HTTP API | `internal/rag`、`internal/transport`、`cmd/server` | 检索与 RAG |
| [`60-testing-and-evaluation.md`](project/60-testing-and-evaluation.md) | 单元/集成测试和知识库评测集 | 各模块测试、`testdata` | 所有功能工作包 |
| [`70-deployment-and-operations.md`](project/70-deployment-and-operations.md) | Docker Compose、配置、迁移和启动检查 | `compose.yaml`、配置与启动代码 | 存储、Embedding、server/indexer |
| [`80-implementation-plan.md`](project/80-implementation-plan.md) | 分阶段实施顺序和并行边界 | 全局 | 上述全部工作包 |

## 推荐分配方式

```text
00 共享上下文
 ├─ 10 文档管线 ─────────┐
 ├─ 20 Embedding 与存储 ─┼─ 30 索引生命周期
 │                       └─ 40 检索与 RAG ── 50 回答与 API
 ├─ 60 测试与评测（随各阶段持续推进）
 └─ 70 部署与运维（先建基础设施，最后完成联调）
```

详细执行批次见 [`project/80-implementation-plan.md`](project/80-implementation-plan.md)。

## 原章节映射

| 原章节 | 新位置 |
|---|---|
| 一至四：目标、架构、选型、目录 | `00-shared-context.md` |
| 五：文档处理 | `10-document-pipeline.md` |
| 六至七：Embedding、数据库 | `20-embedding-and-storage.md` |
| 八：增量索引 | `30-index-lifecycle.md` |
| 九至十：向量检索、在线问答 | `40-retrieval-and-rag.md` |
| 十一至十三：回答、拒答、API | `50-answer-and-api.md` |
| 十四：测试 | `60-testing-and-evaluation.md` |
| 十五：部署 | `70-deployment-and-operations.md` |
| 十六：实施顺序 | `80-implementation-plan.md` |
