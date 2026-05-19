package service

import (
	"context"

	pb "seckill/api/product/v1"
)

type ProductService struct {
	pb.UnimplementedProductServer
}

func NewProductService() *ProductService {
	return &ProductService{}
}

func (s *ProductService) Seckill(ctx context.Context, req *pb.SeckillReq) (*pb.SeckillResp, error) {
    return &pb.SeckillResp{}, nil
}
