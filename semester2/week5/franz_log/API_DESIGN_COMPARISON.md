# Franz-Go vs Sarama：API 设计与封装深度对比

## 1. 架构设计哲学

### Sarama：面向对象 + 回调接口
```
┌─────────────────────────────────────────────────────────────┐
│                     Sarama Architecture                      │
├─────────────────────────────────────────────────────────────┤
│  核心思想：接口驱动（Interface-driven）                      │
│  设计模式：Strategy Pattern + Observer Pattern               │
│  耦合度：中高（需实现特定接口）                              │
└─────────────────────────────────────────────────────────────┘
```

**特点：**
- 大量使用 Go 接口定义行为契约
- 消费者需要实现 `ConsumerGroupHandler` 接口
- 通过 Channel 进行异步通信
- 配置和实现分离

### Franz-Go：函数式 + 构建器模式
```
┌─────────────────────────────────────────────────────────────┐
│                   Franz-Go Architecture                    │
├─────────────────────────────────────────────────────────────┤
│  核心思想：函数式配置（Functional Options）                  │
│  设计模式：Builder Pattern + Fluent Interface                │
│  耦合度：低（直接调用 API）                                  │
└─────────────────────────────────────────────────────────────┘
```

**特点：**
- 使用 Functional Options 模式进行配置
- 统一的 `Client` 接口处理生产和消费
- 基于回调的错误处理
- 原生支持 Go 的 `context` 包

---

## 2. 初始化方式对比

### Sarama：配置对象模式

```go
// 1. 创建配置对象（多步骤配置）
config := sarama.NewConfig()
config.Producer.Return.Successes = true      // 必须显式启用
config.Producer.Return.Errors = true         // 必须显式启用
config.Producer.Retry.Max = 3                // 重试配置
config.Producer.RequiredAcks = sarama.WaitForAll
config.Net.MaxOpenRequests = 5
config.Net.DialTimeout = 10 * time.Second
config.Metadata.RefreshFrequency = 10 * time.Minute

// 2. 创建生产者（需要单独处理错误）
producer, err := sarama.NewAsyncProducer(brokers, config)
if err != nil {
    log.Fatal(err)
}
defer producer.AsyncClose()

// 3. 创建消费者（需要单独配置）
consumerConfig := sarama.NewConfig()
consumerConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
consumerConfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin

consumer, err := sarama.NewConsumerGroup(brokers, groupID, consumerConfig)
if err != nil {
    log.Fatal(err)
}
```

**优点：**
- 配置项分类清晰（Producer/Consumer/Net/Metadata）
- 适合需要精细控制的场景

**缺点：**
- 配置分散，学习曲线陡峭
- 不同组件需要不同配置对象
- 默认行为不够直观

---

### Franz-Go：Functional Options 模式

```go
// 1. 生产者客户端（一行配置）
producer, err := kgo.NewClient(
    kgo.SeedBrokers(brokers...),
    kgo.WithProduceRequestTimeout(10*time.Second),
    kgo.WithMaxProduceRequests(5),
    kgo.RecordRetries(3),
    kgo.RequiredAcks(kgo.AllISRAcks()),
    kgo.AllowAutoTopicCreation(),
)

// 2. 消费者客户端（复用相同模式）
consumer, err := kgo.NewClient(
    kgo.SeedBrokers(brokers...),
    kgo.ConsumerGroup("my-group"),
    kgo.ConsumeTopics("topic1", "topic2"),
    kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
    kgo.AutoCommitInterval(5*time.Second),
)

// 3. 甚至可以统一创建（同一客户端既可生产也可消费）
client, err := kgo.NewClient(
    kgo.SeedBrokers(brokers...),
    // 生产配置
    kgo.RequiredAcks(kgo.AllISRAcks()),
    // 消费配置
    kgo.ConsumerGroup("my-group"),
    kgo.ConsumeTopics("topic1"),
)
```

**优点：**
- 自描述配置（从函数名就能看懂作用）
- 可选参数灵活组合
- 生产/消费可共用同一客户端

