package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"mebellar-backend/models"

	"github.com/google/uuid"
)

// GetCategoryAttributes godoc
// @Summary      Kategoriya atributlarini olish
// @Description  Kategoriya ID bo'yicha dinamik form maydonlarini qaytaradi (sort_order bo'yicha tartiblangan)
// @Tags         categories
// @Accept       json
// @Produce      json
// @Param        id path string true "Kategoriya ID"
// @Success      200  {object}  models.CategoryAttributesResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /categories/{id}/attributes [get]
func GetCategoryAttributes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// URL dan category ID olish: /api/categories/{id}/attributes
		path := r.URL.Path
		parts := strings.Split(path, "/")
		
		// Find the category ID (before "attributes")
		var categoryID string
		for i, part := range parts {
			if part == "attributes" && i > 0 {
				categoryID = parts[i-1]
				break
			}
		}

		if categoryID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Kategoriya ID kiritilmagan",
			})
			return
		}

		// UUID validatsiyasi
		if _, err := uuid.Parse(categoryID); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri kategoriya ID formati",
			})
			return
		}

		// Kategoriya mavjudligini tekshirish
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)", categoryID).Scan(&exists)
		if err != nil || !exists {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Kategoriya topilmadi",
			})
			return
		}

		// Atributlarni olish
		query := `
			SELECT id, category_id, key, type, label::text, 
			       COALESCE(options::text, '[]'), is_required, sort_order,
			       created_at, updated_at
			FROM category_attributes
			WHERE category_id = $1
			ORDER BY sort_order ASC, created_at ASC
		`

		rows, err := db.Query(query, categoryID)
		if err != nil {
			log.Printf("GetCategoryAttributes query error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Atributlarni olishda xatolik",
			})
			return
		}
		defer rows.Close()

		attributes := []models.CategoryAttribute{}
		for rows.Next() {
			var attr models.CategoryAttribute
			var labelJSON, optionsJSON string

			err := rows.Scan(
				&attr.ID, &attr.CategoryID, &attr.Key, &attr.Type,
				&labelJSON, &optionsJSON, &attr.IsRequired, &attr.SortOrder,
				&attr.CreatedAt, &attr.UpdatedAt,
			)
			if err != nil {
				log.Printf("GetCategoryAttributes scan error: %v", err)
				continue
			}

			// Parse label JSON
			if err := json.Unmarshal([]byte(labelJSON), &attr.Label); err != nil {
				log.Printf("Label parse error: %v", err)
				attr.Label = map[string]string{}
			}

			// Parse options JSON
			if err := json.Unmarshal([]byte(optionsJSON), &attr.Options); err != nil {
				log.Printf("Options parse error: %v", err)
				attr.Options = nil
			}

			attributes = append(attributes, attr)
		}

		log.Printf("✅ %d ta atribut topildi (category: %s)", len(attributes), categoryID)

		writeJSON(w, http.StatusOK, models.CategoryAttributesResponse{
			Success:    true,
			Attributes: attributes,
			Count:      len(attributes),
		})
	}
}

