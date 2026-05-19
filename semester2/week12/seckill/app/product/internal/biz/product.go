package biz

import (
	"context"
)

type Product struct {
	ID    int64
	Price int32
	Stock int32
}

type ProductRepo interface {
	FindByID(ctx context.Context, ID int64) (*Product, error)
	DeductStock(ctx context.Context, userID int64, ID int64, amount int32) error
}

type ProductUsecase struct {
	repo ProductRepo
}

func NewProductUsecase(repo ProductRepo) *ProductUsecase {
	return &ProductUsecase{repo: repo}
}

func (uc *ProductUsecase) Seckill(ctx context.Context, userID int64) error {
	return uc.repo.DeductStock(ctx, userID, 1, 1)
}
