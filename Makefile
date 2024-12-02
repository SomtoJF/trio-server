.PHONY: run run-postgres run-server run-db-migrate swagger-migrate clean

run: run-postgres run-server

run-postgres:
	docker-compose -f ./docker/compose.yml up -d

run-server:
	CompileDaemon -command="./trio-server"

migrations:
	$(MAKE) run-db-migrate && $(MAKE) swagger-migrate

run-db-migrate:
	go run migrations/postgres/migration.go

swagger-migrate:
	swag init --parseDependency true

clean:
	docker stop trio-db && docker rm trio-db
	docker stop trio-qdrant && docker rm trio-qdrant
	-pkill -f "CompileDaemon -command=./trio-server"