**缺点：**
- 配置项较多时行数增加
- 需要熟悉 kgo 包中的各种 Option 函数

---

## 3. 生产者 API 深度对比

### Sarama：Channel-based Async

```go
// 消息定义 - 字段多但大多可选
type ProducerMessage struct {
    Topic          string            // 必须
    Key            Encoder           // 可选
    Value          Encoder           // 必须
    Headers        []RecordHeader    // 可选
    Metadata       interface{}       // 可选（用户自定义数据）
    Offset         int64             // 内部使用
    Partition      int32             // 内部使用
    Timestamp      time.Time         // 内部使用
    retries        int               // 内部使用
    flags          int               // 内部使用
}

// 发送消息 - 非阻塞 Channel 操作
msg := &sarama.ProducerMessage{
    Topic: "my-topic",
    Key:   sarama.StringEncoder("user-123"),     // 需要显式 Encoder
    Value: sarama.ByteEncoder(data),               // 需要显式 Encoder
}

// 发送到内部 Channel（可能阻塞如果 Channel 满）
producer.Input() <- msg

// 必须启动 goroutine 监听结果
go func() {
    for {
        select {
        case success := <-producer.Successes():
            fmt.Printf("Success: %s/%d/%d\n", 
                success.Topic, success.Partition, success.Offset)
        case err := <-producer.Errors():
            fmt.Printf("Failed: %v\n", err.Err)
        }
    }
}()
```

**设计分析：**
```
┌────────────────────────────────────────────┐
│  Sarama Producer Flow                      │
├────────────────────────────────────────────┤
│                                            │
│  [User] ──Input()──▶ [Internal Queue]      │
│                       │                    │
│                       ▼                    │
│              [Partitioner]                 │
│                       │                    │
│                       ▼                    │
│              [Broker Connection]           │
│                       │                    │
│           ┌──────────┴──────────┐         │
│           ▼                     ▼         │
│   [Successes]              [Errors]       │
│      Channel                Channel       │
│           │                     │         │
│           └──────────┬──────────┘         │
│                      ▼                    │
│              [User Handler]               │
│                                            │
└────────────────────────────────────────────┘

特点：
- 基于 Channel 的 CSP (Communicating Sequential Processes) 模型
- 用户需要管理两个监听 goroutine
- Encoder 接口需要显式包装数据
```

---

### Franz-Go：Callback-based Async

```go
// 消息定义 - 简单直接
type Record struct {
    Context context.Context    // 支持 context 传播
    Topic   string             // 必须
    Key     []byte             // 直接字节切片
    Value   []byte             // 直接字节切片
    Headers []Header           // 可选
    Offset  int64              // 消费时使用
    Partition int32            // 消费时使用
    Timestamp time.Time        // 消费时使用
}

// 发送消息 - 带 Promise 回调
record := &kgo.Record{
    Topic: "my-topic",
    Key:   []byte("user-123"),      // 直接 []byte
    Value: data,                     // 直接 []byte
}

// 异步发送 + 回调（无需监听 Channel）
client.Produce(ctx, record, func(r *kgo.Record, err error) {
    if err != nil {
        log.Printf("Failed: %v", err)
        return
    }
    log.Printf("Success: %s/%d/%d", 
        r.Topic, r.Partition, r.Offset)
})

// 也可以同步发送（阻塞等待）
results := client.ProduceSync(ctx, record1, record2, record3)
for _, result := range results {
    if result.Err != nil {
        log.Printf("Failed: %v", result.Err)
    }
}
```

