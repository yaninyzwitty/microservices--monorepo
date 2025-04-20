package model

import "time"

type Product struct {
	ID          string    `json:"id"`
	CategoryID  string    `json:"categoryId"`
	Category    *Category `json:"category"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Stock       int32     `json:"stock"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
