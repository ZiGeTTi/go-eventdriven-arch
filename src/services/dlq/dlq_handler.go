package dlq

import (
	"context"
	"encoding/json"
	"go-order-eda/src/infrastructure/log"
	"go-order-eda/src/services/events"
	"go-order-eda/src/services/order/domain/persistence"
)

type DLQHandler struct {
	orderRepository *persistence.OrderRepository
	logger          log.Logger
}

// DLQ wrapper structs to implement EventHandler interface
type OrderCreatedDLQHandler struct {
	*DLQHandler
}

type OrderCancelledDLQHandler struct {
	*DLQHandler
}

type InventoryStatusUpdatedDLQHandler struct {
	*DLQHandler
}

func NewDLQHandler(
	orderRepo *persistence.OrderRepository,
	logger log.Logger,
) *DLQHandler {
	return &DLQHandler{
		orderRepository: orderRepo,
		logger:          logger,
	}
}

func (d *DLQHandler) NewOrderCreatedDLQHandler() *OrderCreatedDLQHandler {
	return &OrderCreatedDLQHandler{DLQHandler: d}
}

func (d *DLQHandler) NewOrderCancelledDLQHandler() *OrderCancelledDLQHandler {
	return &OrderCancelledDLQHandler{DLQHandler: d}
}

func (d *DLQHandler) NewInventoryStatusUpdatedDLQHandler() *InventoryStatusUpdatedDLQHandler {
	return &InventoryStatusUpdatedDLQHandler{DLQHandler: d}
}

// EventHandler interface implementations
func (h *OrderCreatedDLQHandler) Handle(ctx context.Context, msgBody []byte) {
	h.HandleOrderCreatedDLQ(ctx, msgBody)
}

func (h *OrderCancelledDLQHandler) Handle(ctx context.Context, msgBody []byte) {
	h.HandleOrderCancelledDLQ(ctx, msgBody)
}

func (h *InventoryStatusUpdatedDLQHandler) Handle(ctx context.Context, msgBody []byte) {
	h.HandleInventoryStatusUpdatedDLQ(ctx, msgBody)
}

// HandleOrderCreatedDLQ handles failed OrderCreated events from DLQ
func (h *DLQHandler) HandleOrderCreatedDLQ(ctx context.Context, msgBody []byte) {
	h.logger.Info(ctx, "Processing OrderCreated DLQ event")

	// Try to extract orderID from the event
	var event events.OrderCreatedEvent
	orderID := "unknown"
	if err := json.Unmarshal(msgBody, &event); err == nil {
		orderID = event.ID
	}

	// Store the failed event for replay
	err := h.orderRepository.StoreEventForReplay(ctx, orderID, msgBody)
	if err != nil {
		h.logger.Exception(ctx, "Failed to store OrderCreated DLQ event for replay", err)
	} else {
		h.logger.Info(ctx, "OrderCreated DLQ event stored for replay, orderID: "+orderID)
	}
}

// HandleOrderCancelledDLQ handles failed OrderCancelled events from DLQ
func (h *DLQHandler) HandleOrderCancelledDLQ(ctx context.Context, msgBody []byte) {
	h.logger.Info(ctx, "Processing OrderCancelled DLQ event")

	// Try to extract orderID from the event
	var event events.OrderCancelledEvent
	orderID := "unknown"
	if err := json.Unmarshal(msgBody, &event); err == nil {
		orderID = event.OrderID
	}

	// Store the failed event for replay
	err := h.orderRepository.StoreEventForReplay(ctx, orderID, msgBody)
	if err != nil {
		h.logger.Exception(ctx, "Failed to store OrderCancelled DLQ event for replay", err)
	} else {
		h.logger.Info(ctx, "OrderCancelled DLQ event stored for replay, orderID: "+orderID)
	}
}

// HandleInventoryStatusUpdatedDLQ handles failed InventoryStatusUpdated events from DLQ
func (h *DLQHandler) HandleInventoryStatusUpdatedDLQ(ctx context.Context, msgBody []byte) {
	h.logger.Info(ctx, "Processing InventoryStatusUpdated DLQ event")

	// Try to extract orderID from the event
	var event events.InventoryStatusUpdatedEvent
	orderID := "unknown"
	if err := json.Unmarshal(msgBody, &event); err == nil {
		orderID = event.OrderID
	}

	// Store the failed event for replay
	err := h.orderRepository.StoreEventForReplay(ctx, orderID, msgBody)
	if err != nil {
		h.logger.Exception(ctx, "Failed to store InventoryStatusUpdated DLQ event for replay", err)
	} else {
		h.logger.Info(ctx, "InventoryStatusUpdated DLQ event stored for replay, orderID: "+orderID)
	}
}
