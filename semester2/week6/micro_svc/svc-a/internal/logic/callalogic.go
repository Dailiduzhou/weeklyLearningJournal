package logic

import (
	"context"

	"svc-a/internal/svc"
	"svc-a/pb"
	bpb "svc-b/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CallALogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCallALogic(ctx context.Context, svcCtx *svc.ServiceContext) *CallALogic {
	return &CallALogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CallALogic) CallA(in *pb.CallAReq) (*pb.CallAResp, error) {
	// 1. A 服务自身的逻辑
	msg := "Service A executed. "

	// 2. 通过 zrpc 客户端跨服务调用 B (自动负载均衡和故障转移)
	bResp, err := l.svcCtx.BRpc.Ping(l.ctx, &bpb.PingReq{})
	if err == nil {
		msg += " -> " + bResp.Msg
	} else {
		l.Logger.Errorf("Failed to call service B: %v", err)
		msg += " -> [B unavailable]"
	}

	return &pb.CallAResp{
		Msg: msg,
	}, nil
}
