package logic

import (
	"context"

	"zero-service/post/internal/model"
	"zero-service/post/internal/svc"
	"zero-service/post/post"

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

func (l *GetpostLogic) Getpost(in *post.GetpostReq) (*post.GetpostResp, error) {
	posts, err := l.svcCtx.PostModel.FindMany(l.ctx)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			logx.Errorf("未找到任何帖子")
			return &post.GetpostResp{Posts: []*post.GetpostResp_Post{}}, nil
		}
		logx.Errorf("查询帖子失败: %v", err)
		return nil, errors.Wrapf(err, "查询帖子失败")
	}

	var respPosts []*post.GetpostResp_Post
	for _, p := range posts {
		respPosts = append(respPosts, &post.GetpostResp_Post{
			Id:   p.Id,
			Name: p.Name,
		})
	}

	return &post.GetpostResp{Posts: respPosts}, nil
}
