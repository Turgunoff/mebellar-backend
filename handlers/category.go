package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"mebellar-backend/models"

	"github.com/google/uuid"
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

// CreateCategoryRequest - Kategoriya yaratish so'rovi
type CreateCategoryRequest struct {
	Name     string  `json:"name"`
	IconURL  string  `json:"icon_url,omitempty"`
	ParentID *string `json:"parent_id,omitempty"`
}

// CreateCategory godoc
// @Summary      Yangi kategoriya yaratish (Admin)
// @Description  Admin panel uchun yangi kategoriya qo'shish (multipart/form-data bilan fayl yuklash)
// @Tags         admin
// @Accept       multipart/form-data
// @Produce      json
// @Param        name formData string true "Kategoriya nomi"
// @Param        parent_id formData string false "Parent kategoriya ID"
// @Param        icon formData file false "Kategoriya ikonasi (jpg, png)"
// @Success      201  {object}  models.CategoryResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/categories [post]
func CreateCategory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// User Role middleware dan olingan
		userRole := r.Header.Get("X-User-Role")
		if userRole != "admin" && userRole != "moderator" {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Bu sahifaga kirish huquqingiz yo'q",
			})
			return
		}

		// Multipart form data ni parse qilish (max 10MB)
		err := r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Forma ma'lumotlarini o'qib bo'lmadi: " + err.Error(),
			})
			return
		}

		// Name olish
		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Kategoriya nomi kiritilishi shart",
			})
			return
		}

		// Parent ID olish (optional)
		var parentID *string
		if parentIDStr := strings.TrimSpace(r.FormValue("parent_id")); parentIDStr != "" {
			parentID = &parentIDStr
			// Parent ID ni tekshirish
			var exists bool
			err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)", *parentID).Scan(&exists)
			if err != nil || !exists {
				writeJSON(w, http.StatusBadRequest, models.AuthResponse{
					Success: false,
					Message: "Parent kategoriya topilmadi",
				})
				return
			}
		}

		// Fayl yuklash
		var iconURL string
		file, header, err := r.FormFile("icon")
		if err == nil {
			defer file.Close()

			// Fayl tipini tekshirish
			contentType := header.Header.Get("Content-Type")
			allowedTypes := []string{"image/jpeg", "image/jpg", "image/png"}
			isValidType := false
			for _, allowedType := range allowedTypes {
				if contentType == allowedType {
					isValidType = true
					break
				}
			}

			// Fayl kengaytmasini tekshirish
			ext := strings.ToLower(filepath.Ext(header.Filename))
			if !isValidType && ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
				writeJSON(w, http.StatusBadRequest, models.AuthResponse{
					Success: false,
					Message: "Faqat JPG va PNG rasmlar qabul qilinadi",
				})
				return
			}

			// Upload papkasini yaratish (agar mavjud bo'lmasa)
			uploadDir := "./uploads/categories"
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				log.Printf("Upload papkasini yaratishda xatolik: %v", err)
				writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
					Success: false,
					Message: "Server xatosi: papka yaratib bo'lmadi",
				})
				return
			}

			// Unique filename yaratish (UUID + extension)
			fileID := uuid.New().String()
			filename := fileID + ext
			filePath := filepath.Join(uploadDir, filename)

			// Faylni saqlash
			dst, err := os.Create(filePath)
			if err != nil {
				log.Printf("Fayl yaratishda xatolik: %v", err)
				writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
					Success: false,
					Message: "Faylni saqlashda xatolik",
				})
				return
			}
			defer dst.Close()

			_, err = io.Copy(dst, file)
			if err != nil {
				log.Printf("Fayl nusxalashda xatolik: %v", err)
				writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
					Success: false,
					Message: "Faylni saqlashda xatolik",
				})
				return
			}

			// Relative path ni saqlash
			iconURL = "/uploads/categories/" + filename
			log.Printf("✅ Fayl yuklandi: %s", iconURL)
		}

		// Kategoriya ID yaratish (UUID)
		categoryID := uuid.New().String()

		// Kategoriyani bazaga qo'shish
		query := `
			INSERT INTO categories (id, name, icon_url, parent_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
			RETURNING id
		`

		var insertedID string
		err = db.QueryRow(query, categoryID, name, iconURL, parentID).Scan(&insertedID)
		if err != nil {
			log.Printf("CreateCategory: Insert xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Kategoriyani yaratishda xatolik: " + err.Error(),
			})
			return
		}

		log.Printf("✅ Kategoriya yaratildi: %s (ID: %s)", name, insertedID)

		// Yaratilgan kategoriyani qaytarish
		var c models.Category
		err = db.QueryRow(`
			SELECT id, parent_id, name, COALESCE(icon_url, ''),
			(SELECT COUNT(*) FROM products p WHERE p.category_id = c.id AND p.is_active = true) as product_count
			FROM categories c
			WHERE id = $1
		`, insertedID).Scan(&c.ID, &c.ParentID, &c.Name, &c.IconURL, &c.ProductCount)

		if err != nil {
			log.Printf("CreateCategory: Fetch xatosi: %v", err)
		}

		writeJSON(w, http.StatusCreated, models.CategoryResponse{
			Success:  true,
			Message:  "Kategoriya muvaffaqiyatli yaratildi",
			Category: &c,
		})
	}
}

