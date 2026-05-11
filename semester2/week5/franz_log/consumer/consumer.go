package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

type UserEvent struct {
	UserID    int    `json:"user_id"`
	Actions   string `json:"action"`
	Timestamp int64  `json:"timestamp"`
}

type DLQMessage struct {
	OriginalMessage []byte `json:"original_message"`
	Error           string `json:"error"`
	Timestamp       int64  `json:"timestamp"`
	Topic           string `json:"topic"`
}

type WindowStat struct {
	UVMap       map[int]struct{}
	ActionCount map[string]int
}

func main() {
	eventChan := make(chan UserEvent, 5000)

	go startMetricsEngine(eventChan)

	brokers := []string{os.Getenv("KAFKA_BROKERS")}
	if brokers[0] == "" {
		brokers = []string{"localhost:9092"}
	}

	// 创建死信队列生产者
	dlqClient, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.AllowAutoTopicCreation(),
	)
	if err != nil {
		log.Fatalf("创建死信队列生产者失败: %v", err)
	}
	defer dlqClient.Close()

	// 创建 kgo 客户端，配置消费者组
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup("analytics-group"),
		kgo.ConsumeTopics("user_behavior"),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()), // 从头开始消费
		kgo.AllowAutoTopicCreation(),
	)
	if err != nil {
		log.Fatalf("创建消费者组失败: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("分析消费者已启动，正在实时统计中...")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// 拉取消息
				fetches := client.PollFetches(ctx)

				// 处理错误
				if errs := fetches.Errors(); len(errs) > 0 {
					for _, err := range errs {
						log.Printf("消费错误: %v", err)
					}
					continue
				}

				// 处理消息
				fetches.EachPartition(func(p kgo.FetchTopicPartition) {
					for _, record := range p.Records {
						var evt UserEvent
						if err := json.Unmarshal(record.Value, &evt); err != nil {
							log.Printf("Unmarshaling failure: %v", err)
							// 发送到死信队列
							sendToDLQ(dlqClient, record, err)
							continue
						}

						eventChan <- evt
						// 标记消息已处理（用于提交 offset）
						client.MarkCommitRecords(record)
					}
				})

				// 提交已处理的 offset
				if err := client.CommitUncommittedOffsets(ctx); err != nil {
					log.Printf("Commit offset error: %v", err)
				}
			}
		}
	}()

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt)
	<-sigterm
	fmt.Println("正在关闭消费者...")
	cancel()
}

// startMetricsEngine 实时流计算引擎
func startMetricsEngine(eventChan <-chan UserEvent) {
	// stats 按照分钟级时间戳 (int64) 进行聚合
	stats := make(map[int64]*WindowStat)

	// 使用一个定时器，每 10 秒打印一次当前（及上一分钟）的统计状态，增强练手体验的可见性
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-eventChan:
			// 提取所在分钟的基准时间戳 (例如: 1710000055 -> 1710000000)
			minuteStamp := event.Timestamp - (event.Timestamp % 60)

			// 如果这一分钟还没被初始化，则初始化
			if _, exists := stats[minuteStamp]; !exists {
				stats[minuteStamp] = &WindowStat{
					UVMap:       make(map[int]struct{}),
					ActionCount: make(map[string]int),
				}
			}

			// 更新该分钟的统计数据
			window := stats[minuteStamp]
			window.UVMap[event.UserID] = struct{}{} // 记录去重后的 UV
			window.ActionCount[event.Actions]++     // 累加对应 action 行为

		case <-ticker.C:
			// 定时触发：打印内存中收集到的统计信息
			fmt.Println("\n=== 最新每分钟数据统计面板 ===")

			// 计算当前的分钟级时间戳
			currentMinute := time.Now().Unix()
			currentMinute = currentMinute - (currentMinute % 60)

			// 我们打印当前分钟和上一分钟的数据
			for _, mStamp := range []int64{currentMinute - 60, currentMinute} {
				if window, exists := stats[mStamp]; exists {
					timeStr := time.Unix(mStamp, 0).Format("15:04:00")
					fmt.Printf("[时间窗口 %s] | UV (独立用户): %3d | 动作统计: Login=%3d, Click=%3d, Logout=%3d\n",
						timeStr,
						len(window.UVMap),
						window.ActionCount["login"],
						window.ActionCount["click"],
						window.ActionCount["logout"])
				}
			}

			// （可选操作）为了防止内存泄漏，这里可以清理掉过期的旧窗口数据
			for ts := range stats {
				if ts < currentMinute-60 {
					delete(stats, ts)
				}
			}
		}
	}
}

// sendToDLQ 发送失败消息到死信队列
func sendToDLQ(dlqClient *kgo.Client, record *kgo.Record, processingErr error) {
	dlqMsg := DLQMessage{
		OriginalMessage: record.Value,
		Error:           processingErr.Error(),
		Timestamp:       time.Now().Unix(),
		Topic:           record.Topic,
	}

	dlqMsgBytes, err := json.Marshal(dlqMsg)
	if err != nil {
		log.Printf("序列化死信消息失败: %v", err)
		return
	}

	// 发送到死信队列 topic
	dlqClient.Produce(context.Background(), &kgo.Record{
		Topic: "user_behavior_dlq",
		Value: dlqMsgBytes,
	}, func(r *kgo.Record, err error) {
		if err != nil {
			log.Printf("发送到死信队列失败: %v", err)
		} else {
			log.Printf("消息已发送到死信队列: %s", string(record.Value))
		}
	})
}
