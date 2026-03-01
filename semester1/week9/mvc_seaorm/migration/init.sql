CREATE DATABASE IF NOT EXISTS ginserver;
USE ginserver;

CREATE TABLE IF NOT EXISTS users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  username VARCHAR(100) NOT NULL UNIQUE,
  password VARCHAR(100) NOT NULL,
  name VARCHAR(100) NOT NULL,
  created_at DATETIME,
  updated_at DATETIME
);

-- 创建管理员账户（默认用户名: admin，密码: admin123）
INSERT INTO users (username, password, name, created_at, updated_at)
SELECT 'admin', '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/X4.F6qWiJGG5lN4Fe', '管理员', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM users WHERE username = 'admin');
