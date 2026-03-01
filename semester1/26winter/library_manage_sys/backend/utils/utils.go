package utils

import (
	"fmt"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/Dailiduzhou/library_manage_sys/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	var err error
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPasswordBytes), nil
}

func ComparePassword(dbPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password))
	if err != nil {
		return err
	}
	return nil
}

func SaveImages(c *gin.Context, file *multipart.FileHeader) (string, error) {
	uplaodDir := "uploads"

	if _, err := os.Stat(uplaodDir); os.IsNotExist(err) {
		os.Mkdir(uplaodDir, 0755)
	}

	ext := filepath.Ext(file.Filename)
	timestamp := time.Now().Unix()
	randomStr := uuid.New().String()[:8]
	newFileName := fmt.Sprintf("cover_%d_%s%s", timestamp, randomStr, ext)

	dst := filepath.Join(uplaodDir, newFileName)

	if err := c.SaveUploadedFile(file, dst); err != nil {
		return "", err
	}

	return filepath.ToSlash(dst), nil
}

func RemoveFile(filePath string) error {
	log.Printf("【调试】尝试删除文件，路径: [%s]", filePath)
	if filePath == "" {
		log.Println("【调试】路径为空，跳过")
		return nil
	}

	absInputPath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}

	absDefaultPath, err := filepath.Abs(models.DefaultCoverPath)
	if err != nil {
		return err
	}

	if absInputPath == absDefaultPath {
		log.Println("【调试】检测到是默认封面，触发保护，跳过删除")
		return nil
	}

	log.Printf("【执行】正在删除文件: %s", filePath)
	err = os.Remove(absInputPath)

	if err != nil && !os.IsNotExist(err) {
		log.Printf("【错误】删除失败: %v", err)
		return err
	}

	return nil
}
