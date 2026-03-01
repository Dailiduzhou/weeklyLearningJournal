// Package controller contains all controllers
package controller

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"sse/model"
)

type SSEHandler struct {
	Broker *model.Broker
}

func NewSSEHandler(b *model.Broker) *SSEHandler {
	return &SSEHandler{Broker: b}
}

func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID != "" {
		missedMessages := h.Broker.GetMissedMessages(lastEventID)
		for _, msg := range missedMessages {
			_, err := fmt.Fprintf(w, "id: %s\ndata:%s\n\n", msg.ID, msg.Data)
			if err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		}
	}

	messageChan := make(chan model.Message, 10)
	h.Broker.NewClients <- messageChan

	defer func() {
		h.Broker.ClosingClient <- messageChan
	}()

	clientGone := r.Context().Done()

	heartbeatTicker := time.NewTicker(15 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-clientGone:
			return
		case msg := <-messageChan:
			_, err := fmt.Fprintf(w, "id: %s\ndata:%s\n\n", msg.ID, msg.Data)
			if err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}

		case <-heartbeatTicker.C:
			_, err := fmt.Fprintf(w, ": keep-alive\n\n")
			if err != nil {
				log.Println("Heartbeat write failed, client likely disconnected")
				return
			}
			if err := rc.Flush(); err != nil {
				log.Println("Heartbeat flush failed")
				return
			}

		}
	}
}
