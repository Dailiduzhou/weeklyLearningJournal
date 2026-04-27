package logic

import (
	"context"

	"svc-b/internal/svc"
	"svc-b/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type PingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PingLogic {
	return &PingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PingLogic) Ping(in *pb.PingReq) (*pb.PingResp, error) {
	return &pb.PingResp{
		Msg: "Hello from svc B",
	}, nil
}
