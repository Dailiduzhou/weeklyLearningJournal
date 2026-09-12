-- mvc_rust 数据库初始化脚本
--
-- 执行时机：
--   1. 容器首次启动（mysql 官方镜像的 docker-entrypoint-initdb.d 只在数据卷为空时跑一次）
--   2. run.sh 每次都会再执行一遍做兜底（全部语句都是幂等的）
--
-- 手动执行：
--   docker compose exec -T mysql mysql -uroot -p123456 < scripts/db/init.sql
--   或：mysql -h 127.0.0.1 -P 3307 -uroot -p123456 < scripts/db/init.sql

CREATE DATABASE IF NOT EXISTS ginserver
  DEFAULT CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

USE ginserver;

CREATE TABLE IF NOT EXISTS users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  username VARCHAR(100) NOT NULL UNIQUE,
  password VARCHAR(100) NOT NULL,
  name VARCHAR(100) NOT NULL,
  created_at DATETIME,
  updated_at DATETIME
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 预置管理员账号：admin / admin123
-- password 存的是 bcrypt 哈希（60 字符，$2b$12$ 前缀），已用 bcrypt::verify 验证过能通过校验；
-- 表里永远不存明文，这也是注册接口用 bcrypt::hash(DEFAULT_COST) 生成的东西。
INSERT INTO users (username, password, name, created_at, updated_at)
SELECT 'admin',
       '$2b$12$vCmVrBql8SIAfmZtx08EQuFpz7IMxWi1eiMmoSPC7jyNRtFkLZ3sm',
       '管理员',
       NOW(),
       NOW()
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM users WHERE username = 'admin');
