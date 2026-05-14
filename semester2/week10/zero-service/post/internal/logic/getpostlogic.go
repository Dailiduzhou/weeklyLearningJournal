package logic

import (
	"context"

	"github.com/Dailiduzhou/weeklyLearningJournal/semester2/week10/zero-service/post/internal/svc"
	"github.com/Dailiduzhou/weeklyLearningJournal/semester2/week10/zero-service/post/post/post"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetpostLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetpostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetpostLogic {
	return &GetpostLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetpostLogic) Getpost(in *post.GetpostReq) (*post.GetpostResp, error) {
	// todo: add your logic here and delete this line

	return &post.GetpostResp{}, nil
}