**设计分析：**
```
┌────────────────────────────────────────────┐
│  Franz-Go Producer Flow                    │
├────────────────────────────────────────────┤
│                                            │
│  [User] ──Produce()──▶ [Client]            │
│                          │                 │
│              ┌───────────┴───────────┐    │
│              ▼                       ▼    │
│        [Internal Queue]      [Promise]    │
│              │                       │    │
│              ▼                       ▼    │
│        [Sequencer]            [Callback]  │
│              │                       │    │
│              ▼                       ▼    │
│        [Broker Conn]          [User Code] │
│              │                            │
│              ▼                            │
│        [Result Trigger]                   │
│              │                            │
│              └──────────▶ [Promise]       │
│                              │            │
│                              ▼            │
│                        [Callback]         │
│                                            │
└────────────────────────────────────────────┘

特点：
- 基于 Promise/Callback 模式
- 无需管理额外 goroutine
- 支持 context 传播（取消、超时）
- 数据直接 []byte，无需 Encoder 包装
```

---

## 4. 消费者 API 深度对比

### Sarama：接口实现模式

```go
// 必须实现 3 个接口方法
type ConsumerGroupHandler interface {
    Setup(session ConsumerGroupSession) error      // 消费开始前调用
    Cleanup(session ConsumerGroupSession) error    // 消费结束后调用
    ConsumeClaim(session ConsumerGroupSession, 
                  claim ConsumerGroupClaim) error  // 实际消费逻辑
}

// 实现示例
type MyConsumer struct {
    ready chan bool
}

func (c *MyConsumer) Setup(s sarama.ConsumerGroupSession) error {
    fmt.Println("Consumer group 重新平衡完成")
    close(c.ready)
    return nil
}

func (c *MyConsumer) Cleanup(s sarama.ConsumerGroupSession) error {
    fmt.Println("Consumer group 即将重新平衡")
    return nil
}

func (c *MyConsumer) ConsumeClaim(
    session sarama.ConsumerGroupSession, 
    claim sarama.ConsumerGroupClaim,
) error {
    // 遍历消息 Channel
    for msg := range claim.Messages() {
        // 处理消息
        fmt.Printf("收到: %s = %s\n", msg.Topic, string(msg.Value))
        
        // 手动提交 offset
        session.MarkMessage(msg, "")
    }
    return nil
}

// 使用方式
func main() {
    consumer := &MyConsumer{ready: make(chan bool)}
    
    ctx := context.Background()
    client, _ := sarama.NewConsumerGroup(brokers, groupID, config)
    
    // 需要无限循环处理重平衡
    for {
        err := client.Consume(ctx, []string{"topic"}, consumer)
        if err != nil {
            log.Fatal(err)
        }
        if ctx.Err() != nil {
            return
        }
        consumer.ready = make(chan bool)
    }
}
```

**设计分析：**
```
┌────────────────────────────────────────────┐
│  Sarama Consumer Architecture              │
├────────────────────────────────────────────┤
│                                            │
│  ┌─────────────┐     ┌─────────────┐      │
│  │ ConsumerGroup │──▶│ Coordinator │      │
│  └─────────────┘     └─────────────┘      │
│         │                                  │
│         ▼                                  │
│  ┌─────────────┐     ┌─────────────┐      │
│  │   Setup()   │     │  Cleanup()  │      │
│  │   回调      │     │   回调      │      │
│  └─────────────┘     └─────────────┘      │
│         │                                  │
│         ▼                                  │
│  ┌─────────────────────────────────────┐  │
│  │      ConsumeClaim() 回调            │  │
│  │  ┌─────────────────────────────┐   │  │
│  │  │ for msg := range messages   │   │  │
│  │  │     process(msg)            │   │  │
│  │  │     session.MarkMessage()   │   │  │
│  │  └─────────────────────────────┘   │  │
│  └─────────────────────────────────────┘  │
│                                            │
└────────────────────────────────────────────┘

问题：
- 需要理解 Consumer Group 重平衡机制
- 必须手动处理无限循环
- 接口实现繁琐
- Setup/Cleanup 在重平衡时多次调用
```

---

### Franz-Go：直接轮询模式

