package models

import "time"

// Banner - banner modeli
// @Description Home sahifadagi banner
type Banner struct {
	ID         string            `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Title      map[string]string `json:"title"`    // {"uz": "...", "ru": "...", "en": "..."}
	Subtitle   map[string]string `json:"subtitle"` // {"uz": "...", "ru": "...", "en": "..."}
	ImageURL   string            `json:"image_url" example:"https://example.com/banner.jpg"`
	TargetType string            `json:"target_type" example:"category"` // none, category, product, external
	TargetID   *string           `json:"target_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	SortOrder  int               `json:"sort_order" example:"0"`
	IsActive   bool              `json:"is_active" example:"true"`
	CreatedAt  time.Time         `json:"created_at"`
}

// GetTitle - Helper function to get title in a specific language, fallback to 'uz'
func (b *Banner) GetTitle(lang string) string {
	if b.Title == nil {
		return ""
	}
	if title, ok := b.Title[lang]; ok && title != "" {
		return title
	}
	// Fallback to 'uz'
	if title, ok := b.Title["uz"]; ok {
		return title
	}
	// If 'uz' doesn't exist, return first available
	for _, title := range b.Title {
		return title
	}
	return ""
}

// GetSubtitle - Helper function to get subtitle in a specific language, fallback to 'uz'
func (b *Banner) GetSubtitle(lang string) string {
	if b.Subtitle == nil {
		return ""
	}
	if subtitle, ok := b.Subtitle[lang]; ok && subtitle != "" {
		return subtitle
	}
	// Fallback to 'uz'
	if subtitle, ok := b.Subtitle["uz"]; ok {
		return subtitle
	}
	// If 'uz' doesn't exist, return first available
	for _, subtitle := range b.Subtitle {
		return subtitle
	}
	return ""
}

// BannersResponse - bannerlar javob modeli
type BannersResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message,omitempty"`
	Banners []Banner `json:"banners"`
	Count   int      `json:"count"`
}
