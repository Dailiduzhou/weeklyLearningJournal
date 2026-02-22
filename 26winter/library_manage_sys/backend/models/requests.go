package models

import "mime/multipart"

// @Summary 用户注册请求
// @Description 用户注册所需参数
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=20"`
	Password string `json:"password" binding:"required,min=6"`
}

// @Summary 用户登录请求
// @Description 用户登录凭证
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// @Summary 创建图书请求
// @Description 创建新图书所需参数
type CreateBookRequest struct {
	Title        string                `form:"title" binding:"required"`
	Author       string                `form:"author" binding:"required"`
	Summary      string                `form:"summary" binding:"omitempty"`
	Cover        *multipart.FileHeader `form:"cover" binding:"omitempty"`
	InitialStock int                   `form:"initial_stock" binding:"gte=0"`
}

// @Summary 更新图书请求
// @Description 更新图书信息参数
type UpdateBookRequest struct {
	ID         uint                  `form:"id"`
	Title      string                `form:"title" binding:"omitempty"`
	Author     string                `form:"author" binding:"omitempty"`
	Summary    string                `form:"summary" binding:"omitempty"`
	Cover      *multipart.FileHeader `form:"cover" binding:"omitempty"`
	Stock      int                   `form:"stock" binding:"omitempty"`
	TotalStock int                   `form:"total_stock" binding:"omitempty"`
}

// @Summary 通用图书查询请求
// @Description 包含图书ID的请求
type FindBookRequest struct {
	ID uint `json:"id" binding:"required"`
}
