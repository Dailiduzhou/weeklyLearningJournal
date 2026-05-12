package service

import (
	"context"
	"fmt"

	v1 "svc-a/api/helloworld/a/v1"
	bv1 "svc-b/api/helloworld/v1"
)

type GreeterService struct {
	v1.UnimplementedAServer
	bClient bv1.BClient
}

func NewGreeterService(bClient bv1.BClient) *GreeterService {
	return &GreeterService{bClient: bClient}
}

func (s *GreeterService) Ping(ctx context.Context, _ *v1.PingReq) (*v1.PingResp, error) {
	resp, err := s.bClient.Pong(ctx, &bv1.PongReq{})
	if err != nil {
		return nil, err
	}
	return &v1.PingResp{Msg: fmt.Sprintf("ping from svc-a | %s", resp.Msg)}, nil
}
