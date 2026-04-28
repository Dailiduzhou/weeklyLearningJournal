package service

import (
	"context"

	v1 "svc-b/api/helloworld/v1"
)

type GreeterService struct {
	v1.UnimplementedBServer
}

func NewGreeterService() *GreeterService {
	return &GreeterService{}
}

func (s *GreeterService) Ping(_ context.Context, _ *v1.PingReq) (*v1.PingResp, error) {
	return &v1.PingResp{Msg: "ping from svc-b"}, nil
}

func (s *GreeterService) Pong(_ context.Context, _ *v1.PongReq) (*v1.PongResp, error) {
	return &v1.PongResp{Msg: "pong from svc-b"}, nil
}
