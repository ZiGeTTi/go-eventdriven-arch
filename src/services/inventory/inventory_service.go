package inventory

import (
	"context"
	"go-order-eda/src/infrastructure/log"
)

type inventoryService struct {
	logger            log.Logger
	productRepository ProductRepository
}

type InventoryService interface {
	// Business logic methods for inventory management
	GetProductStock(ctx context.Context, productID string) (*Product, error)
	UpdateProductQuantity(ctx context.Context, productID string, quantity int) error
	GetLowStockProducts(ctx context.Context, threshold int) ([]Product, error)
	AddProduct(ctx context.Context, product Product) error
	GetAllProducts(ctx context.Context) ([]Product, error)
	ReserveProduct(ctx context.Context, productID string, quantity int) (bool, error)
	ReleaseReservedProduct(ctx context.Context, productID string, quantity int) error
}

func NewInventoryService(logger log.Logger, productRepo ProductRepository) InventoryService {
	return &inventoryService{
		logger:            logger,
		productRepository: productRepo,
	}
}

// GetProductStock retrieves current stock information for a product
func (s *inventoryService) GetProductStock(ctx context.Context, productID string) (*Product, error) {
	return s.productRepository.GetProductById(ctx, productID)
}

// UpdateProductQuantity updates the available quantity of a product
func (s *inventoryService) UpdateProductQuantity(ctx context.Context, productID string, quantity int) error {
	return s.productRepository.UpdateProductQuantity(ctx, productID, quantity)
}

// GetLowStockProducts returns products with stock below the threshold
func (s *inventoryService) GetLowStockProducts(ctx context.Context, threshold int) ([]Product, error) {
	return s.productRepository.GetLowStockProducts(ctx, threshold)
}

// AddProduct adds a new product to the inventory
func (s *inventoryService) AddProduct(ctx context.Context, product Product) error {
	return s.productRepository.AddProduct(ctx, product)
}

// GetAllProducts retrieves all products in the inventory
func (s *inventoryService) GetAllProducts(ctx context.Context) ([]Product, error) {
	return s.productRepository.GetAllProducts(ctx)
}

// ReserveProduct reserves a quantity of a product for an order
func (s *inventoryService) ReserveProduct(ctx context.Context, productID string, quantity int) (bool, error) {
	return s.productRepository.CheckAndReserveProduct(ctx, productID, quantity)
}

// ReleaseReservedProduct releases reserved quantity back to available stock
func (s *inventoryService) ReleaseReservedProduct(ctx context.Context, productID string, quantity int) error {
	return s.productRepository.ReleaseReservedProduct(ctx, productID, quantity)
}
