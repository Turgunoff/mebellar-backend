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

// StringMap - PostgreSQL JSONB uchun map[string]string type (name, description uchun)
type StringMap map[string]string

// Value - database ga yozish uchun
func (s StringMap) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan - database dan o'qish uchun
func (s *StringMap) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for StringMap")
	}

	return json.Unmarshal(bytes, s)
}

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
// DELIVERY SETTINGS - Regional delivery pricing
// ============================================

// RegionSettings - bir hudud uchun yetkazib berish sozlamalari
// @Description Regional delivery settings
type RegionSettings struct {
	RegionID          string  `json:"region_id,omitempty" example:"tashkent_city"`
	RegionName        string  `json:"region_name,omitempty" example:"Toshkent sh."`
	DeliveryPrice     float64 `json:"delivery_price" example:"50000"`
	DeliveryDays      string  `json:"delivery_days" example:"1-2"`
	HasInstallation   bool    `json:"has_installation" example:"true"`
	InstallationPrice float64 `json:"installation_price" example:"100000"`
	Comment           string  `json:"comment,omitempty" example:"Shahar ichida bepul"`
}

// DeliverySettings - yetkazib berish sozlamalari (default + overrides)
// @Description Delivery settings with default and regional overrides
type DeliverySettings struct {
	Default   RegionSettings   `json:"default"`
	Overrides []RegionSettings `json:"overrides,omitempty"`
}

// Value - database ga yozish uchun
func (d DeliverySettings) Value() (driver.Value, error) {
	return json.Marshal(d)
}

// Scan - database dan o'qish uchun
func (d *DeliverySettings) Scan(value interface{}) error {
	if value == nil {
		*d = DeliverySettings{
			Default: RegionSettings{
				DeliveryDays: "3-5",
			},
			Overrides: []RegionSettings{},
		}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for DeliverySettings")
	}

	return json.Unmarshal(bytes, d)
}

// GetRegionPrice - ma'lum hudud uchun narxni olish
func (d *DeliverySettings) GetRegionPrice(regionID string) RegionSettings {
	for _, override := range d.Overrides {
		if override.RegionID == regionID {
			return override
		}
	}
	return d.Default
}

// ============================================
// PRODUCT MODEL
// ============================================

// Product - mahsulot modeli (MVP uchun moslashuvchan arxitektura)
// @Description Mahsulot ma'lumotlari
type Product struct {
	ID               string           `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ShopID           string           `json:"shop_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	CategoryID       *string          `json:"category_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name             StringMap        `json:"name" swaggertype:"object"`
	Description      StringMap        `json:"description" swaggertype:"object"`
	Price            float64          `json:"price" example:"5500000"`
	DiscountPrice    *float64         `json:"discount_price,omitempty" example:"4400000"`
	Images           pq.StringArray   `json:"images" swaggertype:"array,string" example:"https://example.com/img1.jpg,https://example.com/img2.jpg"`
	Specs            JSONB            `json:"specs,omitempty" swaggertype:"object"`
	Variants         JSONBArray       `json:"variants,omitempty" swaggertype:"array,object"`
	DeliverySettings DeliverySettings `json:"delivery_settings,omitempty"`
	Rating           float64          `json:"rating" example:"4.8"`
	ViewCount        int              `json:"view_count" example:"150"`
	SoldCount        int              `json:"sold_count" example:"12"`
	IsNew            bool             `json:"is_new" example:"true"`
	IsPopular        bool             `json:"is_popular" example:"true"`
	IsActive         bool             `json:"is_active" example:"true"`
	CreatedAt        time.Time        `json:"created_at"`
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

// Category - kategoriya modeli category.go faylida aniqlangan
