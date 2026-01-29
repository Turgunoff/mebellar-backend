package models

import "time"

// Region - O'zbekiston viloyatlari modeli
// @Description Region ma'lumotlari
type Region struct {
	ID        int       `json:"id" example:"1"`
	Name      string    `json:"name" example:"Toshkent sh."`               // Legacy: plain string name
	NameJSONB StringMap `json:"name_jsonb,omitempty" swaggertype:"object"` // Multi-language: {"uz": "...", "ru": "...", "en": "..."}
	Code      string    `json:"code,omitempty" example:"UZ-TK"`
	IsActive  bool      `json:"is_active" example:"true"`
	Ordering  int       `json:"ordering" example:"1"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// GetName - Helper function to get name in a specific language, fallback to 'uz' then legacy name
func (r *Region) GetName(lang string) string {
	if r.NameJSONB != nil {
		if name, ok := r.NameJSONB[lang]; ok && name != "" {
			return name
		}
		if name, ok := r.NameJSONB["uz"]; ok && name != "" {
			return name
		}
	}
	return r.Name
}

// RegionResponse - bitta region javobi
type RegionResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message,omitempty"`
	Region  *Region `json:"region,omitempty"`
}

// RegionsResponse - regionlar ro'yxati javobi
type RegionsResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message,omitempty"`
	Regions []Region `json:"regions"`
	Count   int      `json:"count"`
}

// CreateRegionRequest - region yaratish so'rovi
type CreateRegionRequest struct {
	Name     StringMap `json:"name" binding:"required"` // {"uz": "...", "ru": "...", "en": "..."}
	Code     string    `json:"code" binding:"required"` // ISO 3166-2 code (e.g., UZ-TK)
	Ordering int       `json:"ordering"`
	IsActive *bool     `json:"is_active,omitempty"`
}

// UpdateRegionRequest - region yangilash so'rovi
type UpdateRegionRequest struct {
	Name     *StringMap `json:"name,omitempty"`
	Code     *string    `json:"code,omitempty"`
	Ordering *int       `json:"ordering,omitempty"`
	IsActive *bool      `json:"is_active,omitempty"`
}
