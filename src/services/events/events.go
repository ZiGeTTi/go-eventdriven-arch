package events

import (
	"errors"
	"time"
)

const (
	// Event types
	OrderRequested         = "order.requested"     // New: Initial order request
	OrderCreated           = "order.created"
	OrderCancelled         = "order.cancelled"
	InventoryStatusUpdated = "inventory.status.updated"
	NotificationSent       = "notification.sent"
	
	// Event status enums for order_events collection
	EventStatusPending   = "pending"   // Event is waiting to be processed
	EventStatusFailed    = "failed"    // Event processing failed, needs replay
	EventStatusCompleted = "completed" // Event was successfully processed
	EventStatusReplaying = "replaying" // Event is currently being replayed
	
	// Order status enums
	OrderStatusRequested = "Requested"
	OrderStatusCreated   = "Created"
	OrderStatusCancelled = "Cancelled"
	OrderStatusCompleted = "Completed"
	OrderStatusFailed    = "Failed"
)

type OrderRequestedEvent struct {
	ID        string    `json:"id"`
	Product   Product   `json:"product"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	Version   int       `json:"version"`
	TimeStamp time.Time `json:"timestamp"`
}

func (e *OrderRequestedEvent) Validate() error {
	if e.ID == "" || e.Product.ID == "" || e.Product.Quantity <= 0 {
		return errors.New("missing required fields in OrderRequestedEvent")
	}
	return nil
}

type OrderCreatedEvent struct {
	ID        string    `json:"id"`
	Product   Product   `json:"product"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	Version   int       `json:"version"`
	TimeStamp time.Time `json:"timestamp"`
}

func (e *OrderCreatedEvent) Validate() error {
	if e.ID == "" || e.Product.ID == "" || e.Status == "" {
		return errors.New("missing required fields in OrderCreatedEvent")
	}
	return nil
}

type Product struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

type OrderCancelledEvent struct {
	OrderID   string    `json:"orderId"`
	Status    string    `json:"status"`
	Version   int       `json:"version"`
	TimeStamp time.Time `json:"timestamp"`
}

func (e *OrderCancelledEvent) Validate() error {
	if e.OrderID == "" || e.Status == "" {
		return errors.New("missing required fields in OrderCancelledEvent")
	}
	return nil
}

type InventoryStatusUpdatedEvent struct {
	OrderID   string    `json:"orderId"` // Add OrderID to maintain event chain
	ProductID string    `json:"productId"`
	HasStock  bool      `json:"hasStock"`
	Version   int       `json:"version"`
	TimeStamp time.Time `json:"timestamp"`
}

func (e *InventoryStatusUpdatedEvent) Validate() error {
	if e.OrderID == "" || e.ProductID == "" {
		return errors.New("missing required fields in InventoryStatusUpdatedEvent")
	}
	return nil
}

type NotificationSentEvent struct {
	OrderID   string    `json:"orderId"`
	Message   string    `json:"message"`
	Version   int       `json:"version"`
	TimeStamp time.Time `json:"timestamp"`
}

func (e *NotificationSentEvent) Validate() error {
	if e.OrderID == "" || e.Message == "" {
		return errors.New("missing required fields in NotificationSentEvent")
	}
	return nil
}
