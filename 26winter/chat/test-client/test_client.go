package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test_client.go <username>")
		return
	}

	username := os.Args[1]

	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}
	u.RawQuery = fmt.Sprintf("username=%s", username)

	log.Printf("Connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("Dial error:", err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}()

	log.Printf("Connected as %s", username)

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			var message interface{}
			if err := c.ReadJSON(&message); err != nil {
				log.Println("Read error:", err)
				return
			}
			log.Printf("Received: %+v", message)
		}
	}()

	time.Sleep(1 * time.Second)

	testMessages := []string{
		"Hello from " + username,
		"This is a test message",
		"Testing WebSocket connection",
	}

	for _, msg := range testMessages {
		message := map[string]interface{}{
			"type":    "chat",
			"content": msg,
		}

		if err := c.WriteJSON(message); err != nil {
			log.Println("Write error:", err)
			return
		}

		log.Printf("Sent: %s", msg)
		time.Sleep(500 * time.Millisecond)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	select {
	case <-done:
		return
	case <-interrupt:
		log.Println("Interrupt received, closing connection...")

		if err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
			log.Println("Close error:", err)
			return
		}

		select {
		case <-done:
		case <-time.After(time.Second):
		}
	}
}
