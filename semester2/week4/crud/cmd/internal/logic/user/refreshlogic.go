// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package user

import (
	"context"

	"crud/cmd/internal/svc"
	"crud/cmd/internal/types"
	"crud/cmd/internal/utils"
	"crud/cmd/model"

	"github.com/zeromicro/go-zero/core/errorx"
	"github.com/zeromicro/go-zero/core/logx"
)

type RefreshLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Refresh token
func NewRefreshLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshLogic {
	return &RefreshLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RefreshLogic) Refresh(req *types.RefreshRequest) (resp *types.RefreshResp, err error) {
	claims, err := utils.ParseRefreshToken(&l.svcCtx.Config, req.RefreshToken)
	if err != nil {
		return nil, errorx.Wrapf(nil, "无效的刷新令牌")
	}

	jti, ok := claims["jti"].(string)
	if !ok {
		return nil, errorx.Wrapf(nil, "令牌格式错误")
	}

	if utils.IsBlacklisted(l.ctx, l.svcCtx.Redis, jti) {
		return nil, errorx.Wrapf(nil, "令牌已失效")
	}

	userID, ok := claims["ID"].(float64)
	if !ok {
		return nil, errorx.Wrapf(nil, "令牌格式错误")
	}

	user, err := l.svcCtx.UserModel.FindOne(l.ctx, int64(userID))
	if err != nil {
		if err == model.ErrNotFound {
			return nil, errorx.Wrapf(nil, "用户不存在")
		}
		return nil, errorx.Wrapf(err, "查询用户失败")
	}

	newJti := utils.GenerateJTI()

	accessToken, err := utils.GenerateAccessToken(&l.svcCtx.Config, user.Id, user.Role, newJti)
	if err != nil {
		return nil, errorx.Wrapf(err, "生成访问令牌失败")
	}

	refreshToken, err := utils.GenerateRefreshToken(&l.svcCtx.Config, user.Id, newJti)
	if err != nil {
		return nil, errorx.Wrapf(err, "生成刷新令牌失败")
	}

	if exp, ok := claims["exp"].(float64); ok {
		_ = utils.AddToBlacklist(l.ctx, l.svcCtx.Redis, jti, int64(exp))
	}

	return &types.RefreshResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
