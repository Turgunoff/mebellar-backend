package models

import (
	"github.com/lib/pq" // Arraylar uchun (ranglar)
)

// Product - bazadagi jadval bilan bir xil bo'lishi kerak
type Product struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Price       float64        `json:"price"`
	CategoryID  int            `json:"category_id"`
	Colors      pq.StringArray `json:"colors"`    // Postgres array
	ImageURL    string         `json:"image_url"` // MinIO link
}
