package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"mebellar-backend/models"

	"github.com/google/uuid"
)

// generateSlug - name dan slug yaratish (URL-friendly)
// Masalan: "Yashash xonasi" -> "yashash-xonasi"
func generateSlug(name string) string {
	// Kichik harflarga o'tkazish
	slug := strings.ToLower(name)
	
	// Maxsus belgilarni olib tashlash va bo'shliqlarni tire bilan almashtirish
	reg := regexp.MustCompile(`[^a-z0-9\s-]`)
	slug = reg.ReplaceAllString(slug, "")
	
	// Bo'shliqlarni tire bilan almashtirish
	reg = regexp.MustCompile(`[\s]+`)
	slug = reg.ReplaceAllString(slug, "-")
	
	// Boshida va oxirida tirelarni olib tashlash
	slug = strings.Trim(slug, "-")
	
	// Agar slug bo'sh bo'lsa, default qiymat
	if slug == "" {
		slug = "category"
	}
	
	return slug
}

// ensureUniqueSlug - slugning unique ekanligini ta'minlash
func ensureUniqueSlug(db *sql.DB, baseSlug string, excludeID string) string {
	slug := baseSlug
	counter := 1
	
	for {
		var exists bool
		query := "SELECT EXISTS(SELECT 1 FROM categories WHERE slug = $1"
		args := []interface{}{slug}
		
		if excludeID != "" {
			query += " AND id != $2"
			args = append(args, excludeID)
		}
		query += ")"
		
		err := db.QueryRow(query, args...).Scan(&exists)
		if err != nil || !exists {
			break
		}
		
		slug = fmt.Sprintf("%s-%d", baseSlug, counter)
		counter++
	}
	
	return slug
}

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
	// Barcha kategoriyalarni olish (faqat faol kategoriyalar)
	query := `
		SELECT 
			c.id, c.parent_id, c.name, COALESCE(c.slug, ''), COALESCE(c.icon_url, ''),
			COALESCE(c.is_active, true), COALESCE(c.sort_order, 0),
			(SELECT COUNT(*) FROM products p WHERE p.category_id = c.id AND p.is_active = true) as product_count
		FROM categories c
		WHERE c.is_active = true
		ORDER BY c.sort_order ASC, c.name ASC
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
		err := rows.Scan(&c.ID, &c.ParentID, &c.Name, &c.Slug, &c.IconURL, &c.IsActive, &c.SortOrder, &c.ProductCount)
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

// getFlatCategories - barcha kategoriyalarni tekis ro'yxatda olish (faqat faol kategoriyalar)
func getFlatCategories(db *sql.DB, w http.ResponseWriter) {
	query := `
		SELECT 
			c.id, c.parent_id, c.name, COALESCE(c.slug, ''), COALESCE(c.icon_url, ''),
			COALESCE(c.is_active, true), COALESCE(c.sort_order, 0),
			(SELECT COUNT(*) FROM products p WHERE p.category_id = c.id AND p.is_active = true) as product_count
		FROM categories c
		WHERE c.is_active = true
		ORDER BY c.sort_order ASC, c.parent_id NULLS FIRST, c.name ASC
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
		err := rows.Scan(&c.ID, &c.ParentID, &c.Name, &c.Slug, &c.IconURL, &c.IsActive, &c.SortOrder, &c.ProductCount)
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

