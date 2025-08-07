package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"go-order-eda/src/config"
	"go-order-eda/src/services/events"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type OrderRepository struct {
	collection *mongo.Collection
}

// OrderDocument is the storage model for MongoDB
type OrderDocument struct {
	ID        string          `bson:"id"`
	Amount    float64         `bson:"amount"`
	Status    string          `bson:"status"`
	Product   ProductDocument `bson:"product"`
	CreatedAt time.Time       `bson:"created_at"`
}
type ProductDocument struct {
	ID       string `bson:"id"`
	Name     string `bson:"name"`
	Quantity int    `bson:"quantity"`
}

func NewOrderRepository(cfg *config.Config, client *mongo.Client) *OrderRepository {
	return &OrderRepository{
		collection: client.Database(cfg.MongoDBDatabaseName).Collection("orders"),
	}
}

func (r *OrderRepository) CreateOrder(ctx context.Context, order *OrderDocument) (string, error) {
	doc := OrderDocument{
		ID:     order.ID, // Fix: Use the provided ID
		Amount: order.Amount,
		Status: order.Status,
		Product: ProductDocument{
			ID:       order.Product.ID,
			Name:     order.Product.Name,
			Quantity: order.Product.Quantity,
		},
		CreatedAt: time.Now().Local(), // Use local time
	}

	_, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return "", err
	}
	return doc.ID, nil
}

func (r *OrderRepository) GetOrderByID(ctx context.Context, id string) (*OrderDocument, error) {
	var doc OrderDocument
	err := r.collection.FindOne(ctx, bson.M{"id": id}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *OrderRepository) UpdateOrder(ctx context.Context, id string, update bson.M) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": update})
	return err
}

func (r *OrderRepository) CancelOrder(ctx context.Context, id string) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": bson.M{"status": "cancelled"}})
	return err
}
func (r *OrderRepository) StoreEventForReplay(ctx context.Context, orderID string, eventData []byte) error {
	// Validate that eventData is valid JSON
	if !json.Valid(eventData) {
		return errors.New("invalid JSON event data")
	}

	// Create OrderEvent document with proper structure
	eventDoc := OrderEvent{
		ID:        primitive.NewObjectID().Hex(), // Generate unique ID
		OrderID:   orderID,
		EventData: eventData, // Store as raw JSON bytes
		CreatedAt: time.Now().Local(),
		Replayed:  false,                    // Initially not replayed
		Status:    events.EventStatusFailed, // Mark as failed for DLQ events
	}

	coll := r.collection.Database().Collection("order_events")
	_, err := coll.InsertOne(ctx, eventDoc)
	return err
}

// StoreEventAsPending stores an event with pending status for tracking
func (r *OrderRepository) StoreEventAsPending(ctx context.Context, orderID string, eventData []byte) (string, error) {
	// Validate that eventData is valid JSON
	if !json.Valid(eventData) {
		return "", errors.New("invalid JSON event data")
	}

	// Create OrderEvent document with pending status
	eventDoc := OrderEvent{
		ID:        primitive.NewObjectID().Hex(), // Generate unique ID
		OrderID:   orderID,
		EventData: eventData, // Store as raw JSON bytes
		CreatedAt: time.Now().Local(),
		Replayed:  false,                     // Not yet processed
		Status:    events.EventStatusPending, // Mark as pending for new events
	}

	coll := r.collection.Database().Collection("order_events")
	_, err := coll.InsertOne(ctx, eventDoc)
	if err != nil {
		return "", err
	}
	return eventDoc.ID, nil
}

// UpdateEventData updates the event data with the tracking ID
func (r *OrderRepository) UpdateEventData(ctx context.Context, eventID string, eventData []byte) error {
	coll := r.collection.Database().Collection("order_events")
	_, err := coll.UpdateOne(ctx, bson.M{"_id": eventID}, bson.M{"$set": bson.M{
		"eventData": eventData,
	}})
	return err
}
