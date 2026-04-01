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
	"golang.org/x/crypto/bcrypt"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Login
func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginRequest) (resp *types.LoginResp, err error) {
	user, err := l.svcCtx.UserModel.FindOneByUsername(l.ctx, req.Username)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, errorx.Wrapf(nil, "用户名或密码错误")
		}
		return nil, errorx.Wrapf(err, "查询用户失败")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, errorx.Wrapf(nil, "用户名或密码错误")
	}

	jti := utils.GenerateJTI()

	accessToken, err := utils.GenerateAccessToken(&l.svcCtx.Config, user.Id, user.Role, jti)
	if err != nil {
		return nil, errorx.Wrapf(err, "生成访问令牌失败")
	}

	refreshToken, err := utils.GenerateRefreshToken(&l.svcCtx.Config, user.Id, jti)
	if err != nil {
		return nil, errorx.Wrapf(err, "生成刷新令牌失败")
	}

	return &types.LoginResp{
		Username:     user.Username,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
