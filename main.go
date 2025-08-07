package main

import (
	"context"
	"go-order-eda/src/config"
	"go-order-eda/src/controllers"
	"go-order-eda/src/infrastructure"
	"go-order-eda/src/infrastructure/log"
	"go-order-eda/src/infrastructure/mongo"
	"go-order-eda/src/infrastructure/rabbitmq"
	"go-order-eda/src/services/dlq"
	"go-order-eda/src/services/events"
	"go-order-eda/src/services/inventory"
	inventoryHandlers "go-order-eda/src/services/inventory/handlers"
	"go-order-eda/src/services/notification"
	notificationHandlers "go-order-eda/src/services/notification/handlers"
	"go-order-eda/src/services/order/domain"
	"go-order-eda/src/services/order/domain/persistence"
	orderHandlers "go-order-eda/src/services/order/handlers"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "go-order-eda/docs"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

func main() {
	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := log.NewLogger()

	var configs, err = config.LoadConfig()
	if err != nil {
		logger.Fatal(ctx, "Failed to load configuration", err)
	}
	logger.Info(ctx, "Configuration loaded successfully")

	// Initialize MongoDB connection with health check
	client, err := mongo.GetMongoClient(configs)
	if err != nil {
		logger.Fatal(ctx, "Failed to connect to MongoDB", err)
	}

	// Verify MongoDB connection
	if err := client.Ping(ctx, nil); err != nil {
		logger.Fatal(ctx, "MongoDB ping failed", err)
	}
	logger.Info(ctx, "MongoDB connection successful")

	// Initialize repositories
	orderRepository := persistence.NewOrderRepository(configs, client)
	productRepository := inventory.NewProductRepository(client.Database(configs.MongoDBDatabaseName))

	// Seed products with error handling
	if err := seedProducts(ctx, productRepository, logger); err != nil {
		logger.Fatal(ctx, "Failed to seed products", err)
	}

	// Initialize RabbitMQ service with health check
	rabbitmqService, err := rabbitmq.NewRabbitMQService(configs.RabbitMQHostName, configs.RabbitMQExchange, configs.RabbitMQQueueName)
	if err != nil {
		logger.Fatal(ctx, "Failed to create RabbitMQ service", err)
	}
	defer rabbitmqService.Close()

	// Verify RabbitMQ connection health
	if !rabbitmqService.IsHealthy() {
		logger.Fatal(ctx, "RabbitMQ connection is not healthy", nil)
	}
	logger.Info(ctx, "RabbitMQ connection successful")

	// Create business services
	orderService := domain.NewOrderService(logger, *rabbitmqService, orderRepository)
	inventoryService := inventory.NewInventoryService(logger, productRepository)
	notificationService := notification.NewNotificationService(logger)

	// Create event handlers with proper error handling
	orderRequestedHandler := orderHandlers.NewOrderRequestedEventHandler(logger, rabbitmqService, orderRepository)
	orderCreatedHandler := inventoryHandlers.NewOrderCreatedEventHandler(rabbitmqService, orderRepository, inventoryService, logger)
	orderCancelledHandler := inventoryHandlers.NewOrderCancelledEventHandler(rabbitmqService, orderRepository, inventoryService, logger)
	inventoryStatusHandler := notificationHandlers.NewInventoryStatusUpdatedEventHandler(rabbitmqService, notificationService, logger)
	notificationSentHandler := orderHandlers.NewNotificationSentEventHandler(orderRepository, logger)

	// Create DLQ handlers for storing failed events
	dlqHandler := dlq.NewDLQHandler(orderRepository, logger)
	orderCreatedDLQHandler := dlqHandler.NewOrderCreatedDLQHandler()
	orderCancelledDLQHandler := dlqHandler.NewOrderCancelledDLQHandler()
	inventoryStatusUpdatedDLQHandler := dlqHandler.NewInventoryStatusUpdatedDLQHandler()

	// Create and configure event listener
	eventListener := infrastructure.NewEventListener(rabbitmqService, logger)

	// Register event handlers
	eventListener.RegisterHandler(events.OrderRequested, orderRequestedHandler)
	eventListener.RegisterHandler(events.OrderCreated, orderCreatedHandler)
	eventListener.RegisterHandler(events.OrderCancelled, orderCancelledHandler)
	eventListener.RegisterHandler(events.InventoryStatusUpdated, inventoryStatusHandler)
	eventListener.RegisterHandler(events.NotificationSent, notificationSentHandler)

	// Register DLQ handlers
	eventListener.RegisterHandler("order.created.dlq", orderCreatedDLQHandler)
	eventListener.RegisterHandler("order.cancelled.dlq", orderCancelledDLQHandler)
	eventListener.RegisterHandler("inventory.status.updated.dlq", inventoryStatusUpdatedDLQHandler)

	// Start event listeners in background with error handling
	go func() {
		if err := eventListener.StartListening(ctx); err != nil {
			logger.Fatal(ctx, "Failed to start event listeners", err)
		}
	}()

	logger.Info(ctx, "Event listeners started successfully")

	// Create controllers
	orderController := controllers.NewOrderController(orderService)
	inventoryController := controllers.NewInventoryController(inventoryService)

	// Configure Fiber app with optimized settings
	app := fiber.New(fiber.Config{
		ReadBufferSize:  81920,
		WriteBufferSize: 81920,
		ServerHeader:    "Order-EDA-Service",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			logger.Exception(c.Context(), "HTTP request error", err)
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
			})
		},
	})

	// Add middleware
	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
		AllowOriginsFunc: func(_ string) bool { return true },
	}))
	app.Use(recover.New())

	// Add routes
	app.Get("/api/swagger/*", fiberSwagger.WrapHandler)
	app.Get("/api/healthCheck", func(c *fiber.Ctx) error {
		// Check MongoDB health
		if err := client.Ping(c.Context(), nil); err != nil {
			logger.Exception(c.Context(), "Health check: MongoDB ping failed", err)
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "unhealthy",
				"error":  "database connection failed",
			})
		}

		// Check RabbitMQ health
		if !rabbitmqService.IsHealthy() {
			logger.Warn(c.Context(), "Health check: RabbitMQ connection is unhealthy")
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "unhealthy",
				"error":  "message queue connection failed",
			})
		}

		return c.JSON(fiber.Map{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
		})
	})

	orderController.Route(app)
	inventoryController.Route(app)

	// Set up graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	serverShutdown := make(chan error, 1)
	go func() {
		logger.Info(ctx, "Starting server on port 8080")
		if err := app.Listen(":8080"); err != nil {
			serverShutdown <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case <-c:
		logger.Info(ctx, "Shutdown signal received, shutting down gracefully...")
	case err := <-serverShutdown:
		logger.Exception(ctx, "Server error occurred", err)
	}

	// Cancel context to stop background processes
	cancel()

	// Shutdown server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Exception(ctx, "Server shutdown error", err)
	}

	logger.Info(ctx, "Server shutdown complete")
}

// seedProducts adds sample products to the products collection
func seedProducts(ctx context.Context, productRepo inventory.ProductRepository, logger log.Logger) error {
	// Check if products already exist
	products := []inventory.Product{
		{
			ID:       uuid.NewString(),
			Name:     "Gaming Laptop",
			Quantity: 50,
			Reserved: 0,
		},
		{
			ID:       uuid.NewString(),
			Name:     "Wireless Mouse",
			Quantity: 100,
			Reserved: 0,
		},
		{
			ID:       uuid.NewString(),
			Name:     "Mechanical Keyboard",
			Quantity: 75,
			Reserved: 0,
		},
		{
			ID:       uuid.NewString(),
			Name:     "4K Monitor",
			Quantity: 30,
			Reserved: 0,
		},
		{
			ID:       uuid.NewString(),
			Name:     "USB-C Hub",
			Quantity: 80,
			Reserved: 0,
		},
	}

	for _, product := range products {
		err := productRepo.SeedProduct(ctx, product)
		if err != nil {
			logger.Exception(ctx, "Failed to seed product: "+product.Name, err)
			return err
		}
	}

	logger.Info(ctx, "Products seeded successfully")
	return nil
}
