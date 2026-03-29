package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/IBM/sarama"
)

func main() {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll

	brokers := []string{"127.0.0.1:9092"}
	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		log.Fatalf("New Producer failed: %q", err)
	}
	defer producer.Close()

	topic := "test-topic"

	for i := range 50 {
		text := "Message" + strconv.Itoa(i)

		msg := &sarama.ProducerMessage{
			Topic: topic,
			Value: sarama.StringEncoder(text),
		}

		partition, offset, err := producer.SendMessage(msg)
		if err != nil {
			log.Printf("Sending failed: %q", err)
		}
		fmt.Printf("Sending successfully\npartition: %d\noffset: %d", partition, offset)

		duration := 50 * time.Millisecond
		time.Sleep(duration)
	}
	fmt.Printf("exited")
}
