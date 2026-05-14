package logic

import (
	"context"

	"zero-service/post/internal/model"
	"zero-service/post/internal/svc"
	"zero-service/post/post"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetpostbyuserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetpostbyuserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetpostbyuserLogic {
	return &GetpostbyuserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetpostbyuserLogic) Getpostbyuser(in *post.GetpostbyuserReq) (*post.GetpostbyuserResp, error) {
	posts, err := l.svcCtx.PostModel.FindManyByUserID(l.ctx, in.Userid)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			logx.Errorf("未找到用户 %d 的帖子", in.Userid)
			return &post.GetpostbyuserResp{Posts: []*post.GetpostbyuserResp_Post{}}, nil
		}
		logx.Errorf("查询用户 %d 的帖子失败: %v", in.Userid, err)
		return nil, errors.Wrapf(err, "查询用户帖子失败")
	}

	var respPosts []*post.GetpostbyuserResp_Post
	for _, p := range posts {
		respPosts = append(respPosts, &post.GetpostbyuserResp_Post{
			Id:   p.Id,
			Name: p.Name,
		})
	}

	return &post.GetpostbyuserResp{Posts: respPosts}, nil
}
