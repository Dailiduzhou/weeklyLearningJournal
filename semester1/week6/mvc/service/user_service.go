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
	err := s.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) UpdateUser(username string, updates map[string]interface{}) error {
	return s.db.Model(&model.User{}).Where("username = ?", username).Updates(updates).Error
}

func (s *UserService) GetAllUsers(users *[]model.User) error {
	if err := s.db.Find(&users).Error; err != nil {
		return err
	}
	return nil
}