// CreateCategoryAttribute godoc
// @Summary      Yangi kategoriya atributi yaratish (Admin)
// @Description  Admin panel uchun kategoriyaga yangi dinamik form maydoni qo'shish
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path string true "Kategoriya ID"
// @Param        request body models.CreateCategoryAttributeRequest true "Atribut ma'lumotlari"
// @Success      201  {object}  models.CategoryAttributeResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/categories/{id}/attributes [post]
func CreateCategoryAttribute(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Admin check
		userRole := r.Header.Get("X-User-Role")
		if userRole != "admin" && userRole != "moderator" {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Bu sahifaga kirish huquqingiz yo'q",
			})
			return
		}

		// URL dan category ID olish: /api/admin/categories/{id}/attributes
		path := r.URL.Path
		parts := strings.Split(path, "/")
		
		var categoryID string
		for i, part := range parts {
			if part == "attributes" && i > 0 {
				categoryID = parts[i-1]
				break
			}
		}

		if categoryID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Kategoriya ID kiritilmagan",
			})
			return
		}

		// UUID validatsiyasi
		if _, err := uuid.Parse(categoryID); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri kategoriya ID formati",
			})
			return
		}

		// Kategoriya mavjudligini tekshirish
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)", categoryID).Scan(&exists)
		if err != nil || !exists {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Kategoriya topilmadi",
			})
			return
		}

		// Request body parse
		var req models.CreateCategoryAttributeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri JSON format: " + err.Error(),
			})
			return
		}

		// Validatsiya
		if req.Key == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Key maydoni kiritilishi shart",
			})
			return
		}

		if !models.ValidateAttributeType(req.Type) {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri type. Ruxsat etilgan: text, number, dropdown, switch",
			})
			return
		}

		if req.Label == nil || len(req.Label) == 0 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Label maydoni kiritilishi shart",
			})
			return
		}

		// Dropdown uchun options tekshirish
		if req.Type == "dropdown" && (req.Options == nil || len(req.Options) == 0) {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Dropdown turi uchun options kiritilishi shart",
			})
			return
		}

		// Key unique bo'lishini tekshirish
		var keyExists bool
		err = db.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM category_attributes WHERE category_id = $1 AND key = $2)",
			categoryID, req.Key,
		).Scan(&keyExists)
		if err == nil && keyExists {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: fmt.Sprintf("'%s' kalit bu kategoriyada allaqachon mavjud", req.Key),
			})
			return
		}

		// Label va Options ni JSON ga o'tkazish
		labelJSON, err := json.Marshal(req.Label)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Label JSON formatiga o'tkazishda xatolik",
			})
			return
		}

		var optionsJSON []byte
		if req.Options != nil {
			optionsJSON, err = json.Marshal(req.Options)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
					Success: false,
					Message: "Options JSON formatiga o'tkazishda xatolik",
				})
				return
			}
		}

		// Atribut ID yaratish
		attrID := uuid.New().String()

		// Bazaga qo'shish
		query := `
			INSERT INTO category_attributes 
			(id, category_id, key, type, label, options, is_required, sort_order)
			VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, $8)
			RETURNING id, created_at, updated_at
		`

		var attr models.CategoryAttribute
		err = db.QueryRow(
			query,
			attrID, categoryID, req.Key, req.Type,
			string(labelJSON), string(optionsJSON),
			req.IsRequired, req.SortOrder,
		).Scan(&attr.ID, &attr.CreatedAt, &attr.UpdatedAt)

		if err != nil {
			log.Printf("CreateCategoryAttribute insert error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Atribut yaratishda xatolik: " + err.Error(),
			})
			return
		}

		// Response tayyorlash
		attr.CategoryID = categoryID
		attr.Key = req.Key
		attr.Type = req.Type
		attr.Label = req.Label
		attr.Options = req.Options
		attr.IsRequired = req.IsRequired
		attr.SortOrder = req.SortOrder

		log.Printf("✅ Yangi atribut yaratildi: %s (key: %s, category: %s)", attr.ID, req.Key, categoryID)

		writeJSON(w, http.StatusCreated, models.CategoryAttributeResponse{
			Success:   true,
			Message:   "Atribut muvaffaqiyatli yaratildi",
			Attribute: &attr,
		})
	}
}

