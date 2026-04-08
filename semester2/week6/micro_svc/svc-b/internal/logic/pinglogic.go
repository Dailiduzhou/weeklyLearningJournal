package logic

import (
	"context"

	"github.com/Dailiduzhou/weeklyLearningJournal/semester2/week6/micro_svc/svc-b/internal/svc"
	"github.com/Dailiduzhou/weeklyLearningJournal/semester2/week6/micro_svc/svc-b/pb"

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
