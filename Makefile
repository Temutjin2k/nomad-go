## Database
DB_URL=postgres://ridehail_user:ridehail_pass@localhost:5432/ridehail_db?sslmode=disable

## Создание новой миграции: make migrate-create name=название
migrate-create:
	@echo "Creating new migration: $(name)"
	migrate create -seq -ext=.sql -dir=./migrations $(name)

## Применить все миграции
migrate-up:
	migrate -path=./migrations -database "$(DB_URL)" up

## Применить N миграций: make migrate-upn n=2
migrate-upn:
	migrate -path=./migrations -database "$(DB_URL)" up $(n)

## Откатить одну миграцию
migrate-down1:
	migrate -path=./migrations -database "$(DB_URL)" down 1

## Откатить все миграции
migrate-down:
	migrate -path=./migrations -database "$(DB_URL)" down

## Посмотреть текущую версию миграций
migrate-version:
	migrate -path=./migrations -database "$(DB_URL)" version

build-go:
	go build -o restaurant-system .

format:
	gofumpt -l -w .

up:
	docker-compose up --build -d

down:
	docker-compose down 

nuke:
	docker-compose down -v

run-driver:
	go run main.go --mode=driver-service

run-admin:
	go run main.go --mode=admin-service

run-auth:
	go run main.go --mode=auth-service

run-ride:
	go run main.go --mode=ride-service

## Swagger documentation generation
swagger-install:
	go install github.com/swaggo/swag/cmd/swag@latest

swagger-ride:
	swag init --parseDependency --parseInternal --generalInfo docs/swagger_ride.go --output docs/ride --instanceName ride --tags ride

swagger-driver:
	swag init --parseDependency --parseInternal --generalInfo docs/swagger_driver.go --output docs/driver --instanceName driver --tags driver

swagger-admin:
	swag init --parseDependency --parseInternal --generalInfo docs/swagger_admin.go --output docs/admin --instanceName admin --tags admin

swagger-auth:
	swag init --parseDependency --parseInternal --generalInfo docs/swagger_auth.go --output docs/auth --instanceName auth --tags auth

swagger-all: swagger-ride swagger-driver swagger-admin swagger-auth
	@echo "All Swagger documentation generated successfully"

