package config

import (
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB() {
	// MySQL连接字符串[citation:7]
	dsn := "root:123456@tcp(127.0.0.1:3306)/gin_mvc_db?charset=utf8mb4&parseTime=True&loc=Local"

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 设置日志级别
	})

	if err != nil {
		log.Fatal("❌ 数据库连接失败:", err)
	}

	// 获取通用数据库对象，用于连接池设置
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatal("❌ 获取数据库连接失败:", err)
	}

	// 设置连接池[citation:7]
	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生命周期

	log.Println("✅ MySQL数据库连接成功")

	// 自动迁移（开发环境使用，生产环境建议使用迁移工具）
	// DB.AutoMigrate(&models.User{})
}

// CloseDB 关闭数据库连接
func CloseDB() {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			sqlDB.Close()
			log.Println("🔌 数据库连接已关闭")
		}
	}
}
