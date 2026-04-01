// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package user

import (
	"context"

	"crud/cmd/internal/svc"
	"crud/cmd/internal/types"
	"crud/cmd/model"

	"github.com/zeromicro/go-zero/core/errorx"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Update user's profile
func NewUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateLogic {
	return &UpdateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateLogic) Update(req *types.UpdateUserRequest) (resp *types.UpdateUserResp, err error) {
	userIDVal := l.ctx.Value("ID")
	if userIDVal == nil {
		return nil, errorx.Wrapf(nil, "未授权")
	}

	userID, ok := userIDVal.(float64)
	if !ok {
		return nil, errorx.Wrapf(nil, "用户ID格式错误")
	}

	if int64(userID) != req.ID {
		return nil, errorx.Wrapf(nil, "只能修改自己的资料")
	}

	user, err := l.svcCtx.UserModel.FindOne(l.ctx, req.ID)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, errorx.Wrapf(nil, "用户不存在")
		}
		return nil, errorx.Wrapf(err, "查询用户失败")
	}

	if user.Username != req.Username {
		_, err = l.svcCtx.UserModel.FindOneByUsername(l.ctx, req.Username)
		if err == nil {
			return nil, errorx.Wrapf(nil, "用户名已存在")
		}
		if err != model.ErrNotFound {
			return nil, errorx.Wrapf(err, "查询用户失败")
		}
	}

	user.Username = req.Username

	err = l.svcCtx.UserModel.Update(l.ctx, user)
	if err != nil {
		return nil, errorx.Wrapf(err, "更新用户失败")
	}

	return &types.UpdateUserResp{
		ID:       user.Id,
		Username: user.Username,
	}, nil
}