// UpdateCategoryAttribute godoc
// @Summary      Kategoriya atributini yangilash (Admin)
// @Description  Admin panel uchun atribut ma'lumotlarini yangilash
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path string true "Atribut ID"
// @Param        request body models.UpdateCategoryAttributeRequest true "Yangilash ma'lumotlari"
// @Success      200  {object}  models.CategoryAttributeResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/category-attributes/{id} [put]
func UpdateCategoryAttribute(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat PUT metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Admin check
		userRole := r.Header.Get("X-User-Role")
		if userRole != "admin" && userRole != "moderator" {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Bu sahifaga kirish huquqingiz yo'q",
			})
			return
		}

		// URL dan attribute ID olish
		path := r.URL.Path
		parts := strings.Split(path, "/")
		attrID := parts[len(parts)-1]

		if attrID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Atribut ID kiritilmagan",
			})
			return
		}

		// Atribut mavjudligini tekshirish
		var existingAttr models.CategoryAttribute
		var labelJSON, optionsJSON string
		err := db.QueryRow(`
			SELECT id, category_id, key, type, label::text, COALESCE(options::text, '[]'),
			       is_required, sort_order, created_at, updated_at
			FROM category_attributes WHERE id = $1
		`, attrID).Scan(
			&existingAttr.ID, &existingAttr.CategoryID, &existingAttr.Key,
			&existingAttr.Type, &labelJSON, &optionsJSON,
			&existingAttr.IsRequired, &existingAttr.SortOrder,
			&existingAttr.CreatedAt, &existingAttr.UpdatedAt,
		)

		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Atribut topilmadi",
			})
			return
		}
		if err != nil {
			log.Printf("UpdateCategoryAttribute fetch error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Atributni olishda xatolik",
			})
			return
		}

		// Parse existing JSON fields
		json.Unmarshal([]byte(labelJSON), &existingAttr.Label)
		json.Unmarshal([]byte(optionsJSON), &existingAttr.Options)

		// Request body parse
		var req models.UpdateCategoryAttributeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri JSON format: " + err.Error(),
			})
			return
		}

		// Build update query dynamically
		updateFields := []string{}
		args := []interface{}{}
		argIndex := 1

		if req.Key != nil && *req.Key != "" {
			// Key unique bo'lishini tekshirish (o'zi bundan mustasno)
			var keyExists bool
			err = db.QueryRow(
				"SELECT EXISTS(SELECT 1 FROM category_attributes WHERE category_id = $1 AND key = $2 AND id != $3)",
				existingAttr.CategoryID, *req.Key, attrID,
			).Scan(&keyExists)
			if err == nil && keyExists {
				writeJSON(w, http.StatusBadRequest, models.AuthResponse{
					Success: false,
					Message: fmt.Sprintf("'%s' kalit bu kategoriyada allaqachon mavjud", *req.Key),
				})
				return
			}
			updateFields = append(updateFields, fmt.Sprintf("key = $%d", argIndex))
			args = append(args, *req.Key)
			argIndex++
		}

		if req.Type != nil {
			if !models.ValidateAttributeType(*req.Type) {
				writeJSON(w, http.StatusBadRequest, models.AuthResponse{
					Success: false,
					Message: "Noto'g'ri type. Ruxsat etilgan: text, number, dropdown, switch",
				})
				return
			}
			updateFields = append(updateFields, fmt.Sprintf("type = $%d", argIndex))
			args = append(args, *req.Type)
			argIndex++
		}

		if req.Label != nil && len(req.Label) > 0 {
			labelBytes, _ := json.Marshal(req.Label)
			updateFields = append(updateFields, fmt.Sprintf("label = $%d::jsonb", argIndex))
			args = append(args, string(labelBytes))
			argIndex++
		}

		if req.Options != nil {
			optionsBytes, _ := json.Marshal(req.Options)
			updateFields = append(updateFields, fmt.Sprintf("options = $%d::jsonb", argIndex))
			args = append(args, string(optionsBytes))
			argIndex++
		}

		if req.IsRequired != nil {
			updateFields = append(updateFields, fmt.Sprintf("is_required = $%d", argIndex))
			args = append(args, *req.IsRequired)
			argIndex++
		}

		if req.SortOrder != nil {
			updateFields = append(updateFields, fmt.Sprintf("sort_order = $%d", argIndex))
			args = append(args, *req.SortOrder)
			argIndex++
		}

		if len(updateFields) == 0 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Hech narsa o'zgartirilmadi",
			})
			return
		}

		// Add WHERE clause
		updateFields = append(updateFields, "updated_at = NOW()")
		args = append(args, attrID)

		query := fmt.Sprintf(
			"UPDATE category_attributes SET %s WHERE id = $%d RETURNING updated_at",
			strings.Join(updateFields, ", "), argIndex,
		)

		var updatedAt string
		err = db.QueryRow(query, args...).Scan(&updatedAt)
		if err != nil {
			log.Printf("UpdateCategoryAttribute update error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Atributni yangilashda xatolik: " + err.Error(),
			})
			return
		}

		// Fetch updated attribute
		var attr models.CategoryAttribute
		err = db.QueryRow(`
			SELECT id, category_id, key, type, label::text, COALESCE(options::text, '[]'),
			       is_required, sort_order, created_at, updated_at
			FROM category_attributes WHERE id = $1
		`, attrID).Scan(
			&attr.ID, &attr.CategoryID, &attr.Key, &attr.Type,
			&labelJSON, &optionsJSON, &attr.IsRequired, &attr.SortOrder,
			&attr.CreatedAt, &attr.UpdatedAt,
		)
		if err == nil {
			json.Unmarshal([]byte(labelJSON), &attr.Label)
			json.Unmarshal([]byte(optionsJSON), &attr.Options)
		}

		log.Printf("✅ Atribut yangilandi: %s", attrID)

		writeJSON(w, http.StatusOK, models.CategoryAttributeResponse{
			Success:   true,
			Message:   "Atribut muvaffaqiyatli yangilandi",
			Attribute: &attr,
		})
	}
}

