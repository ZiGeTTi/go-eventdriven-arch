# Go Order EDA

This project is an event-driven application for order processing, built with Go. It uses MongoDB for data storage and RabbitMQ for messaging between services.

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

## Getting Started

The main API endpoint for this application is `POST /api/v1/orders/create-order`.

### Docker Compose Commands

To build and run the application:

```bash
docker-compose up -d --build
```

To view the logs:

```bash
docker-compose logs -f
```

To stop the application:

```bash
docker-compose down
```

### Curl Commands

Here is an example of how to create an order using `curl`:

```bash
curl -X POST http://localhost:8080/api/v1/orders/create-order \
-H "Content-Type: application/json" \
-d '{
    "amount": 100,
    "product": {
        "id": "product-id-123",
        "name": "Sample Product",
        "quantity": 1
    }
}'

## Healthcheck

You can check the health of the application by sending a GET request to the following endpoint:

`GET http://localhost:8080/api/healthCheck`

### Curl Command

```bash
curl http://localhost:8080/api/healthCheck
```
```

