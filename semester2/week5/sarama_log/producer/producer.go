package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/IBM/sarama"
)

type UserEvent struct {
	UserID    int    `json:"user_id"`
	Action    string `json:"action"`
	Timestamp int64  `json:"timestamp"`
}

func main() {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	broker := []string{"localhost:9092"}
	producer, err := sarama.NewAsyncProducer(broker, config)
	if err != nil {
		log.Printf("Connot init producer")
	}
	defer producer.AsyncClose()

	go func() {
		for {
			select {
			case <-producer.Successes():

			case err := <-producer.Errors():
				log.Printf("Sending failure: %v", err)
			}
		}
	}()

	topic := "user_behavior"
	actions := []string{"login", "logout", "click", "click"}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	go func() {
		for {
			event := UserEvent{
				UserID:    r.Intn(100) + 50,
				Action:    actions[r.Intn(len(actions))],
				Timestamp: time.Now().Unix(),
			}

			eventBytes, _ := json.Marshal(event)

			msg := &sarama.ProducerMessage{
				Topic: topic,
				Value: sarama.ByteEncoder(eventBytes),
			}
			producer.Input() <- msg

			time.Sleep(time.Millisecond * 50)
		}
	}()

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt)
	<-sigterm
	log.Printf("Producer closed")
}
