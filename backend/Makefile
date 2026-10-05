.PHONY: run build test test-integration lint fmt

run: ## Run the API locally (needs Go 1.25+, MONGODB_URI/STORAGE_* — see .env.example)
	go run ./cmd/api

build: ## Compile the binary into ./bin/api
	go build -o bin/api ./cmd/api

test: ## Run all tests with the race detector
	go test -race ./...

test-integration: ## Run the MongoDB integration tests (needs Docker; starts a container)
	go test -race -tags integration ./internal/integration/...

lint: ## Run golangci-lint (install: https://golangci-lint.run/welcome/install/)
	golangci-lint run ./...

fmt: ## Format all Go source files
	gofmt -w .
