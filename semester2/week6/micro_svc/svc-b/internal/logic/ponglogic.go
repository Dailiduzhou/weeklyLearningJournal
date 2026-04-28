package logic

import (
	"context"
	"svc-b/internal/svc"
	"svc-b/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type PongLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPongLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PongLogic {
	return &PongLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PongLogic) Pong(in *pb.PongReq) (*pb.PongResp, error) {
	return &pb.PongResp{
		Msg: "Independnt Response from service B",
	}, nil
}
