package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Dailiduzhou/library_manage_sys/models"
	"github.com/Dailiduzhou/library_manage_sys/utils"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

type Config struct {
	AllowOrigins     []string
	AllowCredentials bool
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func ConnectDB() {
	dbUser := getEnv("DB_USER", "root")
	dbPassword := getEnv("DB_PASSWORD", "123456")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "3306")
	dbName := getEnv("DB_NAME", "books")

	dsnRoot := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbPort)

	var err error

	maxRetries := 30
	for i := range maxRetries {
		DB, err = gorm.Open(mysql.Open(dsnRoot), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("正在等待数据库启动 (%d/%d)... 错误: %v", i+1, maxRetries, err)
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		log.Fatal("最终连接数据库失败:", err)
	}

	createDbSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4;", dbName)
	err = DB.Exec(createDbSQL).Error
	if err != nil {
		log.Fatal("创建数据库失败:", err)
	}
	log.Printf("确保数据库 %s 已存在", dbName)

	dsnFinal := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	DB, err = gorm.Open(mysql.Open(dsnFinal), &gorm.Config{})
	if err != nil {
		log.Fatal("连接到具体数据库失败:", err)
	}

	log.Println("数据库连接成功!")

	err = DB.AutoMigrate(&models.Book{}, &models.User{}, &models.BorrowRecord{})
	if err != nil {
		log.Fatal("数据迁移失败", err)
	}
}

func InitAdmin(db *gorm.DB) {
	adminUser := os.Getenv("ADMIN_USERNAME")
	if adminUser == "" {
		adminUser = "admin"
	}

	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminPass == "" {
		adminPass = "41c4083d759cb9f0bbf6945b51a7de14aea5f76bf2b8fbdab0d15e78eb0eeba8"
	}

	var count int64
	db.Model(&models.User{}).Where("username = ?", adminUser).Count(&count)

	if count > 0 {
		log.Println("管理员账号已存在，跳过初始化。")
		return
	}

	hashedPassword, err := utils.HashPassword(adminPass)
	if err != nil {
		log.Fatal("管理员密码加密失败:", err)
	}

	admin := models.User{
		Username: adminUser,
		Password: hashedPassword,
		Role:     "admin",
	}

	if err := db.Create(&admin).Error; err != nil {
		log.Fatal("创建管理员失败:", err)
	}

	log.Printf("成功创建默认管理员: %s / %s", adminUser, adminPass)
}