// GetAdminCategories godoc
// @Summary      Barcha kategoriyalarni olish (Admin uchun - faol va nofaol)
// @Description  Admin panel uchun barcha kategoriyalarni tekis ro'yxatda qaytaradi (is_active filter yo'q)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.CategoriesResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/categories/list [get]
func GetAdminCategories(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
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

		// Barcha kategoriyalarni olish (faol va nofaol)
		query := `
			SELECT 
				c.id, c.parent_id, c.name, COALESCE(c.slug, ''), COALESCE(c.icon_url, ''),
				COALESCE(c.is_active, true), COALESCE(c.sort_order, 0),
				(SELECT COUNT(*) FROM products p WHERE p.category_id = c.id AND p.is_active = true) as product_count
			FROM categories c
			ORDER BY c.sort_order ASC, c.created_at DESC, c.name ASC
		`

		rows, err := db.Query(query)
		if err != nil {
			log.Printf("AdminCategories query xatosi: %v", err)
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
			err := rows.Scan(&c.ID, &c.ParentID, &c.Name, &c.Slug, &c.IconURL, &c.IsActive, &c.SortOrder, &c.ProductCount)
			if err != nil {
				log.Printf("Category scan xatosi: %v", err)
				continue
			}
			categories = append(categories, c)
		}

		log.Printf("✅ %d ta kategoriya topildi (admin - barcha)", len(categories))

		writeJSON(w, http.StatusOK, models.CategoriesResponse{
			Success:    true,
			Categories: categories,
			Count:      len(categories),
		})
	}
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
			SELECT id, parent_id, name, COALESCE(slug, ''), COALESCE(icon_url, ''),
			COALESCE(is_active, true), COALESCE(sort_order, 0),
			(SELECT COUNT(*) FROM products p WHERE p.category_id = c.id AND p.is_active = true) as product_count
			FROM categories c
			WHERE id = $1
		`

		var c models.Category
		err := db.QueryRow(query, categoryID).Scan(&c.ID, &c.ParentID, &c.Name, &c.Slug, &c.IconURL, &c.IsActive, &c.SortOrder, &c.ProductCount)

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
			SELECT id, parent_id, name, COALESCE(slug, ''), COALESCE(icon_url, ''),
			COALESCE(is_active, true), COALESCE(sort_order, 0),
			(SELECT COUNT(*) FROM products p WHERE p.category_id = sc.id AND p.is_active = true) as product_count
			FROM categories sc
			WHERE parent_id = $1
			ORDER BY sort_order ASC, name ASC
		`

		rows, err := db.Query(subQuery, categoryID)
		if err == nil {
			defer rows.Close()
			c.SubCategories = []models.Category{}
			for rows.Next() {
				var sub models.Category
				if err := rows.Scan(&sub.ID, &sub.ParentID, &sub.Name, &sub.Slug, &sub.IconURL, &sub.IsActive, &sub.SortOrder, &sub.ProductCount); err == nil {
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
// @Param        is_active formData boolean false "Kategoriya faolligi (default: true)"
// @Param        sort_order formData integer false "Tartib raqami (default: 0)"
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

		// Slug yaratish (name dan avtomatik)
		baseSlug := generateSlug(name)
		slug := ensureUniqueSlug(db, baseSlug, "")

		// is_active olish (default: true)
		isActive := true
		if isActiveStr := r.FormValue("is_active"); isActiveStr != "" {
			if parsed, err := strconv.ParseBool(isActiveStr); err == nil {
				isActive = parsed
			}
		}

		// sort_order olish (default: 0)
		sortOrder := 0
		if sortOrderStr := r.FormValue("sort_order"); sortOrderStr != "" {
			if parsed, err := strconv.Atoi(sortOrderStr); err == nil {
				sortOrder = parsed
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
			INSERT INTO categories (id, name, slug, icon_url, parent_id, is_active, sort_order, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
			RETURNING id
		`

		var insertedID string
		err = db.QueryRow(query, categoryID, name, slug, iconURL, parentID, isActive, sortOrder).Scan(&insertedID)
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
			SELECT id, parent_id, name, COALESCE(slug, ''), COALESCE(icon_url, ''),
			COALESCE(is_active, true), COALESCE(sort_order, 0),
			(SELECT COUNT(*) FROM products p WHERE p.category_id = c.id AND p.is_active = true) as product_count
			FROM categories c
			WHERE id = $1
		`, insertedID).Scan(&c.ID, &c.ParentID, &c.Name, &c.Slug, &c.IconURL, &c.IsActive, &c.SortOrder, &c.ProductCount)

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
// @Param        is_active formData boolean false "Kategoriya faolligi"
// @Param        sort_order formData integer false "Tartib raqami"
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
		var slug *string
		if nameStr := strings.TrimSpace(r.FormValue("name")); nameStr != "" {
			name = &nameStr
			// Agar name o'zgarsa, slug ham yangilanadi
			baseSlug := generateSlug(nameStr)
			newSlug := ensureUniqueSlug(db, baseSlug, categoryID)
			slug = &newSlug
		}

		// is_active olish (optional)
		var isActive *bool
		if isActiveStr := r.FormValue("is_active"); isActiveStr != "" {
			if parsed, err := strconv.ParseBool(isActiveStr); err == nil {
				isActive = &parsed
			}
		}

		// sort_order olish (optional)
		var sortOrder *int
		if sortOrderStr := r.FormValue("sort_order"); sortOrderStr != "" {
			if parsed, err := strconv.Atoi(sortOrderStr); err == nil {
				sortOrder = &parsed
			}
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
		if name == nil && iconURL == nil && isActive == nil && sortOrder == nil {
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
			
			// Agar name o'zgarsa, slug ham yangilanadi
			if slug != nil {
				updateFields = append(updateFields, fmt.Sprintf("slug = $%d", argIndex))
				args = append(args, *slug)
				argIndex++
			}
		}

		if iconURL != nil {
			updateFields = append(updateFields, fmt.Sprintf("icon_url = $%d", argIndex))
			args = append(args, *iconURL)
			argIndex++
		}

		if isActive != nil {
			updateFields = append(updateFields, fmt.Sprintf("is_active = $%d", argIndex))
			args = append(args, *isActive)
			argIndex++
		}

		if sortOrder != nil {
			updateFields = append(updateFields, fmt.Sprintf("sort_order = $%d", argIndex))
			args = append(args, *sortOrder)
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
			SELECT id, parent_id, name, COALESCE(slug, ''), COALESCE(icon_url, ''),
			COALESCE(is_active, true), COALESCE(sort_order, 0),
			(SELECT COUNT(*) FROM products p WHERE p.category_id = c.id AND p.is_active = true) as product_count
			FROM categories c
			WHERE id = $1
		`, categoryID).Scan(&c.ID, &c.ParentID, &c.Name, &c.Slug, &c.IconURL, &c.IsActive, &c.SortOrder, &c.ProductCount)

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
