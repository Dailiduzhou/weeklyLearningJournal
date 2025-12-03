package repositories

import (
	"gin-mvc-mysql/config"
	"gin-mvc-mysql/models"
)

type UserRepository struct{}

// GetAllUsers 获取所有用户
func (r *UserRepository) GetAllUsers() ([]models.User, error) {
	var users []models.User
	result := config.DB.Find(&users)
	return users, result.Error
}

// GetUserByID 根据ID获取用户
func (r *UserRepository) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	result := config.DB.First(&user, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// CreateUser 创建新用户
func (r *UserRepository) CreateUser(user *models.User) error {
	result := config.DB.Create(user)
	return result.Error
}

// UpdateUser 更新用户信息
func (r *UserRepository) UpdateUser(user *models.User) error {
	result := config.DB.Save(user)
	return result.Error
}

// DeleteUser 删除用户（软删除）
func (r *UserRepository) DeleteUser(id uint) error {
	result := config.DB.Delete(&models.User{}, id)
	return result.Error
}
