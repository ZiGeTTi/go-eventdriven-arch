package events

// EventHandler defines the interface for handling messages from a queue.
type EventHandler interface {
	Handle(topic string, body []byte) error
}
