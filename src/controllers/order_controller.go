package controllers

import (
	"go-order-eda/src/controllers/models"
	"go-order-eda/src/services/order/domain"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type OrderController struct {
	domain.OrderService
}

func NewOrderController(orderService domain.OrderService) *OrderController {
	return &OrderController{
		OrderService: orderService,
	}
}
func (c *OrderController) Route(app *fiber.App) {
	api := app.Group("/api/v1/orders")
	api.Post("/create-order", c.CreateOrder)
	api.Post("/replay-failed-events", c.ReplayFailedEvents)
}

// ReplayFailedEvents godoc
// @Summary      Replay failed order events
// @Description  Replays failed order events that have not been successfully published
// @Tags         orders
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/v1/orders/replay-failed-events [post]
func (c *OrderController) ReplayFailedEvents(ctx *fiber.Ctx) error {
	err := c.OrderService.ReplayFailedEvents(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"status": "Replay complete"})
}

// CreateOrder godoc
// @Summary      Create a new order
// @Description  Creates a new order and returns the status
// @Tags         orders
// @Accept       json
// @Produce      json
// @Param        order  body  models.OrderRequest  true  "Order payload"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/v1/orders/create-order [post]
func (c *OrderController) CreateOrder(ctx *fiber.Ctx) error {
	var order domain.Order
	var OrderRequest models.OrderRequest
	if err := ctx.BodyParser(&OrderRequest); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}
	order = domain.Order{
		ID:     uuid.New().String(),
		Amount: OrderRequest.Amount,
		Product: domain.Product{
			ID:       OrderRequest.Product.ID,
			Name:     OrderRequest.Product.Name,
			Quantity: OrderRequest.Product.Quantity,
		},
		Status: "Pending",
	}
	orderID, err := c.OrderService.CreateOrder(ctx.Context(), order)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "Order created successfully", "order_id": orderID})
}
