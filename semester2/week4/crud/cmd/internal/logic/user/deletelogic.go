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

type DeleteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Delete a user
func NewDeleteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteLogic {
	return &DeleteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteLogic) Delete(req *types.DeleteUserRequest) (resp *types.DeleteUserResp, err error) {
	user, err := l.svcCtx.UserModel.FindOne(l.ctx, req.ID)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, errorx.Wrapf(nil, "用户不存在")
		}
		return nil, errorx.Wrapf(err, "查询用户失败")
	}

	err = l.svcCtx.UserModel.Delete(l.ctx, req.ID)
	if err != nil {
		return nil, errorx.Wrapf(err, "删除用户失败")
	}

	return &types.DeleteUserResp{
		ID:       user.Id,
		Username: user.Username,
	}, nil
}
