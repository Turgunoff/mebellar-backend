package models

// Category - kategoriya modeli
// @Description Mahsulot kategoriyasi
type Category struct {
	ID            string                `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ParentID      *string               `json:"parent_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name          map[string]string    `json:"name" example:"{\"uz\":\"Yashash xonasi\",\"ru\":\"Гостиная\",\"en\":\"Living Room\"}"`
	Slug          string                `json:"slug" example:"living-room"`
	IconURL       string                `json:"icon_url" example:"https://img.icons8.com/fluency/96/living-room.png"`
	IsActive      bool                  `json:"is_active" example:"true"`
	SortOrder     int                   `json:"sort_order" example:"0"`
	ProductCount  int                   `json:"product_count,omitempty" example:"25"`
	SubCategories []Category            `json:"sub_categories,omitempty"`
}

// GetName - Helper function to get name in a specific language, fallback to 'uz'
func (c *Category) GetName(lang string) string {
	if c.Name == nil {
		return ""
	}
	if name, ok := c.Name[lang]; ok && name != "" {
		return name
	}
	// Fallback to 'uz'
	if name, ok := c.Name["uz"]; ok {
		return name
	}
	// If 'uz' doesn't exist, return first available
	for _, name := range c.Name {
		return name
	}
	return ""
}

// CategoryResponse - bitta kategoriya javobi
type CategoryResponse struct {
	Success  bool      `json:"success"`
	Message  string    `json:"message,omitempty"`
	Category *Category `json:"category,omitempty"`
}

// CategoriesResponse - kategoriyalar javob modeli
type CategoriesResponse struct {
	Success    bool       `json:"success"`
	Message    string     `json:"message,omitempty"`
	Categories []Category `json:"categories"`
	Count      int        `json:"count"`
}

// FlatCategory - tekis kategoriya (parent bilan)
type FlatCategory struct {
	ID         string  `json:"id"`
	ParentID   *string `json:"parent_id,omitempty"`
	ParentName *string `json:"parent_name,omitempty"`
	Name       string  `json:"name"`
	IconURL    string  `json:"icon_url"`
}
