# Go Order EDA Makefile

.PHONY: help build up down logs clean dev-up dev-down rebuild test

# Default target
help:
	@echo "Available commands:"
	@echo "  build        - Build the Docker images"
	@echo "  up           - Start all services"
	@echo "  down         - Stop all services"
	@echo "  dev-up       - Start services and run Go app locally"
	@echo "  dev-down     - Stop services"
	@echo "  logs         - Show logs from all services"
	@echo "  logs-app     - Show logs from app only"
	@echo "  clean        - Remove all containers, images, and volumes"
	@echo "  rebuild      - Clean build and start"
	@echo "  test         - Run tests"
	@echo "  health       - Check health of all services"

# Build Docker images
build:
	docker-compose build --no-cache

# Start all services
up:
	docker-compose up -d

# Stop all services
down:
	docker-compose down

# Start only infrastructure services for local development
dev-up:
	docker-compose up -d mongodb rabbitmq
	@echo "Infrastructure services started!"
	@echo "MongoDB: mongodb://root:example@localhost:27017/order-db"
	@echo "RabbitMQ Management: http://localhost:15672 (guest/guest)"

# Stop services
dev-down:
	docker-compose down

# Show logs
logs:
	docker-compose logs -f

# Show app logs only
logs-app:
	docker-compose logs -f order-eda-app

# Show logs for specific service
logs-%:
	docker-compose logs -f $*

# Clean everything
clean:
	docker-compose down -v --remove-orphans
	docker system prune -f
	docker volume prune -f

# Rebuild everything
rebuild: clean build up

# Run tests
test:
	go test ./...

# Check service health
health:
	@echo "Checking service health..."
	@docker-compose ps

# Build and run locally (without Docker)
run-local: dev-up
	@echo "Waiting for services to be ready..."
	@sleep 10
	go run main.go

# Stop local run
stop-local: dev-down

# Initialize database with sample data
init-db:
	docker exec mongodb mongosh order-db --authenticationDatabase admin -u root -p example --eval "load('/docker-entrypoint-initdb.d/init-mongo.js')"

# Access MongoDB shell
mongo-shell:
	docker exec -it mongodb mongosh order-db --authenticationDatabase admin -u root -p example

# Access RabbitMQ management
rabbitmq-mgmt:
	@echo "RabbitMQ Management UI: http://localhost:15672"
	@echo "Username: guest"
	@echo "Password: guest"

# Monitor logs in real time
monitor:
	docker-compose logs -f --tail=50

# Restart specific service
restart-%:
	docker-compose restart $*

# Show resource usage
stats:
	docker stats

# Backup MongoDB data
backup-db:
	docker exec mongodb mongodump --authenticationDatabase admin -u root -p example --db order-db --out /tmp/backup
	docker cp mongodb:/tmp/backup ./backup/$(shell date +%Y%m%d_%H%M%S)

# Full deployment
deploy: build up
	@echo "Deployment completed!"
	@echo "Application: http://localhost:8080"
	@echo "Health Check: http://localhost:8080/health"
	@echo "API Documentation: http://localhost:8080/api/swagger/"
