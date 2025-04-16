package model

import "time"

type Order struct {
	ID         string       `json:"id"`
	UserID     string       `json:"userId"`
	Items      []*OrderItem `json:"items"`
	TotalPrice float64      `json:"totalPrice"`
	Status     string       `json:"status"`
	CreatedAt  time.Time    `json:"createdAt"`
	UpdatedAt  time.Time    `json:"updatedAt"`
}
