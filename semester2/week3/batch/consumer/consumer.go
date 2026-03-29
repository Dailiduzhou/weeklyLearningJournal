package main

import (
	"consumer/model"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
)

func main() {
	config := sarama.NewConfig()
	config.Version = sarama.V4_1_0_0
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	brokers := []string{"127.0.0.1:9092"}
	group := "test-consumer-group"
	topics := []string{"test-topic"}

	client, err := sarama.NewConsumerGroup(brokers, group, config)
	if err != nil {
		log.Fatalf("创建消费者组失败: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	wg.Add(1)

	consumer := &model.Consumer{
		BatchSize: 10,
		BatchTime: 1 * time.Second,
	}

	go func() {
		defer wg.Done()
		for {
			if err := client.Consume(ctx, topics, consumer); err != nil {
				log.Fatalf("消费过程发生错误: %v", err)
			}

			if ctx.Err() != nil {
				return
			}
		}
	}()

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM, syscall.SIGINT)
	<-sigterm

	fmt.Println("Elegantly exit")
	cancel()
	wg.Wait()
	fmt.Println("Consumer group closed successfully")
}
