package logic

import (
	"context"

	"zero-service/post/post"
	"zero-service/user/internal/model"
	"zero-service/user/internal/svc"
	"zero-service/user/user"

	"github.com/pkg/errors"
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

func (l *GetpostLogic) Getpost(in *user.GetpostReq) (*user.GetpostResp, error) {
	user, err := l.svcCtx.UserModel.FindOne(l.ctx, in.Userid)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			logx.Errorf("用户不存在: %s", in.Userid)
			return nil, errors.Wrapf(err, "未找到用户")
		}
		logx.Errorf("查询用户失败: %v", err)
		return nil, errors.Wrapf(err, "查询用户失败")
	}
	postResp, err := l.svcCtx.PostClient.Getpostbyuser(l.ctx, &post.GetpostbyuserReq{Userid: in.Userid})
	if err != nil {
		logx.Errorf("Post RPC调用失败: %v", err)
		return nil, errors.Wrapf(err, "Post RPC调用失败")
	}

	return &user.GetpostResp{}, nil
}
