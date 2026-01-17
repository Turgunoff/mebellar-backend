package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// ============================================
// SELLER PROFILE MODEL
// ============================================

// SocialLinks - ijtimoiy tarmoq havolalari
type SocialLinks struct {
	Instagram string `json:"instagram,omitempty"`
	Telegram  string `json:"telegram,omitempty"`
	Facebook  string `json:"facebook,omitempty"`
	Website   string `json:"website,omitempty"`
	YouTube   string `json:"youtube,omitempty"`
}

// Value - database ga yozish uchun
func (s SocialLinks) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan - database dan o'qish uchun
func (s *SocialLinks) Scan(value interface{}) error {
	if value == nil {
		*s = SocialLinks{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, s)
}

// WorkingHours - ish vaqtlari
type WorkingHours struct {
	Monday    *DaySchedule `json:"monday,omitempty"`
	Tuesday   *DaySchedule `json:"tuesday,omitempty"`
	Wednesday *DaySchedule `json:"wednesday,omitempty"`
	Thursday  *DaySchedule `json:"thursday,omitempty"`
	Friday    *DaySchedule `json:"friday,omitempty"`
	Saturday  *DaySchedule `json:"saturday,omitempty"`
	Sunday    *DaySchedule `json:"sunday,omitempty"`
}

// DaySchedule - kunlik ish jadvali
type DaySchedule struct {
	Open   string `json:"open"`   // "09:00"
	Close  string `json:"close"`  // "18:00"
	Closed bool   `json:"closed"` // Dam olish kuni
}

// Value - database ga yozish uchun
func (w WorkingHours) Value() (driver.Value, error) {
	return json.Marshal(w)
}

// Scan - database dan o'qish uchun
func (w *WorkingHours) Scan(value interface{}) error {
	if value == nil {
		*w = WorkingHours{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, w)
}

// SellerProfile - sotuvchi profili modeli
// @Description Sotuvchi do'koni ma'lumotlari
type SellerProfile struct {
	// Asosiy identifikatorlar
	ID     string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440001"`

	// Biznes ma'lumotlari
	ShopName    string `json:"shop_name" example:"Mebel House"`
	Slug        string `json:"slug" example:"mebel-house"`
	Description string `json:"description,omitempty" example:"Sifatli mebel do'koni"`
	LogoURL     string `json:"logo_url,omitempty" example:"https://example.com/logo.jpg"`
	BannerURL   string `json:"banner_url,omitempty" example:"https://example.com/banner.jpg"`

	// Yuridik va moliyaviy ma'lumotlar (maxfiy)
	LegalName   string `json:"legal_name,omitempty" example:"Mebel House MChJ"`
	TaxID       string `json:"tax_id,omitempty" example:"123456789"`
	BankAccount string `json:"bank_account,omitempty" example:"20208000123456789012"`
	BankName    string `json:"bank_name,omitempty" example:"Kapitalbank"`

	// Aloqa va joylashuv
	SupportPhone string   `json:"support_phone,omitempty" example:"+998901234567"`
	Address      StringMap `json:"address,omitempty" swaggertype:"object" example:"{\"uz\":\"Toshkent, Chilonzor tumani\",\"ru\":\"Ташкент, Чилонзор район\",\"en\":\"Tashkent, Chilonzor district\"}"`
	Latitude     *float64 `json:"latitude,omitempty" example:"41.311081"`
	Longitude    *float64 `json:"longitude,omitempty" example:"69.240562"`

	// JSONB maydonlari
	SocialLinks  SocialLinks  `json:"social_links,omitempty" swaggertype:"object"`
	WorkingHours WorkingHours `json:"working_hours,omitempty" swaggertype:"object"`

	// Status va reyting
	IsVerified bool    `json:"is_verified" example:"false"`
	Rating     float64 `json:"rating" example:"4.5"`

	// Vaqt belgilari
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ============================================
// REQUEST/RESPONSE MODELS
// ============================================

// CreateSellerProfileRequest - sotuvchi profili yaratish so'rovi
type CreateSellerProfileRequest struct {
	ShopName     string       `json:"shop_name" binding:"required"`
	Description  string       `json:"description,omitempty"`
	SupportPhone string       `json:"support_phone,omitempty"`
	Address      *StringMap   `json:"address,omitempty"`
	SocialLinks  SocialLinks  `json:"social_links,omitempty"`
	WorkingHours WorkingHours `json:"working_hours,omitempty"`
}

// UpdateSellerProfileRequest - sotuvchi profili yangilash so'rovi
type UpdateSellerProfileRequest struct {
	ShopName     *string       `json:"shop_name,omitempty"`
	Description  *string       `json:"description,omitempty"`
	LogoURL      *string       `json:"logo_url,omitempty"`
	BannerURL    *string       `json:"banner_url,omitempty"`
	SupportPhone *string       `json:"support_phone,omitempty"`
	Address      *StringMap    `json:"address,omitempty"`
	Latitude     *float64      `json:"latitude,omitempty"`
	Longitude    *float64      `json:"longitude,omitempty"`
	SocialLinks  *SocialLinks  `json:"social_links,omitempty"`
	WorkingHours *WorkingHours `json:"working_hours,omitempty"`
}

// UpdateLegalInfoRequest - yuridik ma'lumotlarni yangilash
type UpdateLegalInfoRequest struct {
	LegalName   string `json:"legal_name"`
	TaxID       string `json:"tax_id"`
	BankAccount string `json:"bank_account"`
	BankName    string `json:"bank_name"`
}

// SellerProfileResponse - sotuvchi profili javobi
type SellerProfileResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message,omitempty"`
	Profile *SellerProfile `json:"profile,omitempty"`
}

// PublicSellerProfile - ommaviy sotuvchi profili (maxfiy ma'lumotlarsiz)
type PublicSellerProfile struct {
	ID           string       `json:"id"`
	ShopName     string       `json:"shop_name"`
	Slug         string       `json:"slug"`
	Description  string       `json:"description,omitempty"`
	LogoURL      string       `json:"logo_url,omitempty"`
	BannerURL    string       `json:"banner_url,omitempty"`
	SupportPhone string       `json:"support_phone,omitempty"`
	Address      StringMap    `json:"address,omitempty"`
	Latitude     *float64     `json:"latitude,omitempty"`
	Longitude    *float64     `json:"longitude,omitempty"`
	SocialLinks  SocialLinks  `json:"social_links,omitempty"`
	WorkingHours WorkingHours `json:"working_hours,omitempty"`
	IsVerified   bool         `json:"is_verified"`
	Rating       float64      `json:"rating"`
}

// ToPublic - maxfiy ma'lumotlarni yashirish
func (s *SellerProfile) ToPublic() PublicSellerProfile {
	return PublicSellerProfile{
		ID:           s.ID,
		ShopName:     s.ShopName,
		Slug:         s.Slug,
		Description:  s.Description,
		LogoURL:      s.LogoURL,
		BannerURL:    s.BannerURL,
		SupportPhone: s.SupportPhone,
		Address:      s.Address,
		Latitude:     s.Latitude,
		Longitude:    s.Longitude,
		SocialLinks:  s.SocialLinks,
		WorkingHours: s.WorkingHours,
		IsVerified:   s.IsVerified,
		Rating:       s.Rating,
	}
}

// GenerateSlug - do'kon nomi asosida slug yaratish
func GenerateSlug(shopName string) string {
	// Oddiy transliteratsiya va slug yaratish
	// Production'da github.com/gosimple/slug kutubxonasidan foydalaning
	slug := ""
	for _, r := range shopName {
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
	if len(slug) > 0 && slug[len(slug)-1] == '-' {
		slug = slug[:len(slug)-1]
	}
	return slug
}
