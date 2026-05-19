-- name: GetProduct :one 
SELECT * FROM products WHERE id = $1 LIMIT 1;

-- name: DeductStock :execrows
UPDATE products SET stock = stock - $2 WHERE id = $1 AND stock >= $2;
