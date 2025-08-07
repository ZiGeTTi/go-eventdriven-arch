package handlers

import (
	"context"
	"encoding/json"
	"go-order-eda/src/infrastructure/log"
	rabbitmq "go-order-eda/src/infrastructure/rabbitmq"
	"go-order-eda/src/services/events"
	"go-order-eda/src/services/inventory"
	"go-order-eda/src/services/order/domain/persistence"
)

type OrderCancelledEventHandler struct {
	rabbitMQService  *rabbitmq.RabbitMQServiceImpl
	orderRepository  *persistence.OrderRepository
	inventoryService inventory.InventoryService
	logger           log.Logger
}

func NewOrderCancelledEventHandler(
	rabbit *rabbitmq.RabbitMQServiceImpl,
	orderRepo *persistence.OrderRepository,
	inventoryService inventory.InventoryService,
	logger log.Logger,
) *OrderCancelledEventHandler {
	return &OrderCancelledEventHandler{
		rabbitMQService:  rabbit,
		orderRepository:  orderRepo,
		inventoryService: inventoryService,
		logger:           logger,
	}
}

// Handle processes the OrderCancelledEvent message
func (h *OrderCancelledEventHandler) Handle(ctx context.Context, msgBody []byte) {
	var event events.OrderCancelledEvent
	if err := json.Unmarshal(msgBody, &event); err != nil {
		h.logger.Exception(ctx, "Failed to unmarshal OrderCancelledEvent", err)
		h.sendToDLQ(msgBody)
		return
	}

	// Get the order to retrieve product information
	order, err := h.orderRepository.GetOrderByID(ctx, event.OrderID)
	if err != nil {
		h.logger.Exception(ctx, "Failed to get order for cancellation", err)
		h.sendToDLQ(msgBody)
		return
	}

	if order == nil {
		h.logger.Warn(ctx, "Order not found for cancellation: "+event.OrderID)
		return
	}

	// Delegate to inventory service to release reserved product
	err = h.inventoryService.ReleaseReservedProduct(ctx, order.Product.ID, order.Product.Quantity)
	if err != nil {
		h.logger.Exception(ctx, "Error releasing reserved product through inventory service", err)
		h.sendToDLQ(msgBody)
		return
	}

	// Update order status to cancelled
	update := map[string]any{"status": "Cancelled"}
	err = h.orderRepository.UpdateOrder(ctx, event.OrderID, update)
	if err != nil {
		h.logger.Exception(ctx, "Failed to update order status to cancelled", err)
		h.sendToDLQ(msgBody)
		return
	}

	h.logger.Info(ctx, "Order cancelled and inventory released for order: "+event.OrderID)
}

func (h *OrderCancelledEventHandler) sendToDLQ(body []byte) {
	// Simply send to DLQ queue - another process will handle storing to MongoDB
	err := h.rabbitMQService.Publish("order.cancelled.dlq", body)
	if err != nil {
		// Use context.TODO() since we don't have ctx in this method
		h.logger.Exception(context.TODO(), "Failed to send event to DLQ", err)
	}
}
