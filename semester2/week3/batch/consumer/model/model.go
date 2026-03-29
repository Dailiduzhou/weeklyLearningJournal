package model

import (
	"fmt"
	"time"

	"github.com/IBM/sarama"
)

type Consumer struct {
	BatchSize int
	BatchTime time.Duration
}

func (c *Consumer) Setup(sarama.ConsumerGroupSession) error { return nil }

func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	batch := make([]*sarama.ConsumerMessage, 0, c.BatchSize)

	ticker := time.NewTicker(c.BatchTime)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				if len(batch) > 0 {
					c.processBatch(session, batch)
				}
				return nil
			}

			batch = append(batch, msg)

			if len(batch) >= c.BatchSize {
				c.processBatch(session, batch)
				batch = make([]*sarama.ConsumerMessage, 0, c.BatchSize)
				ticker.Reset(c.BatchTime)
			}
		case <-ticker.C:
			if len(batch) > 0 {
				fmt.Printf("[超时触发]距上次%v", c.BatchTime)
				c.processBatch(session, batch)
				batch = make([]*sarama.ConsumerMessage, 0, c.BatchSize)
			}
		case <-session.Context().Done():
			if len(batch) > 0 {
				fmt.Println("优雅退出")
				c.processBatch(session, batch)
			}

			return nil
		}
	}
}

func (c *Consumer) processBatch(session sarama.ConsumerGroupSession, messages []*sarama.ConsumerMessage) {
	fmt.Printf("正在处理: %d条消息", len(messages))

	for _, msg := range messages {
		session.MarkMessage(msg, "")
	}

	fmt.Println("批处理完成")
}
