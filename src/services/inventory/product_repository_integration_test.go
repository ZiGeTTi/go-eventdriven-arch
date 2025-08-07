package inventory

import (
	"context"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Integration test that requires a real MongoDB connection
// To run: go test -tags=integration
func TestProductRepository_QuantityDecreases_Integration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if MongoDB URL is provided
	mongoURL := os.Getenv("MONGODB_URL")
	if mongoURL == "" {
		mongoURL = "mongodb://root:example@localhost:27017" // Default for local testing
	}

	// Connect to MongoDB
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURL))
	if err != nil {
		t.Skipf("Cannot connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	// Use a test database
	db := client.Database("test_inventory")
	repo := NewProductRepository(db)
	ctx := context.Background()

	t.Run("quantity decreases and reserved increases on successful reservation", func(t *testing.T) {
		// Arrange - Clean up and create test product
		productID := "test-product-1"
		testProduct := Product{
			ID:       productID,
			Name:     "Test Product",
			Quantity: 10,
			Reserved: 0,
		}

		// Clean up any existing test data
		db.Collection("products").Drop(ctx)

		// Add test product
		err := repo.AddProduct(ctx, testProduct)
		if err != nil {
			t.Fatalf("Failed to add test product: %v", err)
		}

		// Get initial state
		initialProduct, err := repo.GetProductById(ctx, productID)
		if err != nil {
			t.Fatalf("Failed to get initial product: %v", err)
		}
		if initialProduct == nil {
			t.Fatal("Product not found after adding")
		}

		initialQuantity := initialProduct.Quantity
		initialReserved := initialProduct.Reserved
		reserveAmount := 3

		// Act - Reserve product
		success, err := repo.CheckAndReserveProduct(ctx, productID, reserveAmount)

		// Assert reservation succeeded
		if err != nil {
			t.Fatalf("Reservation failed with error: %v", err)
		}
		if !success {
			t.Fatal("Reservation should have succeeded")
		}

		// Get updated state
		updatedProduct, err := repo.GetProductById(ctx, productID)
		if err != nil {
			t.Fatalf("Failed to get updated product: %v", err)
		}
		if updatedProduct == nil {
			t.Fatal("Product not found after reservation")
		}

		// Verify quantity decreased
		expectedQuantity := initialQuantity - reserveAmount
		if updatedProduct.Quantity != expectedQuantity {
			t.Errorf("Expected quantity to be %d after reservation, got %d",
				expectedQuantity, updatedProduct.Quantity)
		}

		// Verify reserved increased
		expectedReserved := initialReserved + reserveAmount
		if updatedProduct.Reserved != expectedReserved {
			t.Errorf("Expected reserved to be %d after reservation, got %d",
				expectedReserved, updatedProduct.Reserved)
		}

		t.Logf("✅ Quantity correctly decreased from %d to %d", initialQuantity, updatedProduct.Quantity)
		t.Logf("✅ Reserved correctly increased from %d to %d", initialReserved, updatedProduct.Reserved)
	})

	t.Run("reservation fails when insufficient quantity", func(t *testing.T) {
		// Arrange - Create product with low quantity
		productID := "test-product-2"
		testProduct := Product{
			ID:       productID,
			Name:     "Low Stock Product",
			Quantity: 2,
			Reserved: 0,
		}

		// Add test product
		err := repo.AddProduct(ctx, testProduct)
		if err != nil {
			t.Fatalf("Failed to add test product: %v", err)
		}

		reserveAmount := 5 // More than available

		// Act - Try to reserve more than available
		success, err := repo.CheckAndReserveProduct(ctx, productID, reserveAmount)

		// Assert reservation failed
		if err != nil {
			t.Fatalf("Unexpected error during reservation: %v", err)
		}
		if success {
			t.Fatal("Reservation should have failed due to insufficient quantity")
		}

		// Verify quantity unchanged
		unchangedProduct, err := repo.GetProductById(ctx, productID)
		if err != nil {
			t.Fatalf("Failed to get product after failed reservation: %v", err)
		}
		if unchangedProduct.Quantity != testProduct.Quantity {
			t.Errorf("Quantity should remain %d after failed reservation, got %d",
				testProduct.Quantity, unchangedProduct.Quantity)
		}
		if unchangedProduct.Reserved != testProduct.Reserved {
			t.Errorf("Reserved should remain %d after failed reservation, got %d",
				testProduct.Reserved, unchangedProduct.Reserved)
		}

		t.Logf("✅ Reservation correctly failed when requesting %d from %d available",
			reserveAmount, testProduct.Quantity)
	})

	t.Run("complete reservation and release cycle", func(t *testing.T) {
		// Arrange
		productID := "test-product-3"
		testProduct := Product{
			ID:       productID,
			Name:     "Cycle Test Product",
			Quantity: 15,
			Reserved: 0,
		}

		// Add test product
		err := repo.AddProduct(ctx, testProduct)
		if err != nil {
			t.Fatalf("Failed to add test product: %v", err)
		}

		reserveAmount := 4

		// Act 1 - Reserve
		success, err := repo.CheckAndReserveProduct(ctx, productID, reserveAmount)
		if err != nil || !success {
			t.Fatalf("Reservation failed: success=%v, err=%v", success, err)
		}

		// Verify after reservation
		afterReserve, err := repo.GetProductById(ctx, productID)
		if err != nil {
			t.Fatalf("Failed to get product after reservation: %v", err)
		}

		expectedQuantityAfterReserve := testProduct.Quantity - reserveAmount
		expectedReservedAfterReserve := testProduct.Reserved + reserveAmount

		if afterReserve.Quantity != expectedQuantityAfterReserve {
			t.Errorf("After reserve: expected quantity %d, got %d",
				expectedQuantityAfterReserve, afterReserve.Quantity)
		}
		if afterReserve.Reserved != expectedReservedAfterReserve {
			t.Errorf("After reserve: expected reserved %d, got %d",
				expectedReservedAfterReserve, afterReserve.Reserved)
		}

		// Act 2 - Release
		err = repo.ReleaseReservedProduct(ctx, productID, reserveAmount)
		if err != nil {
			t.Fatalf("Release failed: %v", err)
		}

		// Verify after release
		afterRelease, err := repo.GetProductById(ctx, productID)
		if err != nil {
			t.Fatalf("Failed to get product after release: %v", err)
		}

		// Should be back to original state
		if afterRelease.Quantity != testProduct.Quantity {
			t.Errorf("After release: expected quantity %d, got %d",
				testProduct.Quantity, afterRelease.Quantity)
		}
		if afterRelease.Reserved != testProduct.Reserved {
			t.Errorf("After release: expected reserved %d, got %d",
				testProduct.Reserved, afterRelease.Reserved)
		}

		t.Logf("✅ Complete cycle: %d → %d (reserve) → %d (release)",
			testProduct.Quantity, afterReserve.Quantity, afterRelease.Quantity)
		t.Logf("✅ Reserved cycle: %d → %d (reserve) → %d (release)",
			testProduct.Reserved, afterReserve.Reserved, afterRelease.Reserved)
	})

	// Cleanup
	db.Collection("products").Drop(ctx)
}
