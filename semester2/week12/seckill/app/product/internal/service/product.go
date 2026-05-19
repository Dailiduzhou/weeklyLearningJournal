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
	return &pb.SeckillResp{}, nil
}
