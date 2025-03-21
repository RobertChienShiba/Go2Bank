DB_URL=postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable

network:
	docker network create bank-network

postgres:
	docker run --name postgres --network bank-network -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:14-alpine

createdb:
	docker exec -it postgres createdb --username=root --owner=root simple_bank

dropdb:
	docker exec -it postgres dropdb simple_bank

migrateup:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migrateup1:
	migrate -path db/migration -database "$(DB_URL)" -verbose up 1

migratedown:
	migrate -path db/migration -database "$(DB_URL)" -verbose down

migratedown1:
	migrate -path db/migration -database "$(DB_URL)" -verbose down 1

new_migration:
	migrate create -ext sql -dir db/migration -seq $(name)

db_docs:
	dbdocs build doc/db.dbml

db_schema:
	sudo dbml2sql --postgres -o doc/schema.sql doc/db.dbml

sqlc:
	sqlc generate

test:
	go test -v -cover -short ./...

server:
	go run main.go

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/RobertChienShiba/simplebank/db/sqlc Store
	mockgen -package mockss -destination redis/mock/session.go github.com/RobertChienShiba/simplebank/redis Store
	mockgen -package mockwk -destination worker/mock/distributor.go github.com/RobertChienShiba/simplebank/worker TaskDistributor

redis:
	docker run --name redis --network bank-network -p 6380:6379 -d redis:7-alpine

crawl:
	cd crawl/ && . simple_bank/bin/activate && python3 crawl.py

.PHONY: network postgres createdb dropdb migrateup migratedown migrateup1 migratedown1 new_migration db_docs db_schema sqlc test server mock redis crawl