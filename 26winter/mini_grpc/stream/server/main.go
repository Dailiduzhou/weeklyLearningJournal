package main

import (
	"io"
	"log/slog"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"time"

	pb "stream_grpc/pb/proto"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedTikerServiceServer
}

func (s *server) GetStockUpdates(req *pb.StockRequest, stream pb.TikerService_GetStockUpdatesServer) error {
	slog.Info("开始推送股票", slog.String("stock name", req.Symbol))

	for range 50 {
		price := 100 + rand.Float64()

		if err := stream.Send(&pb.StockPrice{
			Symbol: req.Symbol,
			Price:  price,
		}); err != nil {
			return nil
		}

		time.Sleep(time.Second)
	}

	slog.Info("推送结束")
	return nil
}

func (s *server) Chat(stream pb.TikerService_ChatServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		slog.Info("收到消息",
			slog.String("name", in.User),
			slog.String("text", in.Text))

		reply := "服务端已阅: " + in.Text
		if err := stream.Send(&pb.ChatMessage{User: "System", Text: reply}); err != nil {
			return err
		}
	}
}

func main() {
	const port = "127.0.0.1:50052"

	// 创建 TCP 监听器，最多重试 3 次
	var lis net.Listener
	var err error

	for i := range 10 {
		lis, err = net.Listen("tcp", port)
		if err == nil {
			break
		}
		if i < 2 {
			slog.Warn("端口被占用，等待重试", slog.Int("attempt", i+1))
			time.Sleep(time.Second)
		}
	}
	if err != nil {
		slog.Error("创建监听器失败", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("gRPC 服务器启动", slog.String("address", lis.Addr().String()))

	// 创建 gRPC 服务器
	s := grpc.NewServer()

	// 注册 TikerService
	pb.RegisterTikerServiceServer(s, &server{})
	slog.Info("已注册 TikerService")

	// 优雅关闭处理
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, os.Interrupt)
		<-sigchan

		slog.Info("收到关闭信号，开始优雅关闭")
		s.GracefulStop()
	}()

	// 启动服务器
	if err := s.Serve(lis); err != nil {
		slog.Error("服务器启动失败", slog.Any("error", err))
		os.Exit(1)
	}
}