```go
// 无需实现任何接口！
func main() {
    // 创建客户端
    client, _ := kgo.NewClient(
        kgo.SeedBrokers(brokers...),
        kgo.ConsumerGroup("my-group"),
        kgo.ConsumeTopics("topic"),
        kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
    )
    defer client.Close()

    ctx := context.Background()
    
    // 简单循环轮询
    for {
        // 拉取消息（阻塞直到有消息或超时）
        fetches := client.PollFetches(ctx)
        
        // 处理错误
        if errs := fetches.Errors(); len(errs) > 0 {
            for _, err := range errs {
                log.Printf("Poll error: %v", err)
            }
            continue
        }
        
        // 处理消息
        iter := fetches.RecordIter()
        for !iter.Done() {
            record := iter.Next()
            
            // 处理消息
            fmt.Printf("收到: %s = %s\n", record.Topic, string(record.Value))
            
            // 标记为已处理（异步提交）
            client.MarkCommitRecords(record)
        }
        
        // 手动提交 offset
        if err := client.CommitUncommittedOffsets(ctx); err != nil {
            log.Printf("Commit error: %v", err)
        }
    }
}
```

**设计分析：**
```
┌────────────────────────────────────────────┐
│  Franz-Go Consumer Architecture            │
├────────────────────────────────────────────┤
│                                            │
│  ┌─────────────┐                           │
│  │   Client    │                           │
│  └─────────────┘                           │
│         │                                  │
│         ▼                                  │
│  ┌──────────────────────────────┐         │
│  │      PollFetches()           │         │
│  │  ┌────────────────────────┐  │         │
│  │  │  内部处理重平衡        │  │         │
│  │  │  管理 partition 分配   │  │         │
│  │  └────────────────────────┘  │         │
│  └──────────────────────────────┘         │
│         │                                  │
│         ▼                                  │
│  ┌──────────────────────────────┐         │
│  │       Fetches 结果          │         │
│  │  ┌────────────────────────┐  │         │
│  │  │  iter.Next()           │  │         │
│  │  │  process(record)       │  │         │
│  │  │  MarkCommitRecords()   │  │         │
│  │  └────────────────────────┘  │         │
│  └──────────────────────────────┘         │
│         │                                  │
│         ▼                                  │
│  ┌──────────────────────────────┐         │
│  │   CommitUncommittedOffsets() │         │
│  └──────────────────────────────┘         │
│                                            │
└────────────────────────────────────────────┘

优势：
- 无需理解重平衡细节
- 无接口实现负担
- 流程线性清晰
- 原生 context 支持
```

---

## 5. 配置系统对比

### Sarama：结构化配置

```go
type Config struct {
    // 网络配置
    Net struct {
        MaxOpenRequests int
        DialTimeout     time.Duration
        ReadTimeout     time.Duration
        WriteTimeout    time.Duration
        TLS             struct {
            Enable bool
            Config *tls.Config
        }
        SASL struct {
            Enable    bool
            Mechanism sarama.SASLMechanism
            User      string
            Password  string
        }
    }

    // 元数据配置
    Metadata struct {
        Retry struct {
            Max         int
            Backoff     time.Duration
        }
        RefreshFrequency time.Duration
        Full             bool
    }

    // 生产者配置
    Producer struct {
        MaxMessageBytes  int
        RequiredAcks     RequiredAcks
        Timeout          time.Duration
        Compression      CompressionCodec
        Partitioner      PartitionerConstructor
        Return           struct {
            Successes bool
            Errors    bool
        }
        // ... 更多字段
    }

    // 消费者配置
    Consumer struct {
        Retry struct {
            Backoff     time.Duration
        }
        Fetch struct {
            Min     int32
            Default int32
            Max     int32
        }
        MaxWaitTime       time.Duration
        MaxProcessingTime time.Duration
        Offsets           struct {
            CommitInterval time.Duration
            Initial        int64
            Retention      time.Duration
        }
        // ... 更多字段
    }
}
```

**优点：**
- 类型安全
- IDE 自动补全友好
- 配置结构清晰

**缺点：**
- 嵌套层次深
- 默认值不明确
- 需要阅读文档了解每个字段

---

### Franz-Go：函数式配置

