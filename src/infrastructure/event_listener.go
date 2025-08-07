package infrastructure

import (
	"context"
	"fmt"
	"go-order-eda/src/infrastructure/log"
	rabbitmq "go-order-eda/src/infrastructure/rabbitmq"
	"sync"
	"time"
)

type EventListener struct {
	rabbitMQService *rabbitmq.RabbitMQServiceImpl
	logger          log.Logger
	handlers        map[string]EventHandler
}

type EventHandler interface {
	Handle(ctx context.Context, msgBody []byte)
}

func NewEventListener(rabbit *rabbitmq.RabbitMQServiceImpl, logger log.Logger) *EventListener {
	return &EventListener{
		rabbitMQService: rabbit,
		logger:          logger,
		handlers:        make(map[string]EventHandler),
	}
}

// RegisterHandler registers an event handler for a specific event type
func (el *EventListener) RegisterHandler(eventType string, handler EventHandler) {
	el.handlers[eventType] = handler
}

// StartListening starts listening for events in background goroutines
func (el *EventListener) StartListening(ctx context.Context) error {
	var wg sync.WaitGroup

	for eventType, handler := range el.handlers {
		wg.Add(1)
		go func(evtType string, h EventHandler) {
			defer wg.Done()
			el.listenToQueue(ctx, evtType, h)
		}(eventType, handler)
	}

	// Wait for all goroutines to finish (they run indefinitely unless context is cancelled)
	wg.Wait()
	return nil
}

// listenToQueue listens to a specific queue and processes messages with retry logic
func (el *EventListener) listenToQueue(ctx context.Context, eventType string, handler EventHandler) {
	queueName := eventType
	maxRetries := 5
	retryDelay := time.Second * 2

	el.logger.Info(ctx, "Starting to listen for events on queue: "+queueName)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		msgs, err := el.rabbitMQService.Consume(queueName)
		if err != nil {
			el.logger.Exception(ctx, fmt.Sprintf("Failed to start consuming queue: %s (attempt %d/%d)", queueName, attempt, maxRetries), err)

			if attempt == maxRetries {
				el.logger.Exception(ctx, "Max retries reached for queue: "+queueName+", giving up", err)
				return
			}

			// Wait before retrying
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
			continue
		}

		el.logger.Info(ctx, "Successfully started consuming queue: "+queueName)

		// Process messages
		for {
			select {
			case <-ctx.Done():
				el.logger.Info(ctx, "Stopping event listener for queue: "+queueName)
				return
			case msg, ok := <-msgs:
				if !ok {
					el.logger.Warn(ctx, "Message channel closed for queue: "+queueName+", attempting to reconnect...")
					break // Exit inner loop to retry connection
				}
				// Process message in a separate goroutine to avoid blocking
				go func() {
					handler.Handle(ctx, msg.Body)
					msg.Ack(false)
				}()
			}
		}
	}
}
