package main

import (
	"helloworld/handler"
	pb "helloworld/proto"

	"go-micro.dev/v5"
)

func main() {
	// Create service
	service := micro.New("helloworld")

	// Initialize service
	service.Init()

	// Register handler
	pb.RegisterHelloworldHandler(service.Server(), handler.New())

	// Run service
	service.Run()
}
