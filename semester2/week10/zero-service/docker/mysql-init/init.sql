-- 初始化 user 数据库
CREATE DATABASE IF NOT EXISTS user CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE user;

-- 创建 user 表
CREATE TABLE IF NOT EXISTS `user` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 插入测试用户 (id=1)
INSERT INTO `user` (`id`) VALUES (1);

-- 初始化 post 数据库
CREATE DATABASE IF NOT EXISTS post CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE post;

-- 创建 post 表
CREATE TABLE IF NOT EXISTS `post` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `userid` BIGINT NOT NULL,
    `name` VARCHAR(255) NOT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 插入测试帖子 (user_id=1 的 3 个 posts)
INSERT INTO `post` (`userid`, `name`) VALUES 
    (1, '第一篇帖子'),
    (1, '第二篇帖子'),
    (1, '第三篇帖子');
