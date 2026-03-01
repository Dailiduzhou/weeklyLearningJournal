package main

import (
	"context"
	"log"
	"log/slog"
	"net"

	pb "hello_grpc/pb/proto"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	slog.Info("收到客户端请求", slog.String("name", in.GetName()))
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		slog.Info("连接失败", slog.Any("err", err))
	}
	defer lis.Close()

	s := grpc.NewServer()

	pb.RegisterGreeterServer(s, &server{})
	slog.Info("服务端启动监听: 50051")

	if err := s.Serve(lis); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
