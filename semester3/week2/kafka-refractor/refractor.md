## 将`sarama`迁移至`franz-go` API 

`AGENTS.md`
```markdown
## 背景

生产级代码，kafka+sarama已经运行。

## 任务要求
1. 将be-classlist_v2中的`sarama`更换为`franz-go`，尽量做到行为对应。(包括Kafka内部rebalance细节。如果不能做到，讲解理由，写注释)
2. 整理生产者和消费者逻辑，将生产者和消费者的代码逻辑分离出来，不要混在业务代码中。增加可维护性。

## 读写边界

目前在单独的git branch上，仅能对`be-classlist_v2/`中的文件进行更改，并提交本地commit。
禁止外部的写操作和云端git操作。

## 步骤

1. 了解`be-classlist_v2`其中的业务逻辑和sarama生产者和消费者的具体代码位置和逻辑。（使用subagents)
2. 了解`franz-go`的源码和用法。
3. 从代码工程角度，确定生产者和消费者逻辑在代码库的位置和重构计划。
4. 着手重写，使用本地commits。（对可以并行的任务，使用subagents）

## 补充信息

所有仓库建立了`codegraph`索引，本地有`ripgrep`。
`franz-go`的源码在本地`/home/mikufan/fork/franz-go/`，记得查询。

```

## 重构获取课表代码逻辑

```markdown
## 背景

生产级代码，kafka+franz-go已经运行。

## 任务要求

解除获取课表部分的代码耦合。分离爬虫、重试、获取本地课表，错误处理等部分的逻辑。
现在不添加DLQ，但保持可扩展性。

## 读写边界

目前在单独的git branch上，仅能对`be-classlist_v2/`中的文件进行更改，并提交本地commit。
禁止外部的写操作和云端git操作。

## 步骤

1. 了解`be-classlist_v2`其中的获取课表部分的代码逻辑和消息队列使用细节。（使用subagents)
2. 从代码工程角度，确定职责分离方式和重构计划。
3. 着手重写，使用本地commits。（对可以并行的任务，使用subagents）

## 补充信息

所有仓库建立了`codegraph`索引，本地有`ripgrep`。
`franz-go`的源码在本地`/home/mikufan/fork/franz-go/`。

```
