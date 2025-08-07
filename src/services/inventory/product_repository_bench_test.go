package inventory

import (
	"testing"
)

// Simple benchmark tests without external dependencies
func BenchmarkProductStruct(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		product := Product{
			ID:       "test-product",
			Name:     "Test Product",
			Quantity: 100,
			Reserved: 0,
		}
		_ = product.ID
		_ = product.Quantity
	}
}

func BenchmarkReservationLogic(b *testing.B) {
	initialQuantity := 100
	reserveAmount := 5

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate the logic that would happen in MongoDB
		newQuantity := initialQuantity - reserveAmount
		newReserved := 0 + reserveAmount

		// Prevent compiler optimization
		_ = newQuantity
		_ = newReserved
	}
}
