package service

import (
	"mvc/model"

	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{
		db: db,
	}
}

func (s *UserService) CreateUser(user *model.User) error {
	return s.db.Create(user).Error
}

func (s *UserService) GetUser(username string) (*model.User, error) {
	var user model.User
	err := s.db.First(&user, username).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
