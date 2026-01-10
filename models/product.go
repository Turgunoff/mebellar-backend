package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/lib/pq"
)

// ============================================
// JSONB TYPE - PostgreSQL JSONB uchun
// ============================================

// JSONB - PostgreSQL JSONB maydoni uchun custom type
type JSONB map[string]interface{}

// Value - database ga yozish uchun
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan - database dan o'qish uchun
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, j)
}

// JSONBArray - JSONB massivi uchun (variants)
type JSONBArray []map[string]interface{}

// Value - database ga yozish uchun
func (j JSONBArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan - database dan o'qish uchun
func (j *JSONBArray) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, j)
}

// ============================================
// PRODUCT MODEL
// ============================================

// Product - mahsulot modeli (MVP uchun moslashuvchan arxitektura)
// @Description Mahsulot ma'lumotlari
type Product struct {
	ID            string         `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CategoryID    *string        `json:"category_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name          string         `json:"name" example:"Premium Divan"`
	Description   string         `json:"description" example:"Zamonaviy dizayndagi divan"`
	Price         float64        `json:"price" example:"5500000"`
	DiscountPrice *float64       `json:"discount_price,omitempty" example:"4400000"`
	Images        pq.StringArray `json:"images" swaggertype:"array,string" example:"https://example.com/img1.jpg,https://example.com/img2.jpg"`
	Specs         JSONB          `json:"specs,omitempty" swaggertype:"object"`
	Variants      JSONBArray     `json:"variants,omitempty" swaggertype:"array,object"`
	Rating        float64        `json:"rating" example:"4.8"`
	IsNew         bool           `json:"is_new" example:"true"`
	IsPopular     bool           `json:"is_popular" example:"true"`
	IsActive      bool           `json:"is_active" example:"true"`
	CreatedAt     time.Time      `json:"created_at"`
}

// DiscountPercent - chegirma foizini hisoblash
func (p *Product) DiscountPercent() int {
	if p.DiscountPrice == nil || *p.DiscountPrice <= 0 || p.Price <= 0 {
		return 0
	}
	discount := ((p.Price - *p.DiscountPrice) / p.Price) * 100
	return int(discount)
}

// HasDiscount - chegirma bormi
func (p *Product) HasDiscount() bool {
	return p.DiscountPrice != nil && *p.DiscountPrice > 0 && *p.DiscountPrice < p.Price
}

// ProductResponse - bitta mahsulot javobi
type ProductResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message,omitempty"`
	Product *Product `json:"product,omitempty"`
}

// ProductsResponse - mahsulotlar javob modeli
type ProductsResponse struct {
	Success  bool      `json:"success"`
	Message  string    `json:"message,omitempty"`
	Products []Product `json:"products"`
	Count    int       `json:"count"`
}

// Category - kategoriya modeli (kelajak uchun)
type Category struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	IconName string  `json:"icon_name"`
	ParentID *string `json:"parent_id,omitempty"`
}
