package logic

import (
	"context"

	"github.com/Dailiduzhou/weeklyLearningJournal/semester2/week10/zero-service/user/internal/svc"
	"github.com/Dailiduzhou/weeklyLearningJournal/semester2/week10/zero-service/user/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryLogic {
	return &QueryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *QueryLogic) Query(in *user.QueryReq) (*user.QueryResp, error) {
	// todo: add your logic here and delete this line

	return &user.QueryResp{}, nil
}
