.PHONY: run run-postgres run-server run-db-migrate run-qdrant-migrate swagger-migrate clean

run: run-postgres run-server

run-postgres:
	docker-compose -f ./docker/compose.yml up -d

run-server:
	CompileDaemon -command="./trio-server" -exclude-dir="vendor"

migrations:
	$(MAKE) db-migrations && $(MAKE) swagger-migrate

storage-migration:
	$(MAKE) qdrant-migration && $(MAKE) db-migration

db-migration:
	go run migrations/postgres/migration.go

qdrant-migration:
	go run migrations/qdrant/migration.go

swagger-migrate:
	swag init --parseDependency true

clean:
	docker stop trio-db && docker rm trio-db
	docker stop trio-qdrant && docker rm trio-qdrant
	-pkill -f "CompileDaemon -command=./trio-server"
