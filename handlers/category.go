package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	"mebellar-backend/models"
)

// GetCategories godoc
// @Summary      Barcha kategoriyalarni olish (daraxt ko'rinishida)
// @Description  Kategoriyalarni ierarxik (nested) formatda qaytaradi
// @Tags         categories
// @Accept       json
// @Produce      json
// @Param        flat query bool false "Tekis ro'yxat (default: false)"
// @Success      200  {object}  models.CategoriesResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /categories [get]
func GetCategories(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Flat parametrini tekshirish
		flat := r.URL.Query().Get("flat") == "true"

		if flat {
			// Tekis ro'yxat (barcha kategoriyalar)
			getFlatCategories(db, w)
			return
		}

		// Daraxt ko'rinishida (nested)
		getNestedCategories(db, w)
	}
}

// getNestedCategories - kategoriyalarni daraxt ko'rinishida olish
func getNestedCategories(db *sql.DB, w http.ResponseWriter) {
	// Barcha kategoriyalarni olish
	query := `
		SELECT 
			c.id, c.parent_id, c.name, COALESCE(c.icon_url, ''),
			(SELECT COUNT(*) FROM products p WHERE p.category_id = c.id AND p.is_active = true) as product_count
		FROM categories c
		ORDER BY c.name ASC
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Categories query xatosi: %v", err)
		writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
			Success: false,
			Message: "Kategoriyalarni olishda xatolik",
		})
		return
	}
	defer rows.Close()

	// Barcha kategoriyalarni map ga yuklash
	allCategories := make(map[string]*models.Category)
	var rootCategories []*models.Category

	for rows.Next() {
		var c models.Category
		err := rows.Scan(&c.ID, &c.ParentID, &c.Name, &c.IconURL, &c.ProductCount)
		if err != nil {
			log.Printf("Category scan xatosi: %v", err)
			continue
		}

		c.SubCategories = []models.Category{} // Initialize
		allCategories[c.ID] = &c

		if c.ParentID == nil {
			rootCategories = append(rootCategories, &c)
		}
	}

	// Sub-kategoriyalarni bog'lash
	for _, cat := range allCategories {
		if cat.ParentID != nil {
			if parent, ok := allCategories[*cat.ParentID]; ok {
				parent.SubCategories = append(parent.SubCategories, *cat)
			}
		}
	}

	// Natijani tayyorlash
	result := make([]models.Category, len(rootCategories))
	for i, cat := range rootCategories {
		result[i] = *cat
	}

	log.Printf("✅ %d ta asosiy kategoriya topildi", len(result))

	writeJSON(w, http.StatusOK, models.CategoriesResponse{
		Success:    true,
		Categories: result,
		Count:      len(result),
	})
}

// getFlatCategories - barcha kategoriyalarni tekis ro'yxatda olish
func getFlatCategories(db *sql.DB, w http.ResponseWriter) {
	query := `
		SELECT 
			c.id, c.parent_id, c.name, COALESCE(c.icon_url, ''),
			(SELECT COUNT(*) FROM products p WHERE p.category_id = c.id AND p.is_active = true) as product_count
		FROM categories c
		ORDER BY c.parent_id NULLS FIRST, c.name ASC
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Categories query xatosi: %v", err)
		writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
			Success: false,
			Message: "Kategoriyalarni olishda xatolik",
		})
		return
	}
	defer rows.Close()

	categories := []models.Category{}
	for rows.Next() {
		var c models.Category
		err := rows.Scan(&c.ID, &c.ParentID, &c.Name, &c.IconURL, &c.ProductCount)
		if err != nil {
			log.Printf("Category scan xatosi: %v", err)
			continue
		}
		categories = append(categories, c)
	}

	log.Printf("✅ %d ta kategoriya topildi (flat)", len(categories))

	writeJSON(w, http.StatusOK, models.CategoriesResponse{
		Success:    true,
		Categories: categories,
		Count:      len(categories),
	})
}

// GetCategoryByID godoc
// @Summary      Kategoriya ma'lumotlarini olish
// @Description  ID bo'yicha kategoriya ma'lumotlarini qaytaradi
// @Tags         categories
// @Accept       json
// @Produce      json
// @Param        id path string true "Kategoriya ID"
// @Success      200  {object}  models.CategoryResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /categories/{id} [get]
func GetCategoryByID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// URL dan ID olish: /api/categories/123 -> 123
		path := r.URL.Path
		parts := strings.Split(path, "/")
		if len(parts) < 4 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Kategoriya ID kiritilmagan",
			})
			return
		}
		categoryID := parts[len(parts)-1]

		if categoryID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri kategoriya ID",
			})
			return
		}

		// Kategoriyani olish (sub-kategoriyalar bilan)
		query := `
			SELECT id, parent_id, name, COALESCE(icon_url, ''),
			(SELECT COUNT(*) FROM products p WHERE p.category_id = c.id AND p.is_active = true) as product_count
			FROM categories c
			WHERE id = $1
		`

		var c models.Category
		err := db.QueryRow(query, categoryID).Scan(&c.ID, &c.ParentID, &c.Name, &c.IconURL, &c.ProductCount)

		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Kategoriya topilmadi",
			})
			return
		}

		if err != nil {
			log.Printf("Category query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Kategoriyani olishda xatolik",
			})
			return
		}

		// Sub-kategoriyalarni olish
		subQuery := `
			SELECT id, parent_id, name, COALESCE(icon_url, ''),
			(SELECT COUNT(*) FROM products p WHERE p.category_id = sc.id AND p.is_active = true) as product_count
			FROM categories sc
			WHERE parent_id = $1
			ORDER BY name ASC
		`

		rows, err := db.Query(subQuery, categoryID)
		if err == nil {
			defer rows.Close()
			c.SubCategories = []models.Category{}
			for rows.Next() {
				var sub models.Category
				if err := rows.Scan(&sub.ID, &sub.ParentID, &sub.Name, &sub.IconURL, &sub.ProductCount); err == nil {
					c.SubCategories = append(c.SubCategories, sub)
				}
			}
		}

		log.Printf("✅ Kategoriya topildi: %s (%d ta sub-kategoriya)", c.Name, len(c.SubCategories))

		writeJSON(w, http.StatusOK, models.CategoryResponse{
			Success:  true,
			Category: &c,
		})
	}
}
