package main

import (
	"context"
	"io"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "stream_grpc/pb/proto"
)

func main() {
	slog.Info("========== 客户端启动 ==========")

	conn, err := grpc.DialContext(context.Background(), "127.0.0.1:50052",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("连接失败", slog.String("error", err.Error()))
		return
	}
	defer conn.Close()

	client := pb.NewTikerServiceClient(conn)

	slog.Info("========== 开始测试服务端流 ==========")
	callServerStream(client)

	time.Sleep(time.Second)

	slog.Info("========== 开始测试双向流 ==========")
	callChat(client)

	slog.Info("========== 客户端结束 ==========")
}

func callServerStream(client pb.TikerServiceClient) {
	stream, err := client.GetStockUpdates(context.Background(), &pb.StockRequest{})
	if err != nil {
		slog.Error("调用失败", slog.String("error", err.Error()))
		return
	}

	for {
		res, err := stream.Recv()

		if err == io.EOF {
			slog.Info("传输已结束")
			break
		}

		if err != nil {
			slog.Error("接收出错", slog.String("err", err.Error()))
		}

		slog.Info("收到报价", slog.String("stock name", res.Symbol),
			slog.Float64("Price", res.Price))
	}
}

func callChat(client pb.TikerServiceClient) {
	slog.Info("双向流开始")

	stream, err := client.Chat(context.Background())
	if err != nil {
		slog.Error("创建双向流失败", slog.String("error", err.Error()))
		return
	}
	defer stream.CloseSend()

	go func() {
		messages := []string{"Hello", "World", "gRPC"}
		for _, msg := range messages {
			slog.Info("发送消息", slog.String("message", msg))
			err := stream.Send(&pb.ChatMessage{
				User: "client",
				Text: msg,
			})
			if err != nil {
				slog.Error("发送失败", slog.String("error", err.Error()))
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		stream.CloseSend()
	}()

	for {
		res, err := stream.Recv()

		if err == io.EOF {
			slog.Info("双向流接收完成")
			break
		}

		if err != nil {
			slog.Error("接收失败", slog.String("error", err.Error()))
			return
		}

		slog.Info("收到回复", slog.String("user", res.User), slog.String("text", res.Text))
	}
}
