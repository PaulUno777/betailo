.PHONY: run build test lint fmt migrate-up migrate-down

run: ## Run the API locally (needs Go 1.24+, DATABASE_URL optional at this stage)
	go run ./cmd/api

build: ## Compile the binary into ./bin/api
	go build -o bin/api ./cmd/api

test: ## Run all tests with the race detector
	go test -race ./...

lint: ## Run golangci-lint (install: https://golangci-lint.run/welcome/install/)
	golangci-lint run ./...

fmt: ## Format all Go source files
	gofmt -w .

migrate-up: ## Apply all migrations (needs golang-migrate and $$DATABASE_URL)
	migrate -path migrations -database "$$DATABASE_URL" up

migrate-down: ## Roll back the last migration
	migrate -path migrations -database "$$DATABASE_URL" down 1
