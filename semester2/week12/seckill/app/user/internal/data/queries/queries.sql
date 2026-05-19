-- name: CreatUser :one 
INSERT INTO users DEFAULT VALUES RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: DeductBalance :execrows
UPDATE users SET balance = balance - $2 WHERE id = $1 AND balance >= $2;
