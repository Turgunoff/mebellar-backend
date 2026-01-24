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
		*s = make(StringMap)
		return nil
	}

	var bytes []byte

	// Handle both []byte and string types (PostgreSQL can return either)
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("type assertion to []byte or string failed for StringMap")
	}

	// Handle empty JSON
	if len(bytes) == 0 || string(bytes) == "{}" {
		*s = make(StringMap)
		return nil
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
// DELIVERY SETTINGS - Regional delivery pricing (New Format)
// ============================================

// HomeDeliverySettings - do'konning o'z viloyati uchun yetkazib berish sozlamalari
// @Description Home region delivery settings
type HomeDeliverySettings struct {
	Price        float64 `json:"price" example:"50000"`
	IsFree       bool    `json:"is_free" example:"true"`
	DeliveryDays string  `json:"delivery_days" example:"1 kun"`
}

// RegionalPriceGroup - bir nechta viloyatlar uchun bitta narx guruhi
// @Description Regional price group for multiple regions
type RegionalPriceGroup struct {
	RegionIDs    []string `json:"ids" example:"1,2,3"`
	Price        float64  `json:"price" example:"60000"`
	Name         string   `json:"name,omitempty" example:"Farg'ona vodiysi"`
	DeliveryDays string   `json:"delivery_days" example:"3-5 kun"`
}

// DeliverySettings - yetkazib berish sozlamalari (yangi format)
// @Description Delivery settings with home region and regional groups
// JSON format:
// {
//   "has_installation": true,
//   "installation_price": 200000,
//   "home_region_price": 0,
//   "is_home_region_free": true,
//   "home_delivery_days": "1 kun",
//   "regional_prices": [
//     {"ids": ["1", "2"], "price": 50000, "delivery_days": "3-5 kun"},
//     {"ids": ["3"], "price": 60000, "delivery_days": "5-7 kun"}
//   ]
// }
type DeliverySettings struct {
	HasInstallation    bool                 `json:"has_installation" example:"true"`
	InstallationPrice  float64              `json:"installation_price" example:"200000"`
	HomeRegionPrice    float64              `json:"home_region_price" example:"0"`
	IsHomeRegionFree   bool                 `json:"is_home_region_free" example:"true"`
	HomeDeliveryDays   string               `json:"home_delivery_days" example:"1 kun"`
	RegionalPrices     []RegionalPriceGroup `json:"regional_prices,omitempty"`
}

// ============================================
// LEGACY DELIVERY SETTINGS - For backward compatibility
// ============================================

// LegacyRegionSettings - eski format uchun (backward compatibility)
// @Description Legacy regional delivery settings (deprecated)
type LegacyRegionSettings struct {
	RegionID          string  `json:"region_id,omitempty" example:"tashkent_city"`
	RegionName        string  `json:"region_name,omitempty" example:"Toshkent sh."`
	DeliveryPrice     float64 `json:"delivery_price" example:"50000"`
	DeliveryDays      string  `json:"delivery_days" example:"1-2"`
	HasInstallation   bool    `json:"has_installation" example:"true"`
	InstallationPrice float64 `json:"installation_price" example:"100000"`
	Comment           string  `json:"comment,omitempty" example:"Shahar ichida bepul"`
}

// LegacyDeliverySettings - eski format (backward compatibility)
// @Description Legacy delivery settings (deprecated)
type LegacyDeliverySettings struct {
	Default   LegacyRegionSettings   `json:"default"`
	Overrides []LegacyRegionSettings `json:"overrides,omitempty"`
}

// Value - database ga yozish uchun
func (d DeliverySettings) Value() (driver.Value, error) {
	return json.Marshal(d)
}

// Scan - database dan o'qish uchun (supports both old and new formats)
func (d *DeliverySettings) Scan(value interface{}) error {
	if value == nil {
		*d = DeliverySettings{
			IsHomeRegionFree: true,
			HomeDeliveryDays: "1-3 kun",
			RegionalPrices:   []RegionalPriceGroup{},
		}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for DeliverySettings")
	}

	// First, try to parse as new format
	var newFormat DeliverySettings
	if err := json.Unmarshal(bytes, &newFormat); err == nil {
		// Check if it's actually new format (has regional_prices or is_home_region_free key)
		var rawMap map[string]interface{}
		json.Unmarshal(bytes, &rawMap)

		if _, hasNewKey := rawMap["regional_prices"]; hasNewKey {
			*d = newFormat
			return nil
		}
		if _, hasNewKey := rawMap["is_home_region_free"]; hasNewKey {
			*d = newFormat
			return nil
		}
		if _, hasNewKey := rawMap["has_installation"]; hasNewKey {
			// Could be new format without regional_prices
			if _, hasDefault := rawMap["default"]; !hasDefault {
				*d = newFormat
				return nil
			}
		}
	}

	// Try to parse as old format and convert
	var oldFormat LegacyDeliverySettings
	if err := json.Unmarshal(bytes, &oldFormat); err == nil {
		// Convert old format to new format
		*d = convertLegacyToNew(oldFormat)
		return nil
	}

	// Fallback: just unmarshal directly
	return json.Unmarshal(bytes, d)
}

// convertLegacyToNew - eski formatni yangi formatga o'girish
func convertLegacyToNew(old LegacyDeliverySettings) DeliverySettings {
	homePrice := old.Default.DeliveryPrice
	isFree := homePrice <= 0

	// Convert overrides to regional price groups
	regionalPrices := make([]RegionalPriceGroup, 0, len(old.Overrides))
	for _, override := range old.Overrides {
		if override.RegionID != "" && override.DeliveryPrice > 0 {
			regionalPrices = append(regionalPrices, RegionalPriceGroup{
				RegionIDs:    []string{override.RegionID},
				Price:        override.DeliveryPrice,
				Name:         override.RegionName,
				DeliveryDays: override.DeliveryDays,
			})
		}
	}

	return DeliverySettings{
		HasInstallation:   old.Default.HasInstallation,
		InstallationPrice: old.Default.InstallationPrice,
		HomeRegionPrice:   homePrice,
		IsHomeRegionFree:  isFree,
		HomeDeliveryDays:  old.Default.DeliveryDays,
		RegionalPrices:    regionalPrices,
	}
}

// GetRegionPrice - ma'lum hudud uchun narxni olish
func (d *DeliverySettings) GetRegionPrice(regionID string) (float64, string, bool) {
	for _, group := range d.RegionalPrices {
		for _, id := range group.RegionIDs {
			if id == regionID {
				return group.Price, group.DeliveryDays, true
			}
		}
	}
	// Return home region settings as default
	if d.IsHomeRegionFree {
		return 0, d.HomeDeliveryDays, true
	}
	return d.HomeRegionPrice, d.HomeDeliveryDays, true
}

// IsRegionAvailable - viloyatga yetkazib berish mavjudmi
func (d *DeliverySettings) IsRegionAvailable(regionID string, homeRegionID string) bool {
	// Home region is always available
	if regionID == homeRegionID {
		return true
	}
	// Check if in any regional group
	for _, group := range d.RegionalPrices {
		for _, id := range group.RegionIDs {
			if id == regionID {
				return true
			}
		}
	}
	return false
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

// CategoryProductsGroup - kategoriya va uning mahsulotlari guruhi
type CategoryProductsGroup struct {
	Category  Category  `json:"category"`
	Products  []Product `json:"products"`
	Total     int       `json:"total"`     // Jami mahsulotlar soni (preview emas)
	HasMore   bool      `json:"has_more"`  // Yana mahsulotlar bormi
}

// GroupedProductsResponse - guruhlangan mahsulotlar javob modeli
type GroupedProductsResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Groups  []CategoryProductsGroup `json:"groups"`
	Count   int                    `json:"count"` // Jami guruhlar soni
}

// Category - kategoriya modeli category.go faylida aniqlangan
