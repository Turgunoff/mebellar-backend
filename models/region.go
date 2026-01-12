package models

import "time"

// Region - O'zbekiston viloyatlari modeli
// @Description Region ma'lumotlari
type Region struct {
	ID        int       `json:"id" example:"1"`
	Name      string    `json:"name" example:"Toshkent sh."`
	Code      string    `json:"code,omitempty" example:"UZ-TK"`
	IsActive  bool      `json:"is_active" example:"true"`
	Ordering  int       `json:"ordering" example:"1"`
	CreatedAt time.Time `json:"created_at,omitempty"`
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
