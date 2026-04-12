# Sarama 到 Franz-Go 迁移总结

## 项目结构

```
franz_log/                      # 新建项目文件夹
├── producer/
│   ├── producer.go             # 使用 kgo 异步 Producer
│   ├── go.mod                  # 依赖 github.com/twmb/franz-go
│   ├── go.sum
│   └── Dockerfile              # 与原项目相同
├── consumer/
│   ├── consumer.go             # 使用 kgo PollFetches
│   ├── go.mod                  # 依赖 github.com/twmb/franz-go
│   ├── go.sum
│   └── Dockerfile              # 与原项目相同
├── docker-compose.yml          # Redpanda 配置保留，容器名更新
└── MIGRATION.md                # 本文件
```

## 关键变更对比

### 1. Producer 变更

| 功能 | Sarama | Franz-Go (kgo) |
|------|--------|----------------|
| **初始化** | `sarama.NewAsyncProducer(brokers, config)` | `kgo.NewClient(kgo.SeedBrokers(brokers...))` |
| **消息类型** | `sarama.ProducerMessage` | `kgo.Record` |
| **异步发送** | `producer.Input() <- msg` | `client.Produce(ctx, record, callback)` |
| **错误处理** | 读取 `Errors()` channel | 在 Produce 回调中处理 `err` |
| **成功确认** | 读取 `Successes()` channel | 在 Produce 回调中确认 |

**代码示例：**
```go
// Sarama
msg := &sarama.ProducerMessage{
    Topic: topic,
    Value: sarama.ByteEncoder(eventBytes),
}
producer.Input() <- msg

// Franz-Go
client.Produce(ctx, &kgo.Record{
    Topic: topic,
    Value: eventBytes,
}, func(r *kgo.Record, err error) {
    if err != nil {
        log.Printf("Sending failure: %v", err)
    }
})
```

### 2. Consumer 变更

| 功能 | Sarama | Franz-Go (kgo) |
|------|--------|----------------|
| **初始化** | `sarama.NewConsumerGroup(brokers, groupID, config)` | `kgo.NewClient(kgo.ConsumerGroup(groupID), kgo.ConsumeTopics(...))` |
| **消费模式** | 实现 `ConsumerGroupHandler` 接口 | 调用 `client.PollFetches(ctx)` |
| **消息处理** | `ConsumeClaim()` 方法内遍历 `claim.Messages()` | `fetches.EachPartition()` 遍历 `Records` |
| **Offset 提交** | `session.MarkMessage(msg, "")` | `client.MarkCommitRecords(record)` + `client.CommitUncommittedOffsets(ctx)` |
| **Offset 策略** | `config.Consumer.Offsets.Initial = sarama.OffsetNewest` | `kgo.ConsumeResetOffset(kgo.NewOffset().AtStart())` |

**代码示例：**
```go
// Sarama - 需实现接口
type BehaviorConsumer struct{}
func (c *BehaviorConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
    for msg := range claim.Messages() {
        // 处理消息
        session.MarkMessage(msg, "")
    }
}

// Franz-Go - 直接调用
fetches := client.PollFetches(ctx)
fetches.EachPartition(func(p kgo.FetchTopicPartition) {
    for _, record := range p.Records {
        // 处理消息
        client.MarkCommitRecords(record)
    }
})
client.CommitUncommittedOffsets(ctx)
```

### 3. 依赖变更

```go
// 原依赖 (go.mod)
require github.com/IBM/sarama v1.47.0

// 新依赖 (go.mod)
require github.com/twmb/franz-go v1.20.7
```

## 保留项

✅ **Redpanda 配置完全保留**
- `docker-compose.yml` 中的 Redpanda 服务配置未做任何修改
- 镜像版本: `v24.3.5`
- 端口映射: `19092`, `18082`, `18081`
- 健康检查配置

✅ **业务逻辑不变**
- `UserEvent` 结构体保持不变
- `startMetricsEngine` 统计引擎逻辑完全保留
- 每 10 秒打印统计面板的逻辑不变
- UV 和 ActionCount 统计方式不变

✅ **配置参数不变**
- Group ID: `"analytics-group"` 保持不变
- Topic: `"user_behavior"` 保持不变
- 环境变量 `KAFKA_BROKERS` 用法不变
- 默认 broker: `"localhost:9092"`

## 运行方式

```bash
# 启动所有服务
cd franz_log
docker-compose up --build

# 单独启动
docker-compose up -d redpanda  # 先启动 redpanda
docker-compose up -d consumer  # 再启动 consumer
docker-compose up -d producer  # 最后启动 producer

# 查看日志
docker-compose logs -f consumer
docker-compose logs -f producer
```

## 优势对比

| 特性 | Sarama | Franz-Go |
|------|--------|----------|
| 维护状态 | IBM 维护，更新较慢 | 活跃维护，更新频繁 |
| API 设计 | 较复杂，需实现接口 | 简洁，函数式 API |
| 性能 | 中等 | 更高性能 |
| 依赖数量 | 较多（含 kerberos 等） | 较少 |
| 上下文支持 | 有限 | 原生 context 支持 |
| 错误处理 | Channel 方式 | 回调方式 |

## 已知差异

1. **Offset 策略**: 原项目使用 `OffsetNewest`（从最新开始），新项目明确使用 `AtStart()`（从头开始）
2. **容器名**: `sarama-consumer` → `franz-consumer`, `sarama-producer` → `franz-producer`
