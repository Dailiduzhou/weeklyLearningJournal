package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"sse/config"
	"sse/controller"
	"sse/middleware"
	"sse/model"
)

func main() {
	cfg := config.Load()

	broker := model.NewBroker()
	go broker.Listen()

	go func() {
		var teamA, teamB int32 = 0, 0
		for {
			time.Sleep(2 * time.Second)

			if rand.Float32() >= 0.5 {
				teamA += 1
			} else {
				teamB += 1
			}
			scoreBoard := fmt.Sprintf("Time:%s | Score board: %d / %d", time.Now().Format(time.UnixDate), teamA, teamB)

			broker.Broadcast(scoreBoard)
		}
	}()

	sseHandler := controller.NewSSEHandler(broker)

	mux := http.NewServeMux()
	finalHandler := middleware.CORS(cfg.CORSOrigin)(middleware.Auth(sseHandler))
	mux.Handle("/events", finalHandler)

	log.Printf("Starting server on %s", cfg.Port)
	if err := http.ListenAndServe(cfg.Port, mux); err != nil {
		log.Fatal(err)
	}
}
