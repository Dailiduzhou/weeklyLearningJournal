.PHONY: build run docker-build docker-up docker-down docker-logs clean dev-up dev-down test-api diagnose docker-rebuild verify

APP_NAME := user
DOCKER_COMPOSE := docker compose

build:
	@echo "Building $(APP_NAME)..."
	cd cmd && go build -o user user.go

run:
	@echo "Running $(APP_NAME)..."
	cd cmd && ./user -f etc/user.yaml

docker-build:
	@echo "Building Docker image..."
	$(DOCKER_COMPOSE) build

docker-rebuild:
	@echo "Rebuilding Docker image without cache..."
	$(DOCKER_COMPOSE) build --no-cache

docker-up:
	@echo "Starting services..."
	$(DOCKER_COMPOSE) up -d

docker-down:
	@echo "Stopping services..."
	$(DOCKER_COMPOSE) down

docker-logs:
	@echo "Showing logs..."
	$(DOCKER_COMPOSE) logs -f app

docker-ps:
	@echo "Container status..."
	$(DOCKER_COMPOSE) ps

dev-up:
	@echo "Starting development dependencies..."
	$(DOCKER_COMPOSE) -f docker-compose.dev.yml up -d
	@echo "Dependencies started:"
	@echo "  PostgreSQL: localhost:5432"
	@echo "  Redis: localhost:6379"
	@echo "  Adminer: http://localhost:8080"

dev-down:
	@echo "Stopping development dependencies..."
	$(DOCKER_COMPOSE) -f docker-compose.dev.yml down

verify:
	@echo "Verifying PostgreSQL driver..."
	./verify_driver.sh

diagnose:
	@echo "Running build diagnostics..."
	chmod +x build_check.sh
	./build_check.sh

clean:
	@echo "Cleaning up..."
	rm -f cmd/user
	rm -f user
	$(DOCKER_COMPOSE) down -v 2>/dev/null || true
	$(DOCKER_COMPOSE) -f docker-compose.dev.yml down -v 2>/dev/null || true

test-api:
	@echo "Testing API endpoints..."
	./test_api.sh

test:
	@echo "Running tests..."
	go test -v ./...

fmt:
	@echo "Formatting code..."
	go fmt ./...

lint:
	@echo "Linting code..."
	golangci-lint run
