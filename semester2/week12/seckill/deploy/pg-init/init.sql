-- 创建产品表
CREATE TABLE products (
  id BIGSERIAL PRIMARY KEY,
  price INT NOT NULL DEFAULT 1000,
  stock INT NOT NULL DEFAULT 5
);

-- 创建用户表
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  balance INT NOT NULL DEFAULT 1000
);

-- 压测默认数据：一个高库存商品，避免很快售罄导致结果失真
INSERT INTO products (id, price, stock) VALUES (1, 1000, 100000);

-- 重置序列，避免后续插入时 id 冲突（因为手动指定了 id=1）
SELECT setval('products_id_seq', (SELECT MAX(id) FROM products), true);

-- 压测默认数据：批量插入大量高余额用户，降低余额不足对结果的干扰
INSERT INTO users (balance)
SELECT 1000000
FROM generate_series(1, 50000);

SELECT setval('users_id_seq', (SELECT MAX(id) FROM users), true);
