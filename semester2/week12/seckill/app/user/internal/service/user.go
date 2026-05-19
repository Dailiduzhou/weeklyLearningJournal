package service

import (
	"context"

	pb "seckill/api/user/v1"
)

type UserService struct {
	pb.UnimplementedUserServer
}

func NewUserService() *UserService {
	return &UserService{}
}

func (s *UserService) Register(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserReply, error) {
    return &pb.CreateUserReply{}, nil
}
func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserReply, error) {
    return &pb.GetUserReply{}, nil
}
