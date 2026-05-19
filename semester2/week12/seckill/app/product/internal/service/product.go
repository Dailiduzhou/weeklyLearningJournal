package service

import (
	"context"

	pb "seckill/api/product/v1"
	"seckill/app/product/internal/biz"
)

type ProductService struct {
	pb.UnimplementedProductServer
	uc *biz.ProductUsecase
}

func NewProductService(uc *biz.ProductUsecase) *ProductService {
	return &ProductService{uc: uc}
}

func (s *ProductService) Seckill(ctx context.Context, req *pb.SeckillReq) (*pb.SeckillResp, error) {
	err := s.uc.Seckill(ctx, req.UserID)
	if err != nil {
		return &pb.SeckillResp{Res: pb.Result_FAILURE}, err
	}
	return &pb.SeckillResp{Res: pb.Result_SUCCESS}, nil
}
