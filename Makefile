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