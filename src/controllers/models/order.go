package models

type OrderRequest struct {
	Amount  float64 `json:"amount"`
	Product struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Quantity int    `json:"quantity"`
	} `json:"product"`
}