```go
// 每个配置都是一个函数
func NewClient(opts ...Opt) (*Client, error)

// Opt 类型定义
type Opt interface {
    apply(*cfg)
}

// 配置示例（按类别分组）

// ===== 连接配置 =====
kgo.SeedBrokers("localhost:9092", "localhost:9093")
kgo.WithDialTimeout(10 * time.Second)
kgo.WithRequestTimeout(30 * time.Second)
kgo.WithMaxBrokerWriteBytes(100 << 20)

// ===== 生产者配置 =====
kgo.RequiredAcks(kgo.AllISRAcks())          // 等待所有副本确认
kgo.RecordRetries(3)                         // 重试3次
kgo.RecordDeliveryTimeout(10 * time.Second)  // 发送超时
kgo.MaxBufferedRecords(10000)                // 缓冲队列大小
kgo.CompressionCodec(kgo.ZstdCompression())  // 压缩算法

// ===== 消费者配置 =====
kgo.ConsumerGroup("my-group")
kgo.ConsumeTopics("topic1", "topic2")
kgo.ConsumePartitions(map[string]map[int32]kgo.Offset{
    "topic1": {0: kgo.NewOffset().At(1234)},
})
kgo.ConsumeResetOffset(kgo.NewOffset().AfterMilli(time.Now().Add(-1*time.Hour).UnixMilli()))
kgo.AutoCommitInterval(5 * time.Second)
kgo.BlockRebalanceOnPoll()  // 在 Poll 期间阻塞重平衡

// ===== 事务配置 =====
kgo.TransactionalID("my-tx-id")
kgo.TransactionTimeout(1 * time.Minute)
```

**优点：**
- 自文档化（函数名就是说明）
- 可组合性强
- 默认值合理（零值即合理默认）
- 支持动态配置更新

**缺点：**
- 需要导入包才能看到所有选项
- 参数多时行数增加

---

## 6. 错误处理对比

### Sarama：Channel + Error 类型

```go
// 错误通过 Channel 传递
type ProducerError struct {
    Msg *ProducerMessage    // 原始消息
    Err error               // 错误原因
}

// 处理方式
go func() {
    for err := range producer.Errors() {
        // 需要手动关联原始消息
        log.Printf("Message to %s failed: %v", 
            err.Msg.Topic, err.Err)
        
        // 可能的重试逻辑
        if err.Err == sarama.ErrNotLeaderForPartition {
            // 等待元数据刷新后重试
        }
    }
}()

// 消费者错误在 ConsumeClaim 中返回
func (c *MyConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, 
    claim sarama.ConsumerGroupClaim) error {
    for msg := range claim.Messages() {
        if err := process(msg); err != nil {
            // 返回错误会导致重新平衡
            return err
        }
    }
    return nil
}
```

---

### Franz-Go：Callback + Explicit Error

```go
// 生产者错误在回调中处理
client.Produce(ctx, record, func(r *kgo.Record, err error) {
    if err != nil {
        // err 包含详细的错误类型信息
        switch {
        case errors.Is(err, kerr.UnknownTopicOrPartition):
            log.Printf("Topic 不存在: %v", err)
        case errors.Is(err, kerr.NotLeaderForPartition):
            log.Printf("分区 Leader 变化: %v", err)
        default:
            log.Printf("发送失败: %v", err)
        }
        return
    }
    // 成功处理
})

// 消费者错误通过 PollFetches 返回
type FetchError struct {
    Topic     string
    Partition int32
    Err       error
}

// 显式错误处理
fetches := client.PollFetches(ctx)
if errs := fetches.Errors(); len(errs) > 0 {
    for _, err := range errs {
        log.Printf("Partition %s/%d error: %v",
            err.Topic, err.Partition, err.Err)
    }
}

// 使用 Record 上的错误信息
fetches.EachPartition(func(fp kgo.FetchTopicPartition) {
    if fp.Err != nil {
        // 整个分区错误
        log.Printf("Partition error: %v", fp.Err)
        return
    }
    for _, record := range fp.Records {
        // record 级别的错误检查
        if record.Err != nil {
            log.Printf("Record error: %v", record.Err)
            continue
        }
        // 处理正常记录
    }
})
```

