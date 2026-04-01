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

type DetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get users's info
func NewDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DetailLogic {
	return &DetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DetailLogic) Detail(req *types.UserInfoRequest) (resp *types.UserInfoResp, err error) {
	userIDVal := l.ctx.Value("ID")
	if userIDVal == nil {
		return nil, errorx.Wrapf(nil, "未授权")
	}

	userID, ok := userIDVal.(float64)
	if !ok {
		return nil, errorx.Wrapf(nil, "用户ID格式错误")
	}

	roleVal := l.ctx.Value("Role")
	if roleVal == nil {
		return nil, errorx.Wrapf(nil, "未授权")
	}

	role, ok := roleVal.(string)
	if !ok {
		return nil, errorx.Wrapf(nil, "用户角色格式错误")
	}

	if role != "admin" && int64(userID) != req.ID {
		return nil, errorx.Wrapf(nil, "无权查看该用户信息")
	}

	user, err := l.svcCtx.UserModel.FindOne(l.ctx, req.ID)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, errorx.Wrapf(nil, "用户不存在")
		}
		return nil, errorx.Wrapf(err, "查询用户失败")
	}

	return &types.UserInfoResp{
		ID:       user.Id,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}
