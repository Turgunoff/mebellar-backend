package models

import (
	"strings"
	"time"
)

// ============================================
// SHOP MODEL
// ============================================

// Shop - do'kon modeli (Multi-language support)
// @Description Do'kon ma'lumotlari
type Shop struct {
	// Asosiy identifikatorlar
	ID       string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SellerID string  `json:"seller_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	
	// Multi-language maydonlar
	Name        StringMap    `json:"name" swaggertype:"object"`
	Description StringMap    `json:"description,omitempty" swaggertype:"object"`
	Address     StringMap    `json:"address,omitempty" swaggertype:"object"`
	
	// SEO va identifikatsiya
	Slug string `json:"slug" example:"mebel-house-tashkent"`
	
	// Media
	LogoURL   string `json:"logo_url,omitempty" example:"https://example.com/logo.jpg"`
	BannerURL string `json:"banner_url,omitempty" example:"https://example.com/banner.jpg"`
	
	// Aloqa va joylashuv
	Phone     string   `json:"phone,omitempty" example:"+998901234567"`
	Latitude  *float64 `json:"latitude,omitempty" example:"41.311081"`
	Longitude *float64 `json:"longitude,omitempty" example:"69.240562"`
	RegionID  *int     `json:"region_id,omitempty" example:"1"`
	
	// Ish vaqtlari
	WorkingHours WorkingHours `json:"working_hours,omitempty" swaggertype:"object"`
	
	// Status va reyting
	IsActive   bool    `json:"is_active" example:"true"`
	IsVerified bool    `json:"is_verified" example:"false"`
	IsMain     bool    `json:"is_main" example:"false"`
	Rating     float64 `json:"rating" example:"4.5"`
	
	// Vaqt belgilari
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetName - Helper function to get name in a specific language, fallback to 'uz'
func (s *Shop) GetName(lang string) string {
	if s.Name == nil {
		return ""
	}
	if name, ok := s.Name[lang]; ok && name != "" {
		return name
	}
	// Fallback to 'uz'
	if name, ok := s.Name["uz"]; ok {
		return name
	}
	// If 'uz' doesn't exist, return first available
	for _, name := range s.Name {
		return name
	}
	return ""
}

// GetDescription - Helper function to get description in a specific language
func (s *Shop) GetDescription(lang string) string {
	if s.Description == nil {
		return ""
	}
	if desc, ok := s.Description[lang]; ok && desc != "" {
		return desc
	}
	if desc, ok := s.Description["uz"]; ok {
		return desc
	}
	for _, desc := range s.Description {
		return desc
	}
	return ""
}

// GetAddress - Helper function to get address in a specific language
func (s *Shop) GetAddress(lang string) string {
	if s.Address == nil {
		return ""
	}
	if addr, ok := s.Address[lang]; ok && addr != "" {
		return addr
	}
	if addr, ok := s.Address["uz"]; ok {
		return addr
	}
	for _, addr := range s.Address {
		return addr
	}
	return ""
}

// GenerateSlugFromName - Do'kon nomi asosida slug yaratish (English name dan)
func GenerateSlugFromName(nameMap StringMap) string {
	// English name dan slug yaratish
	englishName := ""
	if nameMap != nil {
		if en, ok := nameMap["en"]; ok && en != "" {
			englishName = en
		} else if uz, ok := nameMap["uz"]; ok && uz != "" {
			englishName = uz
		} else {
			// First available
			for _, name := range nameMap {
				englishName = name
				break
			}
		}
	}
	
	if englishName == "" {
		return ""
	}
	
	// Slug yaratish
	slug := ""
	for _, r := range englishName {
		switch {
		case r >= 'a' && r <= 'z':
			slug += string(r)
		case r >= 'A' && r <= 'Z':
			slug += string(r + 32) // lowercase
		case r >= '0' && r <= '9':
			slug += string(r)
		case r == ' ' || r == '-' || r == '_':
			if len(slug) > 0 && slug[len(slug)-1] != '-' {
				slug += "-"
			}
		}
	}
	
	// Oxiridagi tire ni olib tashlash
	slug = strings.Trim(slug, "-")
	
	return slug
}

// ============================================
// REQUEST/RESPONSE MODELS
// ============================================

// CreateShopRequest - do'kon yaratish so'rovi
type CreateShopRequest struct {
	Name        StringMap     `json:"name" binding:"required"` // {"uz": "...", "ru": "...", "en": "..."}
	Description *StringMap    `json:"description,omitempty"`
	Address     *StringMap    `json:"address,omitempty"`
	Phone       *string      `json:"phone,omitempty"`
	RegionID    *int         `json:"region_id,omitempty"`
	Latitude    *float64      `json:"latitude,omitempty"`
	Longitude   *float64      `json:"longitude,omitempty"`
	WorkingHours *WorkingHours `json:"working_hours,omitempty"`
	IsMain      *bool         `json:"is_main,omitempty"`
	IsActive    *bool         `json:"is_active,omitempty"`
}

// UpdateShopRequest - do'kon yangilash so'rovi
type UpdateShopRequest struct {
	Name        *StringMap    `json:"name,omitempty"`
	Description *StringMap    `json:"description,omitempty"`
	Address     *StringMap    `json:"address,omitempty"`
	Phone       *string       `json:"phone,omitempty"`
	RegionID    *int          `json:"region_id,omitempty"`
	Latitude    *float64      `json:"latitude,omitempty"`
	Longitude   *float64      `json:"longitude,omitempty"`
	LogoURL     *string       `json:"logo_url,omitempty"`
	BannerURL   *string       `json:"banner_url,omitempty"`
	WorkingHours *WorkingHours `json:"working_hours,omitempty"`
	IsMain      *bool         `json:"is_main,omitempty"`
	IsActive    *bool         `json:"is_active,omitempty"`
}

// ShopResponse - bitta do'kon javobi
type ShopResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Shop    *Shop  `json:"shop,omitempty"`
}

// ShopsResponse - do'konlar ro'yxati javobi
type ShopsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Shops   []Shop `json:"shops"`
	Count   int    `json:"count"`
	Page    int    `json:"page,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// SellerWithShops - Seller ma'lumotlari bilan do'konlar
type SellerWithShops struct {
	SellerProfile *SellerProfile `json:"seller_profile"`
	Shops         []Shop         `json:"shops"`
	ShopsCount    int            `json:"shops_count"`
}
