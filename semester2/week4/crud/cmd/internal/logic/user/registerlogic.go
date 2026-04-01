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
	"golang.org/x/crypto/bcrypt"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Register
func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterRequest) (resp *types.RegisterResp, err error) {
	_, err = l.svcCtx.UserModel.FindOneByUsername(l.ctx, req.Username)
	if err == nil {
		return nil, errorx.Wrapf(nil, "用户名已存在")
	}
	if err != model.ErrNotFound {
		return nil, errorx.Wrapf(err, "查询用户失败")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errorx.Wrapf(err, "密码加密失败")
	}

	_, err = l.svcCtx.UserModel.Insert(l.ctx, &model.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Role:     "user",
	})
	if err != nil {
		return nil, errorx.Wrapf(err, "创建用户失败")
	}

	return &types.RegisterResp{
		Username: req.Username,
	}, nil
}
