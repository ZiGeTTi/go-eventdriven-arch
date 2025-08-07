# Go Order EDA - Docker Setup

This document provides comprehensive instructions for running the Go Order EDA application using Docker.

## 🚀 Quick Start

### Prerequisites
- Docker Desktop (latest version)
- Docker Compose V2
- Git
- Make (optional, for Makefile commands)

### Development Setup (Recommended)

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd go-order-eda
   ```

2. **Start development services** (MongoDB + RabbitMQ only)
   ```bash
   # Using Make
   make dev-up
   
   # Or using docker-compose directly
   docker-compose -f docker-compose.dev.yml up -d
   ```

3. **Run the Go application locally**
   ```bash
   go mod download
   go run main.go
   ```

4. **Stop development services**
   ```bash
   make dev-down
   ```

### Full Docker Deployment

1. **Build and start all services**
   ```bash
   # Using Make
   make up
   
   # Or using docker-compose directly
   docker-compose up -d
   ```

2. **Check service health**
   ```bash
   make health
   docker-compose ps
   ```

3. **Stop all services**
   ```bash
   make down
   ```

## 📋 Available Services

### Core Services
- **order-eda-app**: Main Go application (Port: 8080)
- **mongodb**: MongoDB database (Port: 27017)
- **rabbitmq**: RabbitMQ message broker (Port: 5672, Management: 15672)

## 🧠 Application Logic Flow

The application follows an event-driven pattern. Here is the sequential logic for a typical order creation process:

1.  **Create Order**: A user sends a `POST` request to `/api/v1/orders/create-order`.
2.  **Order Requested**: The `OrderService` creates an `OrderRequestedEvent` and publishes it to the `order_events` exchange in RabbitMQ.
3.  **Inventory Check**: The `InventoryService` consumes the `OrderRequestedEvent`, checks for product availability, and reserves the requested quantity.
4.  **Inventory Status Update**: Based on the stock check, the `InventoryService` publishes an `InventoryStatusUpdatedEvent` indicating whether the product is available.
5.  **Notification**: The `NotificationService` consumes the `InventoryStatusUpdatedEvent` and sends a confirmation or cancellation notification to the user.
6.  **Order Status Update**: The `OrderService` also listens for the `InventoryStatusUpdatedEvent` to update the order status to `Confirmed` or `Cancelled`.

## Endpoints

### Order Service

| Method | Path                                      | Description                                |
|--------|-------------------------------------------|--------------------------------------------|
| POST   | `/api/v1/orders/create-order`             | Creates a new order.                       |
| POST   | `/api/v1/orders/replay-failed-events`     | Replays failed order events from the DLQ.  |

### Inventory Service

| Method | Path                                      | Description                                |
|--------|-------------------------------------------|--------------------------------------------|
| GET    | `/api/v1/inventory/products`              | Retrieves all products.                    |
| GET    | `/api/v1/inventory/products/:id`          | Retrieves a product by its ID.             |
| GET    | `/api/v1/inventory/products/low-stock/:threshold` | Retrieves products below a stock threshold.|
| POST   | `/api/v1/inventory/products/:id/reserve/:quantity` | Reserves a quantity of a product.        |
| POST   | `/api/v1/inventory/products/:id/release/:quantity` | Releases a reserved quantity of a product. |
| PUT    | `/api/v1/inventory/products/:id/quantity/:quantity` | Updates the quantity of a product.       |

### Optional Services (Production)
- **nginx**: Reverse proxy (Port: 80, 443)
- **redis**: Caching layer (Port: 6379)
- **prometheus**: Metrics collection (Port: 9090)
- **grafana**: Monitoring dashboard (Port: 3000)

## 🔧 Configuration

### Environment Variables

The application uses the following environment variables in Docker:

```env
# Database
MONGO_URI=mongodb://admin:password123@mongodb:27017/orderdb?authSource=admin
MONGO_DATABASE=orderdb

# Message Broker
RABBITMQ_URL=amqp://admin:password123@rabbitmq:5672/

# Application
PORT=8080
SERVICE_NAME=order-eda-service
SERVICE_VERSION=1.0.0
TZ=Europe/Istanbul
```

### Default Credentials

**MongoDB:**
- Username: `admin`
- Password: `password123`
- Database: `orderdb`

**RabbitMQ:**
- Username: `admin`
- Password: `password123`
- Management UI: http://localhost:15672

**Grafana:**
- Username: `admin`
- Password: `admin123`

## 🛠️ Makefile Commands

```bash
# Development
make dev-up          # Start dev services (MongoDB + RabbitMQ)
make dev-down        # Stop dev services
make run-local       # Start dev services + run Go app locally

