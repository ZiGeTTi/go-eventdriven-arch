package handlers

import (
	"context"
	"encoding/json"
	"go-order-eda/src/infrastructure/log"
	rabbitmq "go-order-eda/src/infrastructure/rabbitmq"
	"go-order-eda/src/services/events"
	"go-order-eda/src/services/notification"
	"time"
)

type InventoryStatusUpdatedEventHandler struct {
	rabbitMQService     *rabbitmq.RabbitMQServiceImpl
	notificationService notification.NotificationService
	logger              log.Logger
}

func NewInventoryStatusUpdatedEventHandler(
	rabbit *rabbitmq.RabbitMQServiceImpl,
	notificationService notification.NotificationService,
	logger log.Logger,
) *InventoryStatusUpdatedEventHandler {
	return &InventoryStatusUpdatedEventHandler{
		rabbitMQService:     rabbit,
		notificationService: notificationService,
		logger:              logger,
	}
}

// Handle processes the InventoryStatusUpdatedEvent message
func (h *InventoryStatusUpdatedEventHandler) Handle(ctx context.Context, msgBody []byte) {
	var event events.InventoryStatusUpdatedEvent
	if err := json.Unmarshal(msgBody, &event); err != nil {
		h.logger.Exception(ctx, "Failed to unmarshal InventoryStatusUpdatedEvent", err)
		h.sendToDLQ(msgBody)
		return
	}

	// Send notification based on inventory status
	if event.HasStock {
		h.logger.Info(ctx, "Sending order confirmation notification for product: "+event.ProductID)

		// Send confirmation notification
		notificationReq := notification.NotificationRequest{
			OrderID:     event.OrderID,
			ProductID:   event.ProductID,
			Message:     "Your order has been confirmed! Product: " + event.ProductID,
			Channel:     notification.ChannelEmail, // Default to email
			Recipient:   "customer@example.com",    // TODO: Get actual customer email from order
			MessageType: "confirmation",
		}

		// Send notification via multiple channels
		err := h.notificationService.SendMultiChannelNotification(ctx, notificationReq,
			[]notification.NotificationChannel{
				notification.ChannelEmail,
				notification.ChannelPush,
			})
		if err != nil {
			h.logger.Exception(ctx, "Failed to send confirmation notification", err)
		}
	} else {
		h.logger.Info(ctx, "No stock available for product: "+event.ProductID+", cancelling order: "+event.OrderID)

		// Send cancellation notification
		notificationReq := notification.NotificationRequest{
			OrderID:     event.OrderID,
			ProductID:   event.ProductID,
			Message:     "Your order has been cancelled due to insufficient stock. Product: " + event.ProductID,
			Channel:     notification.ChannelEmail, // Default to email
			Recipient:   "customer@example.com",    // TODO: Get actual customer email from order
			MessageType: "cancellation",
		}

		// Send notification via multiple channels
		err := h.notificationService.SendMultiChannelNotification(ctx, notificationReq,
			[]notification.NotificationChannel{
				notification.ChannelEmail,
				notification.ChannelSMS, // SMS for urgent cancellations
			})
		if err != nil {
			h.logger.Exception(ctx, "Failed to send cancellation notification", err)
		}

		// Fire OrderCancelled event when there's no stock
		orderCancelledEvent := events.OrderCancelledEvent{
			OrderID:   event.OrderID,
			Status:    "Cancelled",
			Version:   1,
			TimeStamp: time.Now().Local(),
		}

		cancelledEventJSON, err := json.Marshal(orderCancelledEvent)
		if err != nil {
			h.logger.Exception(ctx, "Failed to marshal OrderCancelledEvent", err)
			h.sendToDLQ(msgBody)
			return
		}

		err = h.rabbitMQService.Publish(events.OrderCancelled, cancelledEventJSON)
		if err != nil {
			h.logger.Exception(ctx, "Failed to publish OrderCancelledEvent", err)
			h.sendToDLQ(msgBody)
			return
		}

		h.logger.Info(ctx, "OrderCancelled event published for order: "+event.OrderID)
	}

	// Publish NotificationSentEvent
	notificationEvent := events.NotificationSentEvent{
		OrderID:   event.OrderID, // ✅ Use actual OrderID from event chain
		Message:   getNotificationMessage(event.HasStock, event.ProductID),
		Version:   1,
		TimeStamp: time.Now().Local(),
	}

	notificationJSON, err := json.Marshal(notificationEvent)
	if err != nil {
		h.logger.Exception(ctx, "Failed to marshal NotificationSentEvent", err)
		h.sendToDLQ(msgBody)
		return
	}

	err = h.rabbitMQService.Publish(events.NotificationSent, notificationJSON)
	if err != nil {
		h.logger.Exception(ctx, "Failed to publish NotificationSentEvent", err)
		h.sendToDLQ(msgBody)
		return
	}

	h.logger.Info(ctx, "Notification sent and event published for order: "+event.OrderID+" product: "+event.ProductID)
}

func getNotificationMessage(hasStock bool, productID string) string {
	if hasStock {
		return "Order confirmed for product: " + productID
	}
	return "Order cancelled due to insufficient stock for product: " + productID
}

func (h *InventoryStatusUpdatedEventHandler) sendToDLQ(body []byte) {
	// Simply send to DLQ queue - another process will handle storing to MongoDB
	err := h.rabbitMQService.Publish("inventory.status.updated.dlq", body)
	if err != nil {
		h.logger.Exception(context.TODO(), "Failed to send event to DLQ", err)
	}
}
