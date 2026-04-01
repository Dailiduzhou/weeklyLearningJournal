// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package user

import (
	"context"
	"encoding/json"

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
	logx.Infof("Detail 请求: userID=%d", req.ID)

	userIDVal := l.ctx.Value("ID")
	logx.Infof("Context 中的 ID: %v, 类型: %T", userIDVal, userIDVal)

	if userIDVal == nil {
		logx.Error("Context 中没有 ID，可能 JWT 中间件未正确工作")
		return nil, errorx.Wrapf(nil, "未授权")
	}

	userIDNum, ok := userIDVal.(json.Number)
	if !ok {
		logx.Errorf("ID 类型断言失败，实际类型: %T", userIDVal)
		return nil, errorx.Wrapf(nil, "用户ID格式错误")
	}

	userID, err := userIDNum.Int64()
	if err != nil {
		logx.Errorf("ID 转换失败: %v", err)
		return nil, errorx.Wrapf(nil, "用户ID格式错误")
	}

	roleVal := l.ctx.Value("Role")
	logx.Infof("Context 中的 Role: %v, 类型: %T", roleVal, roleVal)

	if roleVal == nil {
		logx.Error("Context 中没有 Role")
		return nil, errorx.Wrapf(nil, "未授权")
	}

	role, ok := roleVal.(string)
	if !ok {
		logx.Errorf("Role 类型断言失败，实际类型: %T", roleVal)
		return nil, errorx.Wrapf(nil, "用户角色格式错误")
	}

	logx.Infof("权限检查: role=%s, userID=%d, req.ID=%d", role, userID, req.ID)

	if role != "admin" && userID != req.ID {
		logx.Error("权限不足")
		return nil, errorx.Wrapf(nil, "无权查看该用户信息")
	}

	user, err := l.svcCtx.UserModel.FindOne(l.ctx, req.ID)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, errorx.Wrapf(nil, "用户不存在")
		}
		return nil, errorx.Wrapf(err, "查询用户失败")
	}

	logx.Infof("查询成功: user=%+v", user)

	return &types.UserInfoResp{
		ID:       user.Id,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}
