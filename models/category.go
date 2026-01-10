package models

// Category - kategoriya modeli
// @Description Mahsulot kategoriyasi
type Category struct {
	ID            string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ParentID      *string    `json:"parent_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name          string     `json:"name" example:"Yashash xonasi"`
	IconURL       string     `json:"icon_url" example:"https://img.icons8.com/fluency/96/living-room.png"`
	ProductCount  int        `json:"product_count,omitempty" example:"25"`
	SubCategories []Category `json:"sub_categories,omitempty"`
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
