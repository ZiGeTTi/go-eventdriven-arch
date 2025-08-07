package domain

import "time"

type Order struct {
	ID     string
	Amount float64
	Status string
	Product
	CreatedAt time.Time
}

type Product struct {
	ID       string
	Name     string
	Quantity int
}

func NewOrder(id string, amount float64) *Order {
	return &Order{
		ID:     id,
		Amount: amount,
		Status: "Pending",
		Product: Product{
			ID:   "1",
			Name: "Sample Product",
		},
		CreatedAt: time.Now().Local(),
	}
}
