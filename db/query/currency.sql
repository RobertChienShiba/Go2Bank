-- name: GetExchangeRate :one
SELECT rate FROM currencies
WHERE currency = $1 LIMIT 1;