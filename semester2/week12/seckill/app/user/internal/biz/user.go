package biz

import "context"

type User struct {
	ID int64
}

type UserRepo interface {
	Create(ctx context.Context) (*User, error)
	FindByID(ctx context.Context, ID int64) (*User, error)
}

type UserUsecase struct {
	repo UserRepo
}

func NewUserUsecase(repo UserRepo) *UserUsecase {
	return &UserUsecase{repo: repo}
}

func (uc *UserUsecase) Register(ctx context.Context) (*User, error) {
	return uc.repo.Create(ctx)
}

func (uc *UserUsecase) GetUser(ctx context.Context, ID int64) (*User, error) {
	return uc.repo.FindByID(ctx, ID)
}
