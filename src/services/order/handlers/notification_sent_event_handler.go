package handlers

import (
	"context"
	"encoding/json"
	"go-order-eda/src/infrastructure/log"
	"go-order-eda/src/services/events"
	"go-order-eda/src/services/order/domain/persistence"
)

type NotificationSentEventHandler struct {
	orderRepository *persistence.OrderRepository
	logger          log.Logger
}

func NewNotificationSentEventHandler(
	orderRepo *persistence.OrderRepository,
	logger log.Logger,
) *NotificationSentEventHandler {
	return &NotificationSentEventHandler{
		orderRepository: orderRepo,
		logger:          logger,
	}
}

// Handle processes the NotificationSentEvent message
func (h *NotificationSentEventHandler) Handle(ctx context.Context, msgBody []byte) {
	var event events.NotificationSentEvent
	if err := json.Unmarshal(msgBody, &event); err != nil {
		h.logger.Exception(ctx, "Failed to unmarshal NotificationSentEvent", err)
		return
	}

	// Update order with notification status
	update := map[string]interface{}{
		"notificationStatus":  "sent",
		"notificationMessage": event.Message,
	}

	err := h.orderRepository.UpdateOrder(ctx, event.OrderID, update)
	if err != nil {
		h.logger.Exception(ctx, "Failed to update order with notification status", err)
		return
	}

	h.logger.Info(ctx, "Order updated with notification status for order: "+event.OrderID)
}
