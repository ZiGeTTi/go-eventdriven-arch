package inventory

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Product struct {
	ID       string `bson:"id"`
	Name     string `bson:"name"`
	Quantity int    `bson:"quantity"`
	Reserved int    `bson:"reserved"`
}
type ProductRepository interface {
	CheckAndReserveProduct(ctx context.Context, productID string, quantity int) (bool, error)
	ReleaseReservedProduct(ctx context.Context, productID string, quantity int) error
	SeedProduct(ctx context.Context, product Product) error
	// New business logic methods
	GetProductById(ctx context.Context, productID string) (*Product, error)
	UpdateProductQuantity(ctx context.Context, productID string, quantity int) error
	GetLowStockProducts(ctx context.Context, threshold int) ([]Product, error)
	AddProduct(ctx context.Context, product Product) error
	GetAllProducts(ctx context.Context) ([]Product, error)
}

type productRepository struct {
	collection *mongo.Collection
}

func NewProductRepository(db *mongo.Database) ProductRepository {
	return &productRepository{
		collection: db.Collection("products"),
	}
}

func (r *productRepository) CheckAndReserveProduct(ctx context.Context, productID string, quantity int) (bool, error) {
	filter := bson.M{"id": productID, "quantity": bson.M{"$gte": quantity}}
	update := bson.M{"$inc": bson.M{"quantity": -quantity, "reserved": quantity}}
	res := r.collection.FindOneAndUpdate(ctx, filter, update)
	if res.Err() != nil {
		if res.Err() == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, res.Err()
	}
	return true, nil
}

func (r *productRepository) ReleaseReservedProduct(ctx context.Context, productID string, quantity int) error {
	filter := bson.M{"id": productID}
	update := bson.M{"$inc": bson.M{"quantity": quantity, "reserved": -quantity}}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *productRepository) SeedProduct(ctx context.Context, product Product) error {
	filter := bson.M{"id": product.ID}
	update := bson.M{"$setOnInsert": product}
	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *productRepository) GetProductById(ctx context.Context, productID string) (*Product, error) {
	var product Product
	err := r.collection.FindOne(ctx, bson.M{"id": productID}).Decode(&product)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Product not found
		}
		return nil, err
	}
	return &product, nil
}

func (r *productRepository) UpdateProductQuantity(ctx context.Context, productID string, quantity int) error {
	filter := bson.M{"id": productID}
	update := bson.M{"$set": bson.M{"quantity": quantity}}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// GetLowStockProducts returns products with stock below the threshold
func (r *productRepository) GetLowStockProducts(ctx context.Context, threshold int) ([]Product, error) {
	filter := bson.M{"quantity": bson.M{"$lt": threshold}}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var products []Product
	for cursor.Next(ctx) {
		var product Product
		if err := cursor.Decode(&product); err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, nil
}

// AddProduct adds a new product to the inventory
func (r *productRepository) AddProduct(ctx context.Context, product Product) error {
	_, err := r.collection.InsertOne(ctx, product)
	return err
}

// GetAllProducts retrieves all products in the inventory
func (r *productRepository) GetAllProducts(ctx context.Context) ([]Product, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var products []Product
	for cursor.Next(ctx) {
		var product Product
		if err := cursor.Decode(&product); err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, nil
}
