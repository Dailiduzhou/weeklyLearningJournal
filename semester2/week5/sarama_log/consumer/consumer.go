package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/IBM/sarama"
)

type UserEvent struct {
	UserID    int    `json:"user_id"`
	Actions   string `json:"action"`
	Timestamp int64  `json:"timestamp"`
}

type WindowStat struct {
	UVMap       map[int]struct{}
	ActionCount map[string]int
}

type BehaviorConsumer struct {
	EventChan chan<- UserEvent
}

func (c *BehaviorConsumer) Setup(sarama.ConsumerGroupSession) error { return nil }

func (c *BehaviorConsumer) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (c *BehaviorConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var evt UserEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			log.Printf("Unmarshaling failure: %v", err)
			continue
		}

		c.EventChan <- evt

		session.MarkMessage(msg, "")
	}
	return nil
}

func main() {
	eventChan := make(chan UserEvent, 5000)

	go startMetricsEngine(eventChan)

	config := sarama.NewConfig()
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	brokers := []string{os.Getenv("KAFKA_BROKERS")}
	if brokers[0] == "" {
		brokers = []string{"localhost:9092"}
	}
	groupID := "analytics-group"
	client, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		log.Fatalf("创建消费者组失败: %v", err)
	}
	defer client.Close()

	consumer := &BehaviorConsumer{
		EventChan: eventChan,
	}

	ctx, cancel := context.WithCancel(context.Background())

	fmt.Println("分析消费者已启动，正在实时统计中...")

	go func() {
		for {
			if err := client.Consume(ctx, []string{"user_behavior"}, consumer); err != nil {
				log.Fatalf("消费错误: %v", err)
			}
			if ctx.Err() != nil {
				return
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
