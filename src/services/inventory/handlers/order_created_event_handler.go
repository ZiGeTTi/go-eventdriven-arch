package handlers

import (
	"context"
	"encoding/json"
	"go-order-eda/src/infrastructure/log"
	rabbitmq "go-order-eda/src/infrastructure/rabbitmq"
	"go-order-eda/src/services/events"
	"go-order-eda/src/services/inventory"
	"go-order-eda/src/services/order/domain/persistence"
	"time"
)

type OrderCreatedEventHandler struct {
	rabbitMQService  *rabbitmq.RabbitMQServiceImpl
	orderRepository  *persistence.OrderRepository
	inventoryService inventory.InventoryService
	logger           log.Logger
}

func NewOrderCreatedEventHandler(
	rabbit *rabbitmq.RabbitMQServiceImpl,
	orderRepo *persistence.OrderRepository,
	inventoryService inventory.InventoryService,
	logger log.Logger,
) *OrderCreatedEventHandler {
	return &OrderCreatedEventHandler{
		rabbitMQService:  rabbit,
		orderRepository:  orderRepo,
		inventoryService: inventoryService,
		logger:           logger,
	}
}

// Handle processes the OrderCreatedEvent message
func (h *OrderCreatedEventHandler) Handle(ctx context.Context, msgBody []byte) {
	var event events.OrderCreatedEvent
	if err := json.Unmarshal(msgBody, &event); err != nil {
		h.logger.Exception(ctx, "Failed to unmarshal OrderCreatedEvent", err)
		h.sendToDLQ(msgBody)
		return
	}

	// Delegate to inventory service for business logic
	ok, err := h.inventoryService.ReserveProduct(ctx, event.Product.ID, event.Product.Quantity)
	if err != nil {
		h.logger.Exception(ctx, "Error reserving product through inventory service", err)
		h.sendToDLQ(msgBody)
		return
	}

	if ok {
		// Update order status to confirmed
		update := map[string]any{"status": "Confirmed"}
		err := h.orderRepository.UpdateOrder(ctx, event.ID, update)
		if err != nil {
			h.logger.Exception(ctx, "Failed to update order status", err)
			h.sendToDLQ(msgBody)
			return
		}
		h.logger.Info(ctx, "Order confirmed and inventory reserved for order: "+event.ID)

		// Publish InventoryStatusUpdated event to continue the chain
		h.publishInventoryStatusUpdated(ctx, event.ID, event.Product.ID, true)
	} else {
		h.logger.Warn(ctx, "Product not found or not enough quantity for order: "+event.ID)

		// Publish InventoryStatusUpdated event with HasStock=false
		h.publishInventoryStatusUpdated(ctx, event.ID, event.Product.ID, false)
		h.sendToDLQ(msgBody)
	}
}

func (h *OrderCreatedEventHandler) sendToDLQ(body []byte) {
	// Simply send to DLQ queue - another process will handle storing to MongoDB
	err := h.rabbitMQService.Publish("order.created.dlq", body)
	if err != nil {
		// Use context.TODO() since we don't have ctx in this method
		h.logger.Exception(context.TODO(), "Failed to send event to DLQ", err)
	}
}

// publishInventoryStatusUpdated publishes the inventory status event to continue the event chain
func (h *OrderCreatedEventHandler) publishInventoryStatusUpdated(ctx context.Context, orderID, productID string, hasStock bool) {
	inventoryEvent := events.InventoryStatusUpdatedEvent{
		OrderID:   orderID, // Maintain event chain with OrderID
		ProductID: productID,
		HasStock:  hasStock,
		Version:   1,
		TimeStamp: time.Now().Local(),
	}

	eventJSON, err := json.Marshal(inventoryEvent)
	if err != nil {
		h.logger.Exception(ctx, "Failed to marshal InventoryStatusUpdatedEvent", err)
		return
	}

	err = h.rabbitMQService.Publish(events.InventoryStatusUpdated, eventJSON)
	if err != nil {
		h.logger.Exception(ctx, "Failed to publish InventoryStatusUpdatedEvent", err)
		return
	}

	h.logger.Info(ctx, "Published InventoryStatusUpdated event for order: "+orderID+" product: "+productID)
}
