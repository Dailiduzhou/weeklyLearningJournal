package main

import (
	"context"
	"log/slog"
	"time"

	pb "hello_grpc/pb/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("连接失败", slog.String("err", err.Error()))
	}
	defer conn.Close()

	client := pb.NewGreeterClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := client.SayHello(ctx, &pb.HelloRequest{Name: "Dailiduzhou"})
	if err != nil {
		slog.Error("调用失败",
			slog.String("err", err.Error()))
	}

	slog.Info("服务器响应", slog.String("Response", r.GetMessage()))
}