// UpdateCategoryRequest - Kategoriya yangilash so'rovi
type UpdateCategoryRequest struct {
	Name    *string `json:"name,omitempty"`
	IconURL *string `json:"icon_url,omitempty"`
}

// UpdateCategory godoc
// @Summary      Kategoriyani yangilash (Admin)
// @Description  Admin panel uchun kategoriya ma'lumotlarini yangilash (multipart/form-data bilan fayl yuklash)
// @Tags         admin
// @Accept       multipart/form-data
// @Produce      json
// @Param        id path string true "Kategoriya ID"
// @Param        name formData string false "Kategoriya nomi"
// @Param        icon formData file false "Kategoriya ikonasi (jpg, png)"
// @Success      200  {object}  models.CategoryResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/categories/{id} [put]
func UpdateCategory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat PUT metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// User Role middleware dan olingan
		userRole := r.Header.Get("X-User-Role")
		if userRole != "admin" && userRole != "moderator" {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Bu sahifaga kirish huquqingiz yo'q",
			})
			return
		}

		// URL dan ID olish
		path := r.URL.Path
		parts := strings.Split(path, "/")
		if len(parts) < 5 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Kategoriya ID kiritilmagan",
			})
			return
		}
		categoryID := parts[len(parts)-1]

		// Multipart form data ni parse qilish (max 10MB)
		err := r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Forma ma'lumotlarini o'qib bo'lmadi: " + err.Error(),
			})
			return
		}

		// Kategoriya mavjudligini tekshirish
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)", categoryID).Scan(&exists)
		if err != nil || !exists {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Kategoriya topilmadi",
			})
			return
		}

		// Name olish (optional)
		var name *string
		if nameStr := strings.TrimSpace(r.FormValue("name")); nameStr != "" {
			name = &nameStr
		}

		// Fayl yuklash (optional)
		var iconURL *string
		file, header, err := r.FormFile("icon")
		if err == nil {
			defer file.Close()

			// Fayl tipini tekshirish
			contentType := header.Header.Get("Content-Type")
			allowedTypes := []string{"image/jpeg", "image/jpg", "image/png"}
			isValidType := false
			for _, allowedType := range allowedTypes {
				if contentType == allowedType {
					isValidType = true
					break
				}
			}

			// Fayl kengaytmasini tekshirish
			ext := strings.ToLower(filepath.Ext(header.Filename))
			if !isValidType && ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
				writeJSON(w, http.StatusBadRequest, models.AuthResponse{
					Success: false,
					Message: "Faqat JPG va PNG rasmlar qabul qilinadi",
				})
				return
			}

			// Upload papkasini yaratish (agar mavjud bo'lmasa)
			uploadDir := "./uploads/categories"
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				log.Printf("Upload papkasini yaratishda xatolik: %v", err)
				writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
					Success: false,
					Message: "Server xatosi: papka yaratib bo'lmadi",
				})
				return
			}

			// Eski faylni o'chirish (agar mavjud bo'lsa)
			var oldIconURL string
			err = db.QueryRow("SELECT COALESCE(icon_url, '') FROM categories WHERE id = $1", categoryID).Scan(&oldIconURL)
			if err == nil && oldIconURL != "" && strings.HasPrefix(oldIconURL, "/uploads/categories/") {
				oldFilePath := "." + oldIconURL
				if err := os.Remove(oldFilePath); err != nil {
					log.Printf("Eski faylni o'chirishda xatolik (e'tiborsiz): %v", err)
				}
			}

			// Unique filename yaratish (UUID + extension)
			fileID := uuid.New().String()
			filename := fileID + ext
			filePath := filepath.Join(uploadDir, filename)

			// Faylni saqlash
			dst, err := os.Create(filePath)
			if err != nil {
				log.Printf("Fayl yaratishda xatolik: %v", err)
				writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
					Success: false,
					Message: "Faylni saqlashda xatolik",
				})
				return
			}
			defer dst.Close()

			_, err = io.Copy(dst, file)
			if err != nil {
				log.Printf("Fayl nusxalashda xatolik: %v", err)
				writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
					Success: false,
					Message: "Faylni saqlashda xatolik",
				})
				return
			}

			// Relative path ni saqlash
			iconPath := "/uploads/categories/" + filename
			iconURL = &iconPath
			log.Printf("✅ Fayl yuklandi: %s", iconPath)
		}

		// Hech narsa yangilanmayapti
		if name == nil && iconURL == nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Hech narsa o'zgartirilmadi",
			})
			return
		}

		// Yangilash so'rovini yaratish
		updateFields := []string{}
		args := []interface{}{}
		argIndex := 1

		if name != nil && *name != "" {
			updateFields = append(updateFields, fmt.Sprintf("name = $%d", argIndex))
			args = append(args, *name)
			argIndex++
		}

		if iconURL != nil {
			updateFields = append(updateFields, fmt.Sprintf("icon_url = $%d", argIndex))
			args = append(args, *iconURL)
			argIndex++
		}

		updateFields = append(updateFields, "updated_at = NOW()")
		args = append(args, categoryID)

		query := fmt.Sprintf("UPDATE categories SET %s WHERE id = $%d", strings.Join(updateFields, ", "), argIndex)

		_, err = db.Exec(query, args...)
		if err != nil {
			log.Printf("UpdateCategory: Update xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Kategoriyani yangilashda xatolik: " + err.Error(),
			})
			return
		}

		log.Printf("✅ Kategoriya yangilandi: %s", categoryID)

		// Yangilangan kategoriyani qaytarish
		var c models.Category
		err = db.QueryRow(`
			SELECT id, parent_id, name, COALESCE(icon_url, ''),
			(SELECT COUNT(*) FROM products p WHERE p.category_id = c.id AND p.is_active = true) as product_count
			FROM categories c
			WHERE id = $1
		`, categoryID).Scan(&c.ID, &c.ParentID, &c.Name, &c.IconURL, &c.ProductCount)

		if err != nil {
			log.Printf("UpdateCategory: Fetch xatosi: %v", err)
		}

		writeJSON(w, http.StatusOK, models.CategoryResponse{
			Success:  true,
			Message:  "Kategoriya muvaffaqiyatli yangilandi",
			Category: &c,
		})
	}
}