---

## 7. 扩展性对比

### Sarama：接口扩展

```go
// 自定义 Partitioner
type MyPartitioner struct{}

func (p *MyPartitioner) Partition(msg *ProducerMessage, 
    numPartitions int32) (int32, error) {
    // 自定义分区逻辑
    return hash(msg.Key) % numPartitions, nil
}

func (p *MyPartitioner) RequiresConsistency() bool {
    return true
}

// 使用自定义 Partitioner
config := sarama.NewConfig()
config.Producer.Partitioner = func(topic string) sarama.Partitioner {
    return &MyPartitioner{}
}

// 自定义 Encoder
type MyEncoder struct {
    Data []byte
}

func (e MyEncoder) Length() int {
    return len(e.Data)
}

func (e MyEncoder) Encode() ([]byte, error) {
    return e.Data, nil
}
```

---

### Franz-Go：函数扩展

```go
// 自定义分区器（通过函数）
partitioner := func(topic string, key, value []byte, 
    numPartitions int32) int32 {
    return int32(hash(key) % uint32(numPartitions))
}

client, _ := kgo.NewClient(
    kgo.SeedBrokers(brokers...),
    kgo.StickyPartitioning(pNum, time.Minute),
    // 或使用手动分区
)

// Hook 系统（更强大的扩展点）
client, _ := kgo.NewClient(
    kgo.WithHooks(&myHooks{}),
)

type myHooks struct{}

func (h *myHooks) OnProduceRecordBuffered(r *kgo.Record) {
    metrics.Inc("records.buffered")
}

func (h *myHooks) OnProduceRecordUnbuffered(r *kgo.Record, 
    err error) {
    if err != nil {
        metrics.Inc("records.failed")
    } else {
        metrics.Inc("records.success")
    }
}

// 拦截器模式
client, _ := kgo.NewClient(
    kgo.WithRecordTransformers(func(r *kgo.Record) *kgo.Record {
        // 修改记录（如添加 header）
        r.Headers = append(r.Headers, kgo.Header{
            Key:   "x-processed-by",
            Value: []byte("my-app"),
        })
        return r
    }),
)
```

---

## 8. 代码复杂度对比

### 实现相同功能：生产 + 消费

| 指标 | Sarama | Franz-Go |
|------|--------|----------|
| **代码行数** | ~80 行 | ~50 行 |
| **接口实现** | 3 个方法 | 0 个 |
| **Goroutine 管理** | 需手动管理 2-3 个 | 自动管理 |
| **Error Channel** | 2 个 Channel | 0 个 Channel |
| **Context 支持** | 有限 | 原生支持 |
| **学习曲线** | 陡峭 | 平缓 |

---

## 9. 适用场景建议

### 选择 Sarama 如果：
- 需要与现有基于 Sarama 的代码集成
- 需要非常精细的 Kafka 协议控制
- 团队已有 Sarama 经验
- 需要特定的 SASL/GSSAPI (Kerberos) 认证

### 选择 Franz-Go 如果：
- 新项目或重构旧项目
- 追求简洁的 API 设计
- 需要高性能（更少的内存分配）
- 希望原生 context 支持
- 需要生产/消费统一客户端

---

## 10. 总结

| 维度 | Sarama | Franz-Go |
|------|--------|----------|
| **API 风格** | 面向对象、接口驱动 | 函数式、构建器模式 |
| **复杂度** | 中高（需实现接口） | 低（直接调用） |
| **灵活性** | 高（可替换各组件） | 中高（通过 Hooks） |
| **性能** | 中等 | 优秀 |
| **维护状态** | 较慢 | 活跃 |
| **Go 惯用法** | 传统 | 现代（context） |
| **文档质量** | 一般 | 优秀 |

**最终建议：**
- **新项目**：首选 Franz-Go
- **遗留项目**：评估迁移成本后决定是否迁移
- **学习目的**：两者都学，理解不同设计哲学
