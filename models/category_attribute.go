package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// AttributeOption - dropdown uchun variant
// @Description Dropdown attribute uchun tanlov varianti
type AttributeOption struct {
	Value string            `json:"value" example:"mdf"`
	Label map[string]string `json:"label" swaggertype:"object"`
}

// AttributeOptions - AttributeOption massivi uchun custom type
type AttributeOptions []AttributeOption

// Value - database ga yozish uchun
func (o AttributeOptions) Value() (driver.Value, error) {
	if o == nil {
		return nil, nil
	}
	return json.Marshal(o)
}

// Scan - database dan o'qish uchun
func (o *AttributeOptions) Scan(value interface{}) error {
	if value == nil {
		*o = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("type assertion to []byte or string failed for AttributeOptions")
	}

	if len(bytes) == 0 || string(bytes) == "null" {
		*o = nil
		return nil
	}

	return json.Unmarshal(bytes, o)
}

// CategoryAttribute - kategoriya atributi modeli
// @Description Kategoriya uchun dinamik form maydoni ta'rifi
type CategoryAttribute struct {
	ID         string           `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CategoryID string           `json:"category_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Key        string           `json:"key" example:"mechanism"`
	Type       string           `json:"type" example:"dropdown" enums:"text,number,dropdown,switch"`
	Label      StringMap        `json:"label" swaggertype:"object"`
	Options    AttributeOptions `json:"options,omitempty" swaggertype:"array,object"`
	IsRequired bool             `json:"is_required" example:"true"`
	SortOrder  int              `json:"sort_order" example:"1"`
	CreatedAt  time.Time        `json:"created_at,omitempty"`
	UpdatedAt  time.Time        `json:"updated_at,omitempty"`
}

// GetLabel - Helper function to get label in a specific language, fallback to 'uz'
func (a *CategoryAttribute) GetLabel(lang string) string {
	if a.Label == nil {
		return a.Key
	}
	if label, ok := a.Label[lang]; ok && label != "" {
		return label
	}
	// Fallback to 'uz'
	if label, ok := a.Label["uz"]; ok {
		return label
	}
	// If 'uz' doesn't exist, return key
	return a.Key
}

// GetOptionLabel - Get option label in a specific language
func (a *CategoryAttribute) GetOptionLabel(value string, lang string) string {
	for _, opt := range a.Options {
		if opt.Value == value {
			if label, ok := opt.Label[lang]; ok && label != "" {
				return label
			}
			// Fallback to 'uz'
			if label, ok := opt.Label["uz"]; ok {
				return label
			}
			return value
		}
	}
	return value
}

// CategoryAttributeResponse - bitta atribut javobi
type CategoryAttributeResponse struct {
	Success   bool               `json:"success"`
	Message   string             `json:"message,omitempty"`
	Attribute *CategoryAttribute `json:"attribute,omitempty"`
}

// CategoryAttributesResponse - atributlar javob modeli
type CategoryAttributesResponse struct {
	Success    bool                `json:"success"`
	Message    string              `json:"message,omitempty"`
	Attributes []CategoryAttribute `json:"attributes"`
	Count      int                 `json:"count"`
}

// CreateCategoryAttributeRequest - atribut yaratish so'rovi
type CreateCategoryAttributeRequest struct {
	Key        string            `json:"key" example:"mechanism"`
	Type       string            `json:"type" example:"dropdown"`
	Label      map[string]string `json:"label"`
	Options    []AttributeOption `json:"options,omitempty"`
	IsRequired bool              `json:"is_required"`
	SortOrder  int               `json:"sort_order"`
}

// UpdateCategoryAttributeRequest - atribut yangilash so'rovi
type UpdateCategoryAttributeRequest struct {
	Key        *string           `json:"key,omitempty"`
	Type       *string           `json:"type,omitempty"`
	Label      map[string]string `json:"label,omitempty"`
	Options    []AttributeOption `json:"options,omitempty"`
	IsRequired *bool             `json:"is_required,omitempty"`
	SortOrder  *int              `json:"sort_order,omitempty"`
}

// ValidateType - type qiymatini tekshirish
func ValidateAttributeType(t string) bool {
	validTypes := []string{"text", "number", "dropdown", "switch"}
	for _, valid := range validTypes {
		if t == valid {
			return true
		}
	}
	return false
}
