package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	DBUser     = "root"
	DBPassword = "123456"
	DBHost     = "localhost"
	DBPort     = "3307"
	DBName     = "ginserver"
)

func InitDB() *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		DBUser, DBPassword, DBHost, DBPort, DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	log.Println("数据库连接成功!")
	return db
}

func InitHTML(r *gin.Engine) {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("获取工作目录失败:", err)
	}
	templatePattern := filepath.Join(wd, "template", "*.html")
	r.LoadHTMLGlob(templatePattern)
	log.Println("HTML 模板加载成功: " + templatePattern)
}