// DeleteCategoryAttribute godoc
// @Summary      Kategoriya atributini o'chirish (Admin)
// @Description  Admin panel uchun atributni o'chirish
// @Tags         admin
// @Produce      json
// @Param        id path string true "Atribut ID"
// @Success      200  {object}  models.AuthResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/category-attributes/{id} [delete]
func DeleteCategoryAttribute(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat DELETE metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Admin check
		userRole := r.Header.Get("X-User-Role")
		if userRole != "admin" && userRole != "moderator" {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Bu sahifaga kirish huquqingiz yo'q",
			})
			return
		}

		// URL dan attribute ID olish
		path := r.URL.Path
		parts := strings.Split(path, "/")
		attrID := parts[len(parts)-1]

		if attrID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Atribut ID kiritilmagan",
			})
			return
		}

		// O'chirish
		result, err := db.Exec("DELETE FROM category_attributes WHERE id = $1", attrID)
		if err != nil {
			log.Printf("DeleteCategoryAttribute error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Atributni o'chirishda xatolik: " + err.Error(),
			})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Atribut topilmadi",
			})
			return
		}

		log.Printf("✅ Atribut o'chirildi: %s", attrID)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: "Atribut muvaffaqiyatli o'chirildi",
		})
	}
}

// AdminCategoryAttributeHandler - PUT va DELETE uchun handler (/api/admin/category-attributes/{id})
func AdminCategoryAttributeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			UpdateCategoryAttribute(db)(w, r)
		case http.MethodDelete:
			DeleteCategoryAttribute(db)(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat PUT yoki DELETE metodi qo'llab-quvvatlanadi",
			})
		}
	}
}

// CategoryAttributesRouter - /api/categories/{id}/attributes uchun router
// Bu handler GetCategoryByID bilan birgalikda ishlaydi
func CategoryAttributesRouter(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		
		// Check if path ends with /attributes
		if strings.HasSuffix(path, "/attributes") {
			GetCategoryAttributes(db)(w, r)
			return
		}
		
		// Otherwise, delegate to GetCategoryByID
		GetCategoryByID(db)(w, r)
	}
}

// AdminCategoryAttributesRouter - /api/admin/categories/{id}/attributes uchun router
func AdminCategoryAttributesRouter(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		
		// Check if path ends with /attributes and is POST
		if strings.HasSuffix(path, "/attributes") && r.Method == http.MethodPost {
			CreateCategoryAttribute(db)(w, r)
			return
		}
		
		// Otherwise, delegate to existing AdminCategoryHandler
		AdminCategoryHandler(db)(w, r)
	}
}
