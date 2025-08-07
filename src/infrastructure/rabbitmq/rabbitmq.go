package rabbitmq

import (
	"fmt"

	"github.com/streadway/amqp"
)

// RabbitMQServiceImpl is an implementation of the RabbitMQService interface.
type RabbitMQServiceImpl struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQService(host, exchange, queueName string) (*RabbitMQServiceImpl, error) {
	conn, err := amqp.Dial(host)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Remove publisher confirmation for now to avoid timeout issues
	// TODO: Implement proper publisher confirmation later if needed

	err = ch.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare an exchange: %w", err)
	}
	// dead-letter exchange
	dlxName := exchange + ".dlx"
	err = ch.ExchangeDeclare(
		dlxName,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare a dead-letter exchange: %w", err)
	}

	dlqName := queueName + ".dlq"
	_, err = ch.QueueDeclare(
		dlqName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare a dead-letter queue: %w", err)
	}

	// Bind the dead-letter queue to the dead-letter exchange
	err = ch.QueueBind(
		dlqName,
		"", // routing key gerek yok gibi
		dlxName,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bind dead-letter queue: %w", err)
	}

	// Declare the main queue with dead-lettering enabled
	args := amqp.Table{
		"x-dead-letter-exchange": dlxName,
	}
	_, err = ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		args,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare a queue: %w", err)
	}

	// Declare event-specific queues
	eventQueues := []string{
		"order.requested", // New: Initial order request queue
		"order.created",
		"order.cancelled",
		"inventory.status.updated",
		"notification.sent",
	}

	for _, eventQueue := range eventQueues {
		_, err = ch.QueueDeclare(
			eventQueue,
			true,
			false,
			false,
			false,
			args,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to declare event queue %s: %w", eventQueue, err)
		}

		// Bind queue to exchange with routing key
		err = ch.QueueBind(
			eventQueue, // queue name
			eventQueue, // routing key (same as queue name)
			exchange,   // exchange
			false,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to bind event queue %s: %w", eventQueue, err)
		}

		// Declare DLQ for each event queue
		dlqName := eventQueue + ".dlq"
		_, err = ch.QueueDeclare(
			dlqName,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to declare DLQ %s: %w", dlqName, err)
		}

		// Bind DLQ to exchange
		err = ch.QueueBind(
			dlqName,  // queue name
			dlqName,  // routing key (same as queue name)
			exchange, // exchange
			false,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to bind DLQ %s: %w", dlqName, err)
		}
	}

	return &RabbitMQServiceImpl{
		conn:    conn,
		channel: ch,
	}, nil
}

// Publish sends a message to a topic on the exchange with proper error handling.
// The message is made persistent to ensure durability across broker restarts.
// Returns an error if the connection is closed or publishing fails.
func (s *RabbitMQServiceImpl) Publish(topic string, body []byte) error {
	// Validate input parameters
	if topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}
	if body == nil {
		return fmt.Errorf("message body cannot be nil")
	}

	// Check connection health
	if s.conn.IsClosed() {
		return fmt.Errorf("connection to RabbitMQ is closed")
	}
	if s.channel == nil {
		return fmt.Errorf("channel is not initialized")
	}

	// Publish the message
	err := s.channel.Publish(
		"order_events", // exchange
		topic,          // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,                        // Make message persistent for durability
			MessageId:    fmt.Sprintf("%s_%d", topic, len(body)), // Simple message ID for tracking
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message to topic '%s': %w", topic, err)
	}

	// Message published successfully
	// Note: Publisher confirmation is disabled to avoid timeout issues
	// TODO: Implement proper publisher confirmation with dedicated channel if needed
	return nil
}

// Close closes the connection to RabbitMQ.
func (s *RabbitMQServiceImpl) Close() {
	s.channel.Close()
	s.conn.Close()
}

// Consume starts consuming messages from a queue.
func (s *RabbitMQServiceImpl) Consume(queueName string) (<-chan amqp.Delivery, error) {
	// Check if connection and channel are still open
	if s.conn.IsClosed() {
		return nil, fmt.Errorf("connection is closed")
	}

	msgs, err := s.channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start consuming queue: %w", err)
	}
	return msgs, nil
}

// IsHealthy checks if the RabbitMQ connection is healthy
func (s *RabbitMQServiceImpl) IsHealthy() bool {
	return !s.conn.IsClosed() && s.channel != nil
}
