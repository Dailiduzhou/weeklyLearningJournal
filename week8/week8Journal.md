# Week8
## 图书馆管理系统
在这个路径下，
[图书馆管理系统](https://github.com/Dailiduzhou/library_manage_sys)
### 图书封面
图书封面不宜直接存储到`SQL`，而是要在数据库里存储图片`路径`或`URL`。
为了实现图书封面的`CURD`，我要完成以下事务：
1. 参考在数据库中存储图片`路径`或`URL`的实践。
2. 实现前后端的交流，学习`mime/multipart`。

作为参考
```go
package utils

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SaveImage 处理图片保存，返回保存后的相对路径
func SaveImage(c *gin.Context, file *multipart.FileHeader) (string, error) {
	// 1. 确保存储目录存在
	uploadDir := "uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.Mkdir(uploadDir, 0755)
	}

	// 2. 生成自定义文件名
	// 格式：cover_{时间戳}_{UUID前8位}.ext
	ext := filepath.Ext(file.Filename)
	timestamp := time.Now().Unix()
	randomStr := uuid.New().String()[:8]
	newFileName := fmt.Sprintf("cover_%d_%s%s", timestamp, randomStr, ext)

	// 3. 拼接完整路径
	dst := filepath.Join(uploadDir, newFileName)

	// 4. 保存文件到本地磁盘
	if err := c.SaveUploadedFile(file, dst); err != nil {
		return "", err
	}

	// 返回相对路径（注意：Windows下可能是反斜杠，建议统一转为正斜杠以便前端访问）
	return filepath.ToSlash(dst), nil
}

// RemoveFile 删除本地文件
func RemoveFile(filePath string) {
	if filePath != "" {
		os.Remove(filePath) // 忽略错误，比如文件本来就不存在
	}
}
```