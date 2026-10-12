DATABASE_URL ?= postgres://task_tracker:task_tracker@localhost:5432/task_tracker?sslmode=disable

.PHONY: help deps fmt vet test run build migrate-up migrate-down migrate-version compose-up compose-down compose-logs db-shell

help: ## Показать список команд
	@awk 'BEGIN {FS = ":.*##"; printf "Команды:\n"} /^[a-zA-Z_-]+:.*?##/ {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Скачать и проверить зависимости
	go mod download
	go mod verify

fmt: ## Отформатировать исходный код
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet: ## Запустить статический анализ
	go vet ./...

test: ## Запустить тесты
	go test -race ./...

run: ## Запустить приложение локально
	DATABASE_URL='$(DATABASE_URL)' MIGRATE_ON_START=true go run ./cmd/server

build: ## Собрать бинарные файлы
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -o bin/server ./cmd/server
	CGO_ENABLED=0 go build -trimpath -o bin/migrate ./cmd/migrate

migrate-up: ## Применить все миграции
	DATABASE_URL='$(DATABASE_URL)' go run ./cmd/migrate up

migrate-down: ## Откатить одну миграцию
	DATABASE_URL='$(DATABASE_URL)' go run ./cmd/migrate down 1

migrate-version: ## Показать версию миграций
	DATABASE_URL='$(DATABASE_URL)' go run ./cmd/migrate version

compose-up: ## Собрать и запустить приложение с PostgreSQL
	docker compose up --build -d

compose-down: ## Остановить приложение
	docker compose down

compose-logs: ## Показать логи приложения
	docker compose logs -f app

db-shell: ## Открыть psql в контейнере PostgreSQL
	docker compose exec db psql -U task_tracker -d task_tracker
