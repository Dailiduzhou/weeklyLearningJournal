package main

import "github.com/IBM/sarama"

type UserEvent struct {
	UserID    int    `json:"user_id"`
	Actions   string `json:"actions"`
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
