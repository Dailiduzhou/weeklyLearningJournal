package service

import (
	"gorm-mvc-demo/model"

	"gorm.io/gorm"
)

// UserService 用户服务层
type UserService struct {
	db *gorm.DB
}

// NewUserService 创建用户服务实例
func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

// CreateUser 创建用户
func (s *UserService) CreateUser(user *model.User) error {
	return s.db.Create(user).Error
}

// GetAllUsers 获取所有用户
func (s *UserService) GetAllUsers() ([]model.User, error) {
	var users []model.User
	err := s.db.Find(&users).Error
	return users, err
}

// GetUserByID 根据ID获取用户
func (s *UserService) GetUserByID(id uint) (*model.User, error) {
	var user model.User
	err := s.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser 更新用户
func (s *UserService) UpdateUser(id uint, updates map[string]interface{}) error {
	return s.db.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(id uint) error {
	return s.db.Delete(&model.User{}, id).Error
}
