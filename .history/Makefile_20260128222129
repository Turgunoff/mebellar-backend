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
	go test -v ./...

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
