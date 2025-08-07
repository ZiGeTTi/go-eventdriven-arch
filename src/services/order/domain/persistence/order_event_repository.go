package persistence

import (
	"context"
	"go-order-eda/src/services/events"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type OrderEvent struct {
	ID         string     `bson:"_id,omitempty"`
	OrderID    string     `bson:"orderId"`
	EventData  []byte     `bson:"eventData"`
	CreatedAt  time.Time  `bson:"createdAt"`
	Replayed   bool       `bson:"replayed"`
	ReplayedAt *time.Time `bson:"replayedAt,omitempty"`
	Status     string     `bson:"status"`
}

// GetUnreplayedEvents fetches events that have not been replayed yet
// Events are returned in FIFO order (oldest first) based on createdAt timestamp
func (r *OrderRepository) GetUnreplayedEvents(ctx context.Context, limit int64) ([]OrderEvent, error) {
	coll := r.collection.Database().Collection("order_events")
	filter := bson.M{
		"replayed": bson.M{"$ne": true},
		"status":   bson.M{"$in": []string{events.EventStatusPending, events.EventStatusFailed}},
	}
	opts := options.Find().SetLimit(limit).SetSort(bson.D{bson.E{Key: "createdAt", Value: 1}}) // 1 = ascending (FIFO)
	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var events []OrderEvent
	for cursor.Next(ctx) {
		var evt OrderEvent
		if err := cursor.Decode(&evt); err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, nil
}

// MarkEventReplayed marks an event as successfully replayed
// Use this method when replaying events from the order_events collection
func (r *OrderRepository) MarkEventReplayed(ctx context.Context, eventID string) error {
	return r.MarkEventAsCompleted(ctx, eventID)
}

// MarkEventAsReplaying marks an event as currently being replayed
func (r *OrderRepository) MarkEventAsReplaying(ctx context.Context, eventID string) error {
	coll := r.collection.Database().Collection("order_events")
	_, err := coll.UpdateOne(ctx, bson.M{"_id": eventID}, bson.M{"$set": bson.M{
		"status": events.EventStatusReplaying,
	}})
	return err
}

// MarkEventAsCompleted marks an event as successfully completed
// Use this when an event has been successfully processed (either first time or after replay)
func (r *OrderRepository) MarkEventAsCompleted(ctx context.Context, eventID string) error {
	coll := r.collection.Database().Collection("order_events")
	now := time.Now().Local()
	_, err := coll.UpdateOne(ctx, bson.M{"_id": eventID}, bson.M{"$set": bson.M{
		"status":     events.EventStatusCompleted,
		"replayed":   true,
		"replayedAt": now,
	}})
	return err
}

// MarkEventAsFailed marks an event as failed for future replay
// Use this when event processing fails and should be retried later
func (r *OrderRepository) MarkEventAsFailed(ctx context.Context, eventID string) error {
	coll := r.collection.Database().Collection("order_events")
	_, err := coll.UpdateOne(ctx, bson.M{"_id": eventID}, bson.M{"$set": bson.M{
		"status": events.EventStatusFailed,
	}})
	return err
}
