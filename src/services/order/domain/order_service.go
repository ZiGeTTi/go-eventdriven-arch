package domain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-order-eda/src/infrastructure/log"
	"go-order-eda/src/infrastructure/rabbitmq"
	"go-order-eda/src/services/events"
	"go-order-eda/src/services/order/domain/persistence"
	"time"
)

type OrderService interface {
	CreateOrder(ctx context.Context, order Order) (string, error)
	CancelOrder(ctx context.Context, orderID string) error
	ReplayFailedEvents(ctx context.Context) error
}

type orderService struct {
	logger          log.Logger
	rabbitMQService rabbitmq.RabbitMQServiceImpl
	orderRepository *persistence.OrderRepository
}

func NewOrderService(
	logger log.Logger,
	rabbitMQService rabbitmq.RabbitMQServiceImpl,
	orderRepository *persistence.OrderRepository,
) *orderService {
	return &orderService{
		logger:          logger,
		rabbitMQService: rabbitMQService,
		orderRepository: orderRepository,
	}
}

// CreateOrder initiates the order creation process by publishing an OrderRequested event.
// This follows the event sourcing pattern where the actual order creation happens in handlers.
// Returns the order ID and any error that occurred during event publishing.
func (s *orderService) CreateOrder(ctx context.Context, order Order) (string, error) {
	if order.ID == "" {
		return "", errors.New("order ID is required")
	}

	// Validate order data
	if order.Product.ID == "" {
		return "", errors.New("product ID is required")
	}
	if order.Product.Quantity <= 0 {
		return "", errors.New("product quantity must be greater than 0")
	}
	if order.Amount <= 0 {
		return "", errors.New("order amount must be greater than 0")
	}

	// Create OrderRequested event
	orderRequestedEvent := events.OrderRequestedEvent{
		ID:        order.ID,
		Product:   events.Product{ID: order.Product.ID, Name: order.Product.Name, Quantity: order.Product.Quantity},
		Amount:    order.Amount,
		Status:    events.OrderStatusRequested,
		Version:   1,
		TimeStamp: time.Now().Local(),
	}

	// Validate the event before publishing
	if err := orderRequestedEvent.Validate(); err != nil {
		s.logger.Exception(ctx, "Order requested event validation failed", err)
		return "", fmt.Errorf("invalid order request: %w", err)
	}

	eventJSON, err := json.Marshal(orderRequestedEvent)
	if err != nil {
		s.logger.Exception(ctx, "failed to marshal order requested event", err)
		return "", fmt.Errorf("failed to process order request: %w", err)
	}

	// Publish with retry logic
	const maxRetries = 2
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = s.rabbitMQService.Publish(events.OrderRequested, eventJSON)
		if err == nil {
			break
		}
		s.logger.Warn(ctx, fmt.Sprintf("Publish OrderRequested failed for order %s, attempt %d/%d: %v",
			order.ID, attempt, maxRetries, err))

		if attempt < maxRetries {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	if err != nil {
		s.logger.Exception(ctx, fmt.Sprintf("failed to publish order requested event for order %s after %d retries",
			order.ID, maxRetries), err)
		return "", fmt.Errorf("failed to publish order request: %w", err)
	}

	s.logger.Info(ctx, fmt.Sprintf("OrderRequested event published successfully for order: %s", order.ID))
	return order.ID, nil
}

// CancelOrder initiates the order cancellation process by publishing an OrderCancelled event.
// This follows the event-driven pattern where the cancellation is processed asynchronously.
func (s *orderService) CancelOrder(ctx context.Context, orderID string) error {
	if orderID == "" {
		return errors.New("order ID is required for cancellation")
	}
	cancellationEvent := events.OrderCancelledEvent{
		OrderID:   orderID,
		Status:    events.OrderStatusCancelled,
		Version:   1,
		TimeStamp: time.Now().Local(),
	}

	// Validate the event before publishing
	if err := cancellationEvent.Validate(); err != nil {
		s.logger.Exception(ctx, "Order cancelled event validation failed", err)
		return fmt.Errorf("invalid cancellation request: %w", err)
	}
	eventJSON, err := json.Marshal(cancellationEvent)
	if err != nil {
		s.logger.Exception(ctx, fmt.Sprintf("failed to marshal cancellation event for order %s", orderID), err)
		return fmt.Errorf("failed to process cancellation: %w", err)
	}

	// Publish with retry logic
	const maxRetries = 2
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = s.rabbitMQService.Publish(events.OrderCancelled, eventJSON)
		if err == nil {
			break
		}
		s.logger.Warn(ctx, fmt.Sprintf("Publish OrderCancelled failed for order %s, attempt %d/%d: %v",
			orderID, attempt, maxRetries, err))

		if attempt < maxRetries {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	if err != nil {
		s.logger.Exception(ctx, fmt.Sprintf("failed to publish order cancelled event for order %s after %d retries",
			orderID, maxRetries), err)
		return fmt.Errorf("failed to publish cancellation event: %w", err)
	}

	s.logger.Info(ctx, fmt.Sprintf("OrderCancelled event published successfully for order: %s", orderID))
	return nil
}

// ReplayFailedEvents processes failed events from the order_events collection
// and attempts to republish them with retry logic and proper status tracking.
func (s *orderService) ReplayFailedEvents(ctx context.Context) error {
	const batchSize = 100
	const maxRetries = 3

	// Fetch unreplayed events in batches for better memory management
	events, err := s.orderRepository.GetUnreplayedEvents(ctx, batchSize)
	if err != nil {
		s.logger.Exception(ctx, "failed to fetch unreplayed events", err)
		return fmt.Errorf("failed to fetch unreplayed events: %w", err)
	}

	if len(events) == 0 {
		s.logger.Info(ctx, "No events to replay")
		return nil
	}

	s.logger.Info(ctx, fmt.Sprintf("Starting replay of %d failed events", len(events)))

	successCount := 0
	failureCount := 0

	for _, evt := range events {
		// Mark event as being replayed for audit trail
		if err := s.orderRepository.MarkEventAsReplaying(ctx, evt.ID); err != nil {
			s.logger.Warn(ctx, fmt.Sprintf("Failed to mark event %s as replaying: %v", evt.ID, err))
		}

		// Attempt to republish with retry logic
		var pubErr error
		for attempt := 1; attempt <= maxRetries; attempt++ {
			// TODO: Should determine correct routing key based on event type instead of hardcoding
			pubErr = s.rabbitMQService.Publish("order.created", evt.EventData)
			if pubErr == nil {
				break
			}
			s.logger.Warn(ctx, fmt.Sprintf("Replay publish failed for event %s, attempt %d/%d: %v",
				evt.ID, attempt, maxRetries, pubErr))

			// Exponential backoff: 1s, 2s, 3s
			time.Sleep(time.Duration(attempt) * time.Second)
		}
		if pubErr == nil {
			if err := s.orderRepository.MarkEventAsCompleted(ctx, evt.ID); err != nil {
				s.logger.Warn(ctx, fmt.Sprintf("Failed to mark event %s as completed: %v", evt.ID, err))
			} else {
				s.logger.Info(ctx, fmt.Sprintf("Event %s successfully replayed and marked as completed", evt.ID))
				successCount++
			}
		} else {
			s.logger.Exception(ctx, fmt.Sprintf("Replay failed for event %s after %d retries", evt.ID, maxRetries), pubErr)
			if err := s.orderRepository.MarkEventAsFailed(ctx, evt.ID); err != nil {
				s.logger.Warn(ctx, fmt.Sprintf("Failed to mark event %s as failed: %v", evt.ID, err))
			}
			failureCount++
		}
	}

	s.logger.Info(ctx, fmt.Sprintf("Replay completed: %d successful, %d failed", successCount, failureCount))

	if failureCount > 0 {
		return fmt.Errorf("replay completed with %d failures out of %d events", failureCount, len(events))
	}

	return nil
}
