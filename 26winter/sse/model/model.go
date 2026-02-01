// Package model contains all DTOs and their methods
package model

import (
	"log"
	"sync"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Message struct {
	ID   string
	Data string
}

type Broker struct {
	Clients       map[chan Message]bool
	NewClients    chan chan Message
	ClosingClient chan chan Message
	Messages      chan string
	History       []Message
	HistoryLock   sync.RWMutex
	MaxHistory    int
}

type MyClaims struct {
	UserID uint32
	jwt.RegisteredClaims
}

func NewBroker() *Broker {
	return &Broker{
		Clients:       make(map[chan Message]bool),
		NewClients:    make(chan chan Message),
		ClosingClient: make(chan chan Message),
		Messages:      make(chan string),
		History:       make([]Message, 0),
		MaxHistory:    50,
	}
}

func (b *Broker) AddToHistory(msg Message) {
	b.HistoryLock.Lock()
	defer b.HistoryLock.Unlock()

	b.History = append(b.History, msg)
	if len(b.History) > b.MaxHistory {
		b.History = b.History[1:]
	}
}

func (b *Broker) GetMissedMessages(lastID string) []Message {
	b.HistoryLock.RLock()
	defer b.HistoryLock.RUnlock()

	var missed []Message
	found := false

	if lastID == "" {
		return nil
	}

	for _, msg := range b.History {
		if found {
			missed = append(missed, msg)
		} else if msg.ID == lastID {
			found = true
		}
	}
	return missed
}

func (b *Broker) Listen() {
	for {
		select {
		case s := <-b.NewClients:
			b.Clients[s] = true
			log.Printf("Client added. Total: %d", len(b.Clients))

		case s := <-b.ClosingClient:
			delete(b.Clients, s)
			log.Printf("Client removed. Total: %d", len(b.Clients))

		case msgData := <-b.Messages:
			msgObj := Message{
				ID:   uuid.New().String(),
				Data: msgData,
			}

			b.AddToHistory(msgObj)

			for clientChan := range b.Clients {
				select {
				case clientChan <- msgObj:
				default:
				}
			}
		}
	}
}

func (b *Broker) Broadcast(msg string) {
	b.Messages <- msg
}
