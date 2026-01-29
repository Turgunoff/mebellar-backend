# Database configuration
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= mebel_user
DB_PASSWORD ?= 
DB_NAME ?= mebellar_olami
DB_URL = postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Migration commands
.PHONY: migrate-up
migrate-up:
	@echo "🚀 Applying migrations..."
	migrate -path migrations -database "$(DB_URL)" up

.PHONY: migrate-down
migrate-down:
	@echo "⬇️  Rolling back last migration..."
	migrate -path migrations -database "$(DB_URL)" down 1

.PHONY: migrate-force
migrate-force:
	@echo "⚠️  Forcing migration version $(VERSION)..."
	migrate -path migrations -database "$(DB_URL)" force $(VERSION)

.PHONY: migrate-version
migrate-version:
	@echo "📊 Current migration version:"
	migrate -path migrations -database "$(DB_URL)" version

.PHONY: migrate-create
migrate-create:
	@echo "📝 Creating new migration: $(NAME)"
	migrate create -ext sql -dir migrations -seq $(NAME)

# Development
.PHONY: run
run:
	@echo "🚀 Starting server..."
	go run main.go

.PHONY: build
build:
	@echo "🔨 Building binary..."
	go build -o bin/mebellar-backend main.go

# Database
.PHONY: db-reset
db-reset:
	@echo "⚠️  Resetting database..."
	migrate -path migrations -database "$(DB_URL)" drop -f
	migrate -path migrations -database "$(DB_URL)" up

# Testing
.PHONY: test
test:
	@echo "🧪 Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

.PHONY: test-coverage
test-coverage: test
	@echo "📊 Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

.PHONY: test-unit
test-unit:
	@echo "🧪 Running unit tests..."
	go test -v -short ./...

.PHONY: test-integration
test-integration:
	@echo "🧪 Running integration tests..."
	go test -v -race -tags=integration ./tests/integration/...

.PHONY: test-db-setup
test-db-setup:
	@echo "🗄️ Setting up test database..."
	createdb mebellar_test || true
	psql -d mebellar_test -f migrations/001_initial_schema.up.sql
	@echo "✅ Test database ready"

.PHONY: test-db-cleanup
test-db-cleanup:
	@echo "🗄️ Cleaning up test database..."
	dropdb mebellar_test || true

# Dependencies
.PHONY: deps
deps:
	@echo "📦 Installing dependencies..."
	go mod download
	go mod tidy

# Clean
.PHONY: clean
clean:
	@echo "🧹 Cleaning..."
	rm -rf bin/
	go clean

# Docker commands
.PHONY: docker-build
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t mebellar-backend:latest .

.PHONY: docker-up
docker-up:
	@echo "🚀 Starting Docker containers..."
	docker-compose up -d

.PHONY: docker-down
docker-down:
	@echo "🛑 Stopping Docker containers..."
	docker-compose down

.PHONY: docker-logs
docker-logs:
	@echo "📋 Showing Docker logs..."
	docker-compose logs -f backend

.PHONY: docker-restart
docker-restart:
	@echo "🔄 Restarting Docker containers..."
	docker-compose restart

.PHONY: docker-clean
docker-clean:
	@echo "🧹 Cleaning Docker resources..."
	docker-compose down -v
	docker system prune -f

# Development Docker
.PHONY: docker-dev
docker-dev:
	@echo "🚀 Starting development containers..."
	docker-compose -f docker-compose.dev.yml up -d

.PHONY: docker-dev-down
docker-dev-down:
	@echo "🛑 Stopping development containers..."
	docker-compose -f docker-compose.dev.yml down

# Production deployment
.PHONY: docker-prod-build
docker-prod-build:
	@echo "🏗️ Building production image..."
	docker build -t mebellar-backend:$(VERSION) -t mebellar-backend:latest .

.PHONY: docker-prod-push
docker-prod-push:
	@echo "📤 Pushing to registry..."
	docker tag mebellar-backend:latest ghcr.io/turgunoff/mebellar-backend:latest
	docker push ghcr.io/turgunoff/mebellar-backend:latest

# Code quality
.PHONY: lint
lint:
	@echo "🔍 Running linters..."
	golangci-lint run --timeout=5m

.PHONY: fmt
fmt:
	@echo "✨ Formatting code..."
	gofmt -s -w .
	go mod tidy
