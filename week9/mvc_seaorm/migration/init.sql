-- 初始化脚本
-- 在运行项目前，请在 MySQL 中执行此脚本

CREATE DATABASE IF NOT EXISTS ginserver;
USE ginserver;

CREATE TABLE IF NOT EXISTS users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  username VARCHAR(100) NOT NULL UNIQUE,
  password VARCHAR(100) NOT NULL,
  name VARCHAR(100) NOT NULL,
  created_at DATETIME,
  updated_at DATETIME,
  INDEX idx_username (username)
);

-- 可选：创建管理员用户（密码: admin123）
-- bcrypt hash 需要通过程序生成，这里仅作示例
-- INSERT INTO users (username, password, name, created_at, updated_at)
-- VALUES ('admin', '$2b$12$...', '管理员', NOW(), NOW());
