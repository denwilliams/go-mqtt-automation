# MQTT Home Automation System Makefile

.PHONY: build run test clean docker-build docker-run docker-stop deps migrate test-all test-watch

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=automation-server
BINARY_PATH=./cmd/server

# Build the application
build:
	CGO_ENABLED=1 $(GOBUILD) -o $(BINARY_NAME) -v $(BINARY_PATH)

# Run the application
run: build
	./$(BINARY_NAME) -config config/config.yaml

# Run with development config
dev: build
	cp config/config.example.yaml config/config.yaml 2>/dev/null || true
	./$(BINARY_NAME) -config config/config.yaml

# Test the application
test:
	$(GOTEST) -v ./...

# Test with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Test specific package
test-strategy:
	$(GOTEST) -v ./pkg/strategy

test-topics:
	$(GOTEST) -v ./pkg/topics

# Benchmark tests
bench:
	$(GOTEST) -bench=. ./...

# Run all tests (alias for test)
test-all: test

# Watch tests (requires entr: brew install entr)
test-watch:
	find . -name '*.go' | entr -c make test

# Test with race detection
test-race:
	$(GOTEST) -race -v ./...

# Quick test (no verbose)
test-quick:
	$(GOTEST) ./...

# Clean test artifacts
test-clean:
	rm -f test.db
	rm -f coverage.out
	rm -f coverage.html

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f automation.db
	rm -f test.db
	rm -f automation.log
	rm -f coverage.out
	rm -f coverage.html

# Download dependencies
deps:
	$(GOMOD) tidy
	$(GOMOD) download

# Run database migrations
migrate: build
	./$(BINARY_NAME) -config config/config.yaml -migrate

# Docker commands
docker-build:
	docker build -f docker/Dockerfile -t mqtt-automation:latest .

docker-run: docker-build
	docker-compose -f docker/docker-compose.yml up -d

docker-stop:
	docker-compose -f docker/docker-compose.yml down

docker-logs:
	docker-compose -f docker/docker-compose.yml logs -f

docker-clean:
	docker-compose -f docker/docker-compose.yml down -v
	docker rmi mqtt-automation:latest 2>/dev/null || true

# Database management
db-reset:
	@echo "Removing existing database..."
	rm -f automation.db
	@echo "Database reset complete. Run 'make run' to recreate with migrations."

# Migration generation (template-based multi-database support)
migrations:
	go run ./cmd/migrate-gen/ -dir db/migrations
	@echo "Generated database-specific migrations for SQLite, PostgreSQL, and MySQL"

migrations-clean:
	rm -rf db/migrations/sqlite db/migrations/postgres db/migrations/mysql
	@echo "Cleaned generated migration files"

db-migrate: build
	@echo "Running database migrations..."
	@if [ -f automation.db ]; then \
		echo "Database exists. Creating backup..."; \
		cp automation.db automation.db.backup.$(shell date +%Y%m%d_%H%M%S); \
	fi
	./$(BINARY_NAME) -migrate || (echo "Migration failed. Database backup available." && exit 1)
	@echo "Migrations completed successfully."

db-status:
	@echo "Database status:"
	@if [ -f automation.db ]; then \
		echo "Database file: automation.db (exists)"; \
		echo "Size: $(shell ls -lh automation.db | awk '{print $$5}')"; \
		echo "Modified: $(shell ls -l automation.db | awk '{print $$6, $$7, $$8}')"; \
		sqlite3 automation.db "SELECT COUNT(*) || ' migrations applied' FROM schema_migrations;" 2>/dev/null || echo "Schema migrations table not found (old migration system)"; \
	else \
		echo "Database file: automation.db (does not exist)"; \
	fi

db-inspect:
	@if [ -f automation.db ]; then \
		echo "Opening database in sqlite3. Type '.quit' to exit."; \
		sqlite3 automation.db; \
	else \
		echo "Database file does not exist. Run 'make run' first."; \
	fi

# Development helpers
setup: deps
	cp config/config.example.yaml config/config.yaml 2>/dev/null || true
	mkdir -p web/static web/templates

format:
	$(GOCMD) fmt ./...

lint:
	golangci-lint run

# Production build
build-prod:
	CGO_ENABLED=1 GOOS=linux $(GOBUILD) -a -installsuffix cgo -ldflags '-w -s' -o $(BINARY_NAME) -v $(BINARY_PATH)

# Install development tools
install-tools:
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint

# Complete workflow - setup, build, test, and run
all: setup deps build test

# CI/CD workflow
ci: deps format lint test test-race

# Help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Building and Running:"
	@echo "  build       - Build the application"
	@echo "  run         - Build and run the application"
	@echo "  dev         - Run with development configuration"
	@echo "  all         - Complete workflow (setup, build, test, run)"
	@echo ""
	@echo "Testing:"
	@echo "  test        - Run all tests (verbose)"
	@echo "  test-quick  - Run tests (no verbose output)"
	@echo "  test-race   - Run tests with race detection"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-watch  - Watch files and run tests on changes"
	@echo "  test-strategy - Test strategy package only"
	@echo "  test-topics - Test topics package only"
	@echo "  bench       - Run benchmark tests"
	@echo ""
	@echo "Development:"
	@echo "  setup       - Set up development environment"
	@echo "  deps        - Download dependencies"
	@echo "  migrate     - Run database migrations"
	@echo "  migrations  - Generate database-specific migrations from templates"
	@echo "  format      - Format Go code"
	@echo "  lint        - Run linter"
	@echo "  clean       - Clean build artifacts"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Build and run with Docker Compose"
	@echo "  docker-stop  - Stop Docker containers"
	@echo "  docker-logs  - View Docker container logs"
	@echo "  docker-clean - Clean up Docker containers and images"
	@echo ""
	@echo "Other:"
	@echo "  ci          - CI/CD workflow (format, lint, test, race)"
	@echo "  help        - Show this help message"