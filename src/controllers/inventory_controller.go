package controllers

import (
	"strconv"

	"go-order-eda/src/services/inventory"

	"github.com/gofiber/fiber/v2"
)

type InventoryController struct {
	inventoryService inventory.InventoryService
}

func NewInventoryController(inventoryService inventory.InventoryService) *InventoryController {
	return &InventoryController{
		inventoryService: inventoryService,
	}
}

func (c *InventoryController) Route(app *fiber.App) {
	api := app.Group("/api/v1/inventory")
	api.Get("/products", c.GetAllProducts)
	api.Get("/products/:id", c.GetProduct)
	api.Get("/products/low-stock/:threshold", c.GetLowStockProducts)
	api.Post("/products/:id/reserve/:quantity", c.ReserveProduct)
	api.Post("/products/:id/release/:quantity", c.ReleaseProduct)
	api.Put("/products/:id/quantity/:quantity", c.UpdateQuantity)
}

// GetAllProducts godoc
// @Summary      Get all products
// @Description  Retrieves all products in inventory
// @Tags         inventory
// @Produce      json
// @Success      200  {array}  inventory.Product
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/v1/inventory/products [get]
func (c *InventoryController) GetAllProducts(ctx *fiber.Ctx) error {
	products, err := c.inventoryService.GetAllProducts(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(products)
}

// GetProduct godoc
// @Summary      Get product by ID
// @Description  Retrieves a specific product by ID
// @Tags         inventory
// @Produce      json
// @Param        id   path      string  true  "Product ID"
// @Success      200  {object}  inventory.Product
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/v1/inventory/products/{id} [get]
func (c *InventoryController) GetProduct(ctx *fiber.Ctx) error {
	productID := ctx.Params("id")
	product, err := c.inventoryService.GetProductStock(ctx.Context(), productID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if product == nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Product not found"})
	}
	return ctx.JSON(product)
}

// GetLowStockProducts godoc
// @Summary      Get low stock products
// @Description  Retrieves products with stock below threshold
// @Tags         inventory
// @Produce      json
// @Param        threshold   path      int  true  "Stock threshold"
// @Success      200  {array}  inventory.Product
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/v1/inventory/products/low-stock/{threshold} [get]
func (c *InventoryController) GetLowStockProducts(ctx *fiber.Ctx) error {
	thresholdStr := ctx.Params("threshold")
	threshold, err := strconv.Atoi(thresholdStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid threshold"})
	}

	products, err := c.inventoryService.GetLowStockProducts(ctx.Context(), threshold)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(products)
}

// ReserveProduct godoc
// @Summary      Reserve product quantity
// @Description  Reserves a quantity of a product
// @Tags         inventory
// @Produce      json
// @Param        id        path      string  true  "Product ID"
// @Param        quantity  path      int     true  "Quantity to reserve"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/v1/inventory/products/{id}/reserve/{quantity} [post]
func (c *InventoryController) ReserveProduct(ctx *fiber.Ctx) error {
	productID := ctx.Params("id")
	quantityStr := ctx.Params("quantity")
	quantity, err := strconv.Atoi(quantityStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid quantity"})
	}

	success, err := c.inventoryService.ReserveProduct(ctx.Context(), productID, quantity)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if !success {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Insufficient stock or product not found"})
	}

	return ctx.JSON(fiber.Map{"message": "Product reserved successfully"})
}

// ReleaseProduct godoc
// @Summary      Release reserved product quantity
// @Description  Releases reserved quantity back to available stock
// @Tags         inventory
// @Produce      json
// @Param        id        path      string  true  "Product ID"
// @Param        quantity  path      int     true  "Quantity to release"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/v1/inventory/products/{id}/release/{quantity} [post]
func (c *InventoryController) ReleaseProduct(ctx *fiber.Ctx) error {
	productID := ctx.Params("id")
	quantityStr := ctx.Params("quantity")
	quantity, err := strconv.Atoi(quantityStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid quantity"})
	}

	err = c.inventoryService.ReleaseReservedProduct(ctx.Context(), productID, quantity)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Reserved product released successfully"})
}

// UpdateQuantity godoc
// @Summary      Update product quantity
// @Description  Updates the available quantity of a product
// @Tags         inventory
// @Produce      json
// @Param        id        path      string  true  "Product ID"
// @Param        quantity  path      int     true  "New quantity"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/v1/inventory/products/{id}/quantity/{quantity} [put]
func (c *InventoryController) UpdateQuantity(ctx *fiber.Ctx) error {
	productID := ctx.Params("id")
	quantityStr := ctx.Params("quantity")
	quantity, err := strconv.Atoi(quantityStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid quantity"})
	}

	err = c.inventoryService.UpdateProductQuantity(ctx.Context(), productID, quantity)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Product quantity updated successfully"})
}
