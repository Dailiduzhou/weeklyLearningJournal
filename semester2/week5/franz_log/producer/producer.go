package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

type UserEvent struct {
	UserID    int    `json:"user_id"`
	Action    string `json:"action"`
	Timestamp int64  `json:"timestamp"`
}

func main() {
	brokers := []string{os.Getenv("KAFKA_BROKERS")}
	if brokers[0] == "" {
		brokers = []string{"localhost:9092"}
	}

	// 创建 kgo 客户端
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.AllowAutoTopicCreation(), // 允许自动创建 topic
	)
	if err != nil {
		log.Fatalf("Cannot init producer: %v", err)
	}
	defer client.Close()

	topic := "user_behavior"
	actions := []string{"login", "logout", "click", "click"}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				event := UserEvent{
					UserID:    r.Intn(100) + 50,
					Action:    actions[r.Intn(len(actions))],
					Timestamp: time.Now().Unix(),
				}

				eventBytes, err := json.Marshal(event)
				if err != nil {
					log.Printf("JSON marshaling error: %v", err)
					continue
				}

				// 异步发送消息，使用 promise 回调处理结果
				client.Produce(ctx, &kgo.Record{
					Topic: topic,
					Value: eventBytes,
				}, func(r *kgo.Record, err error) {
					if err != nil {
						log.Printf("Sending failure: %v", err)
					}
				})

				time.Sleep(time.Millisecond * 50)
			}
		}
	}()

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt)
	<-sigterm
	log.Printf("Producer closed")
}
