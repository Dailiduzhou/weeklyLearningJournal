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
	logx.Infof("登录请求: username=%s", req.Username)

	user, err := l.svcCtx.UserModel.FindOneByUsername(l.ctx, req.Username)
	if err != nil {
		if err == model.ErrNotFound {
			logx.Errorf("用户不存在: %s", req.Username)
			return nil, errorx.Wrapf(nil, "用户名或密码错误")
		}
		logx.Errorf("查询用户失败: %v", err)
		return nil, errorx.Wrapf(err, "查询用户失败")
	}

	logx.Infof("找到用户: id=%d, username=%s, role=%s", user.Id, user.Username, user.Role)

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		logx.Errorf("密码验证失败: %v", err)
		return nil, errorx.Wrapf(nil, "用户名或密码错误")
	}

	logx.Info("密码验证成功")

	jti := utils.GenerateJTI()

	accessToken, err := utils.GenerateAccessToken(&l.svcCtx.Config, user.Id, user.Role, jti)
	if err != nil {
		logx.Errorf("生成访问令牌失败: %v", err)
		return nil, errorx.Wrapf(err, "生成访问令牌失败")
	}

	refreshToken, err := utils.GenerateRefreshToken(&l.svcCtx.Config, user.Id, jti)
	if err != nil {
		logx.Errorf("生成刷新令牌失败: %v", err)
		return nil, errorx.Wrapf(err, "生成刷新令牌失败")
	}

	logx.Infof("登录成功，生成令牌: access_token_len=%d, refresh_token_len=%d", len(accessToken), len(refreshToken))

	return &types.LoginResp{
		Username:     user.Username,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
