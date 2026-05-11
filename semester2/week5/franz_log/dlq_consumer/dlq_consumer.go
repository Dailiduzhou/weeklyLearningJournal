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

type DLQMessage struct {
	OriginalMessage []byte `json:"original_message"`
	Error           string `json:"error"`
	Timestamp       int64  `json:"timestamp"`
	Topic           string `json:"topic"`
}

func main() {
	brokers := []string{os.Getenv("KAFKA_BROKERS")}
	if brokers[0] == "" {
		brokers = []string{"localhost:9092"}
	}

	// 创建死信队列消费者
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup("dlq-consumer-group"),
		kgo.ConsumeTopics("user_behavior_dlq"),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.AllowAutoTopicCreation(),
	)
	if err != nil {
		log.Fatalf("创建死信队列消费者失败: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("死信队列消费者已启动，正在监控失败消息...")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				fetches := client.PollFetches(ctx)

				if errs := fetches.Errors(); len(errs) > 0 {
					for _, err := range errs {
						log.Printf("死信队列消费错误: %v", err)
					}
					continue
				}

				fetches.EachPartition(func(p kgo.FetchTopicPartition) {
					for _, record := range p.Records {
						var dlqMsg DLQMessage
						if err := json.Unmarshal(record.Value, &dlqMsg); err != nil {
							log.Printf("解析死信消息失败: %v", err)
							continue
						}

						// 打印死信消息详情
						fmt.Printf("\n=== 死信队列消息 ===\n")
						fmt.Printf("时间: %s\n", time.Unix(dlqMsg.Timestamp, 0).Format("2006-01-02 15:04:05"))
						fmt.Printf("原始Topic: %s\n", dlqMsg.Topic)
						fmt.Printf("错误信息: %s\n", dlqMsg.Error)
						fmt.Printf("原始消息: %s\n", string(dlqMsg.OriginalMessage))
						fmt.Printf("==================\n")

						// 标记消息已处理
						client.MarkCommitRecords(record)
					}
				})

				if err := client.CommitUncommittedOffsets(ctx); err != nil {
					log.Printf("Commit offset error: %v", err)
				}
			}
		}
	}()

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt)
	<-sigterm
	fmt.Println("正在关闭死信队列消费者...")
	cancel()
}

