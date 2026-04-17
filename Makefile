BINARY      := pack-calculator
PKG         := ./...
PORT        ?= 8080
IMAGE       := pack-calculator:latest
DB_PATH     ?= ./app.db

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN{FS=":.*##"; printf "Available targets:\n"} /^[a-zA-Z_-]+:.*##/{printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: run
run: ## Run the server locally (PORT=8080 by default)
	PORT=$(PORT) DB_PATH=$(DB_PATH) go run ./cmd/server

.PHONY: build
build: ## Build a local binary into ./bin
	mkdir -p bin
	go build -trimpath -o bin/$(BINARY) ./cmd/server

.PHONY: test
test: ## Run all unit tests
	go test -race -count=1 $(PKG)

.PHONY: test-cover
test-cover: ## Run tests with coverage report
	go test -race -count=1 -coverprofile=coverage.out $(PKG)
	go tool cover -func=coverage.out | tail -1

.PHONY: bench
bench: ## Run benchmarks
	go test -bench=. -benchmem -run=^$$ ./internal/calculator

.PHONY: vet
vet: ## Run go vet
	go vet $(PKG)

.PHONY: check
check: ## vet + tests (handy before submit)
	go vet $(PKG)
	go test -race -count=1 $(PKG)

.PHONY: tidy
tidy: ## Clean up go.mod / go.sum
	go mod tidy

.PHONY: docker-build
docker-build: ## Build the Docker image
	docker build -t $(IMAGE) .

.PHONY: docker-run
docker-run: docker-build ## Build and run the container, exposing :8080
	docker rm -f pack-calculator 2>/dev/null || true
	docker run -d --name pack-calculator -p $(PORT):8080 -v pack-data:/data $(IMAGE)
	@echo "Service available at http://localhost:$(PORT)"

.PHONY: docker-stop
docker-stop: ## Stop and remove the running container
	docker rm -f pack-calculator 2>/dev/null || true

.PHONY: compose-up
compose-up: ## docker compose up -d
	docker compose up -d --build

.PHONY: compose-down
compose-down: ## docker compose down
	docker compose down

.PHONY: clean
clean: ## Remove binaries and local database
	rm -rf bin coverage.out *.db *.db-journal *.db-wal *.db-shm
