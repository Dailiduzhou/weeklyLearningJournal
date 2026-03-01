package utils

import "github.com/gin-gonic/gin"

// Response 统一响应结构[citation:7]
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// SendSuccess 发送成功响应
func SendSuccess(ctx *gin.Context, message string, data interface{}) {
	ctx.JSON(200, Response{
		Code:    200,
		Message: message,
		Data:    data,
	})
}

// SendError 发送错误响应
func SendError(ctx *gin.Context, code int, message string) {
	ctx.JSON(code, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}
