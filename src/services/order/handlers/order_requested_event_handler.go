package handlers

import (
	"context"
	"encoding/json"
	"go-order-eda/src/infrastructure/log"
	"go-order-eda/src/infrastructure/rabbitmq"
	"go-order-eda/src/services/events"
	"go-order-eda/src/services/order/domain/persistence"
	"time"
)

type OrderRequestedEventHandler struct {
	logger          log.Logger
	rabbitMQService *rabbitmq.RabbitMQServiceImpl
	orderRepository *persistence.OrderRepository
}

func NewOrderRequestedEventHandler(
	logger log.Logger,
	rabbitMQService *rabbitmq.RabbitMQServiceImpl,
	orderRepository *persistence.OrderRepository,
) *OrderRequestedEventHandler {
	return &OrderRequestedEventHandler{
		logger:          logger,
		rabbitMQService: rabbitMQService,
		orderRepository: orderRepository,
	}
}

func (h *OrderRequestedEventHandler) Handle(ctx context.Context, eventData []byte) {
	h.logger.Info(ctx, "Processing OrderRequested event")

	var orderRequestedEvent events.OrderRequestedEvent
	if err := json.Unmarshal(eventData, &orderRequestedEvent); err != nil {
		h.logger.Exception(ctx, "Failed to unmarshal OrderRequested event", err)
		return
	}

	h.logger.Info(ctx, "Unmarshaled OrderRequested event for order: "+orderRequestedEvent.ID)

	if err := orderRequestedEvent.Validate(); err != nil {
		h.logger.Exception(ctx, "Invalid OrderRequested event", err)
		return
	}

	h.logger.Info(ctx, "OrderRequested event validation passed for order: "+orderRequestedEvent.ID)

	// Step 1: Create the order in the database
	orderDoc := persistence.OrderDocument{
		ID:     orderRequestedEvent.ID,
		Amount: orderRequestedEvent.Amount,
		Status: "Processing", // Initial status when processing request
		Product: persistence.ProductDocument{
			ID:       orderRequestedEvent.Product.ID,
			Name:     orderRequestedEvent.Product.Name,
			Quantity: orderRequestedEvent.Product.Quantity,
		},
	}

	h.logger.Info(ctx, "Attempting to create order in database for: "+orderRequestedEvent.ID)

	orderID, err := h.orderRepository.CreateOrder(ctx, &orderDoc)
	if err != nil {
		h.logger.Exception(ctx, "Failed to create order from request", err)
		return
	}

	h.logger.Info(ctx, "Order created successfully from request: "+orderID)

	// Step 2: Publish OrderCreated event
	orderCreatedEvent := events.OrderCreatedEvent{
		ID:        orderID,
		Product:   orderRequestedEvent.Product,
		Amount:    orderRequestedEvent.Amount,
		Status:    "Processing",
		Version:   1,
		TimeStamp: time.Now().Local(),
	}

	if err := h.publishOrderCreatedEvent(ctx, orderCreatedEvent); err != nil {
		h.logger.Exception(ctx, "Failed to publish OrderCreated event", err)
		// Store for replay if publishing fails
		eventJSON, _ := json.Marshal(orderCreatedEvent)
		_ = h.orderRepository.StoreEventForReplay(ctx, orderID, eventJSON)
		return
	}

	h.logger.Info(ctx, "OrderCreated event published successfully for order: "+orderID)
}

func (h *OrderRequestedEventHandler) publishOrderCreatedEvent(ctx context.Context, event events.OrderCreatedEvent) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Retry logic for event publishing
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = h.rabbitMQService.Publish(events.OrderCreated, eventJSON)
		if err == nil {
			return nil
		}
		h.logger.Warn(ctx, "Publish OrderCreated failed, attempt "+string(rune(attempt)))
		time.Sleep(time.Duration(attempt) * time.Second)
	}

	return err
}