// DeleteCategory godoc
// @Summary      Kategoriyani o'chirish (Admin)
// @Description  Admin panel uchun kategoriyani o'chirish
// @Tags         admin
// @Produce      json
// @Param        id path string true "Kategoriya ID"
// @Success      200  {object}  models.AuthResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/categories/{id} [delete]
func DeleteCategory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat DELETE metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// User Role middleware dan olingan
		userRole := r.Header.Get("X-User-Role")
		if userRole != "admin" && userRole != "moderator" {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Bu sahifaga kirish huquqingiz yo'q",
			})
			return
		}

		// URL dan ID olish
		path := r.URL.Path
		parts := strings.Split(path, "/")
		if len(parts) < 5 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Kategoriya ID kiritilmagan",
			})
			return
		}
		categoryID := parts[len(parts)-1]

		// Kategoriyada mahsulotlar bormi tekshirish
		var productCount int
		err := db.QueryRow("SELECT COUNT(*) FROM products WHERE category_id = $1 AND is_active = true", categoryID).Scan(&productCount)
		if err == nil && productCount > 0 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: fmt.Sprintf("Bu kategoriyada %d ta faol mahsulot mavjud. Avval mahsulotlarni o'chiring yoki boshqa kategoriyaga ko'chiring", productCount),
			})
			return
		}

		// Sub-kategoriyalar bormi tekshirish
		var subCount int
		err = db.QueryRow("SELECT COUNT(*) FROM categories WHERE parent_id = $1", categoryID).Scan(&subCount)
		if err == nil && subCount > 0 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: fmt.Sprintf("Bu kategoriyada %d ta sub-kategoriya mavjud. Avval sub-kategoriyalarni o'chiring", subCount),
			})
			return
		}

		// Kategoriyani o'chirish
		result, err := db.Exec("DELETE FROM categories WHERE id = $1", categoryID)
		if err != nil {
			log.Printf("DeleteCategory: Delete xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Kategoriyani o'chirishda xatolik: " + err.Error(),
			})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Kategoriya topilmadi",
			})
			return
		}

		log.Printf("✅ Kategoriya o'chirildi: %s", categoryID)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: "Kategoriya muvaffaqiyatli o'chirildi",
		})
	}
}

// AdminCategoryHandler - PUT va DELETE uchun handler (/api/admin/categories/{id})
func AdminCategoryHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			UpdateCategory(db)(w, r)
		case http.MethodDelete:
			DeleteCategory(db)(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat PUT yoki DELETE metodi qo'llab-quvvatlanadi",
			})
		}
	}
}
