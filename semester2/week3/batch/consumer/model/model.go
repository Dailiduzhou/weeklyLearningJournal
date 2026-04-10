package model

import (
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
)

type Consumer struct {
	BatchSize  int
	BatchTime  time.Duration
	WorkerNum  int
	MaxRetries int

	workerPool chan struct{}
	wg         sync.WaitGroup
}

func (c *Consumer) Setup(sarama.ConsumerGroupSession) error {
	if c.WorkerNum <= 0 {
		c.WorkerNum = 10
	}
	if c.MaxRetries <= 0 {
		c.MaxRetries = 3
	}
	c.workerPool = make(chan struct{}, c.WorkerNum)
	return nil
}

func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	c.wg.Wait()
	return nil
}

func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	batch := make([]*sarama.ConsumerMessage, 0, c.BatchSize)
	ticker := time.NewTicker(c.BatchTime)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				if len(batch) > 0 {
					c.submitBatch(session, batch)
				}
				return nil
			}

			batch = append(batch, msg)

			if len(batch) >= c.BatchSize {
				c.submitBatch(session, batch)
				batch = make([]*sarama.ConsumerMessage, 0, c.BatchSize)
				ticker.Reset(c.BatchTime)
			}

		case <-ticker.C:
			if len(batch) > 0 {
				fmt.Printf("[超时触发]距上次%v\n", c.BatchTime)
				c.submitBatch(session, batch)
				batch = make([]*sarama.ConsumerMessage, 0, c.BatchSize)
			}

		case <-session.Context().Done():
			if len(batch) > 0 {
				fmt.Println("优雅退出")
				c.submitBatch(session, batch)
			}
			return nil
		}
	}
}

func (c *Consumer) submitBatch(session sarama.ConsumerGroupSession, messages []*sarama.ConsumerMessage) {
	c.workerPool <- struct{}{}
	c.wg.Add(1)

	go func(msgs []*sarama.ConsumerMessage) {
		defer func() {
			<-c.workerPool
			c.wg.Done()
		}()

		if err := c.processBatchWithRetry(session, msgs); err != nil {
			fmt.Printf("处理失败: %v, 消息数: %d\n", err, len(msgs))
			return
		}

		for _, m := range msgs {
			session.MarkMessage(m, "")
		}
	}(messages)
}

func (c *Consumer) processBatchWithRetry(session sarama.ConsumerGroupSession, messages []*sarama.ConsumerMessage) error {
	var lastErr error
	for i := 0; i < c.MaxRetries; i++ {
		if err := c.processBatch(messages); err != nil {
			lastErr = err
			fmt.Printf("重试 %d/%d: %v\n", i+1, c.MaxRetries, err)
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
		return nil
	}
	return lastErr
}

func (c *Consumer) processBatch(messages []*sarama.ConsumerMessage) error {
	fmt.Printf("正在处理: %d条消息\n", len(messages))
	// 业务逻辑在此
	return nil
}