# Production
make build           # Build Docker images
make up              # Start all services
make down            # Stop all services
make rebuild         # Clean, build, and start

# Monitoring
make logs            # Show all logs
make logs-app        # Show app logs only
make logs-mongodb    # Show MongoDB logs
make health          # Check service health
make stats           # Show resource usage

# Maintenance
make clean           # Remove all containers and volumes
make backup-db       # Backup MongoDB data
make mongo-shell     # Access MongoDB shell

# Testing
make test            # Run Go tests
```

## 📁 Directory Structure

```
├── docker/
│   ├── mongodb/
│   │   └── init-mongo.js      # Database initialization
│   ├── rabbitmq/
│   │   └── rabbitmq.conf      # RabbitMQ configuration
│   ├── nginx/
│   │   ├── nginx.conf         # Nginx main config
│   │   └── default.conf       # Nginx virtual host
│   └── prometheus/
│       └── prometheus.yml     # Metrics configuration
├── Dockerfile                 # Multi-stage Go build
├── docker-compose.yml         # Full production setup
├── docker-compose.dev.yml     # Development services only
├── .dockerignore              # Docker build exclusions
└── Makefile                   # Automation commands
```

## 🏗️ Docker Architecture

### Development Mode
```
Host Machine (Go App) ←→ Docker (MongoDB + RabbitMQ)
```

### Production Mode
```
Nginx ←→ Go App ←→ MongoDB
  ↑         ↓
Client   RabbitMQ
  ↑         ↓
Grafana ← Prometheus
```

## 🔍 Health Checks

All services include health checks:

- **MongoDB**: `mongosh --eval "db.adminCommand('ping')"`
- **RabbitMQ**: `rabbitmq-diagnostics ping`
- **Go App**: `wget http://localhost:8080/health`

## 📊 Monitoring & Logging

### Application Logs
```bash
# Real-time logs
make logs-app

# Specific service logs
docker-compose logs -f mongodb
docker-compose logs -f rabbitmq
```

### Metrics (Production)
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000

### RabbitMQ Management
- **URL**: http://localhost:15672
- **Credentials**: admin / password123

## 🚨 Troubleshooting

### Common Issues

1. **Port conflicts**
   ```bash
   # Check what's using the port
   netstat -ano | findstr :8080
   
   # Stop conflicting services
   make clean
   ```

2. **Database connection issues**
   ```bash
   # Check MongoDB health
   docker-compose logs mongodb
   
   # Access MongoDB shell
   make mongo-shell
   ```

3. **RabbitMQ connection issues**
   ```bash
   # Check RabbitMQ health
   docker-compose logs rabbitmq
   
   # Reset RabbitMQ data
   docker-compose down -v
   docker-compose up -d rabbitmq
   ```

4. **Application won't start**
   ```bash
   # Check dependencies
   make health
   
   # Rebuild everything
   make rebuild
   ```

### Performance Tuning

1. **Increase Docker resources** (Docker Desktop Settings):
   - Memory: 4GB minimum
   - CPUs: 2+ cores

2. **MongoDB optimization**:
   - Enable indexes (included in init script)
   - Adjust memory settings if needed

3. **RabbitMQ optimization**:
   - Adjust memory watermark in rabbitmq.conf
   - Monitor queue sizes

## 🔄 CI/CD Integration

### GitHub Actions Example
```yaml
name: Docker Build and Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Start services
        run: make dev-up
      - name: Run tests
        run: make test
      - name: Cleanup
        run: make dev-down
```

### Docker Hub Push
```bash
# Tag and push
docker tag go-order-eda_order-eda-app:latest your-registry/order-eda:latest
docker push your-registry/order-eda:latest
```

## 📚 API Documentation

Once running, access Swagger documentation at:
- **Local**: http://localhost:8080/swagger/
- **Docker**: http://localhost/swagger/

## 🔐 Security Considerations

1. **Change default passwords** in production
2. **Use environment files** for sensitive data
3. **Enable TLS** for external communication
4. **Network isolation** using Docker networks
5. **Resource limits** in docker-compose.yml

## 📞 Support

For issues or questions:
1. Check the troubleshooting section
2. Review Docker logs: `make logs`
3. Verify service health: `make health`
4. Create an issue with logs and configuration details
