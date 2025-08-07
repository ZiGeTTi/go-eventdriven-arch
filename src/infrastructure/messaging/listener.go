package messaging

import (
	"go-order-eda/src/services/events"

	"github.com/streadway/amqp"
)

// EventListener listens for messages on a queue and dispatches them to handlers.
type EventListener struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	handlers map[string]events.EventHandler
}

// NewEventListener creates a new event listener.
func NewEventListener(conn *amqp.Connection) (*EventListener, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &EventListener{
		conn:     conn,
		channel:  ch,
		handlers: make(map[string]events.EventHandler),
	}, nil
}

// RegisterHandler registers an event handler for a specific topic.
func (l *EventListener) RegisterHandler(topic string, handler events.EventHandler) {
	l.handlers[topic] = handler
}

// Listen starts listening for messages on a specific queue and binds it to topics.
func (l *EventListener) Listen(queueName string, topics []string, exchangeName string) error {
	q, err := l.channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return err
	}

	for _, topic := range topics {
		err = l.channel.QueueBind(
			q.Name,       // queue name
			topic,        // routing key
			exchangeName, // exchange
			false,
			nil,
		)
		if err != nil {
			return err
		}
	}

	msgs, err := l.channel.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			if handler, ok := l.handlers[d.RoutingKey]; ok {
				handler.Handle(d.RoutingKey, d.Body)
			}
		}
	}()

	return nil
}

// Close closes the channel and connection.
func (l *EventListener) Close() {
	l.channel.Close()
	l.conn.Close()
}
