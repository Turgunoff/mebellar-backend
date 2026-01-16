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
	"strconv"
	"strings"
	"time"

	"mebellar-backend/models"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// GetProducts godoc
// @Summary      Barcha mahsulotlarni olish
// @Description  Mahsulotlar ro'yxatini qaytaradi. ?category_id= parametri bilan filtrlash mumkin
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        category_id query string false "Kategoriya ID bo'yicha filter"
// @Param        is_new query bool false "Faqat yangi mahsulotlar"
// @Param        is_popular query bool false "Faqat mashhur mahsulotlar"
// @Success      200  {object}  models.ProductsResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /products [get]
func GetProducts(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Query parametrlarini olish
		categoryID := r.URL.Query().Get("category_id")
		isNew := r.URL.Query().Get("is_new")
		isPopular := r.URL.Query().Get("is_popular")

		// SQL so'rov yaratish
		query := `
			SELECT 
				id, category_id, name, description, price, discount_price,
				COALESCE(images, '{}'), COALESCE(specs::text, '{}')::jsonb, 
				COALESCE(variants::text, '[]')::jsonb,
				rating, is_new, is_popular, is_active, created_at
			FROM products 
			WHERE is_active = true
		`

		var args []interface{}
		argIndex := 1

		// Kategoriya filtri
		if categoryID != "" {
			query += ` AND category_id = $` + string(rune('0'+argIndex))
			args = append(args, categoryID)
			argIndex++
		}

		// Yangi mahsulotlar filtri
		if isNew == "true" {
			query += ` AND is_new = true`
		}

		// Mashhur mahsulotlar filtri
		if isPopular == "true" {
			query += ` AND is_popular = true`
		}

		query += ` ORDER BY created_at DESC`

		log.Printf("📦 Products query: %s", query)

		rows, err := db.Query(query, args...)
		if err != nil {
			log.Printf("Products query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Mahsulotlarni olishda xatolik",
			})
			return
		}
		defer rows.Close()

		products := []models.Product{}
		for rows.Next() {
			var p models.Product
			err := rows.Scan(
				&p.ID, &p.CategoryID, &p.Name, &p.Description, &p.Price, &p.DiscountPrice,
				&p.Images, &p.Specs, &p.Variants,
				&p.Rating, &p.IsNew, &p.IsPopular, &p.IsActive, &p.CreatedAt,
			)
			if err != nil {
				log.Printf("Product scan xatosi: %v", err)
				continue
			}
			products = append(products, p)
		}

		log.Printf("✅ %d ta mahsulot topildi", len(products))

		writeJSON(w, http.StatusOK, models.ProductsResponse{
			Success:  true,
			Products: products,
			Count:    len(products),
		})
	}
}

// GetProductByID godoc
// @Summary      Mahsulot ma'lumotlarini olish
// @Description  ID bo'yicha mahsulot to'liq ma'lumotlarini qaytaradi
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        id path string true "Mahsulot ID"
// @Success      200  {object}  models.ProductResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /products/{id} [get]
func GetProductByID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// URL dan ID olish: /api/products/123 -> 123
		path := r.URL.Path
		parts := strings.Split(path, "/")
		if len(parts) < 4 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Mahsulot ID kiritilmagan",
			})
			return
		}
		productID := parts[len(parts)-1]

		if productID == "" || productID == "new" || productID == "popular" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri mahsulot ID",
			})
			return
		}

		query := `
			SELECT 
				id, category_id, name, description, price, discount_price,
				COALESCE(images, '{}'), COALESCE(specs::text, '{}')::jsonb, 
				COALESCE(variants::text, '[]')::jsonb,
				rating, is_new, is_popular, is_active, created_at
			FROM products 
			WHERE id = $1 AND is_active = true
		`

		var p models.Product
		err := db.QueryRow(query, productID).Scan(
			&p.ID, &p.CategoryID, &p.Name, &p.Description, &p.Price, &p.DiscountPrice,
			&p.Images, &p.Specs, &p.Variants,
			&p.Rating, &p.IsNew, &p.IsPopular, &p.IsActive, &p.CreatedAt,
		)

		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Mahsulot topilmadi",
			})
			return
		}

		if err != nil {
			log.Printf("Product query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Mahsulotni olishda xatolik",
			})
			return
		}

		log.Printf("✅ Mahsulot topildi: %s", p.Name)

		writeJSON(w, http.StatusOK, models.ProductResponse{
			Success: true,
			Product: &p,
		})
	}
}

// GetNewArrivals godoc
// @Summary      Yangi mahsulotlarni olish
// @Description  Yangi (is_new=true) mahsulotlarni qaytaradi
// @Tags         products
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.ProductsResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /products/new [get]
func GetNewArrivals(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		query := `
			SELECT 
				id, category_id, name, description, price, discount_price,
				COALESCE(images, '{}'), COALESCE(specs::text, '{}')::jsonb, 
				COALESCE(variants::text, '[]')::jsonb,
				rating, is_new, is_popular, is_active, created_at
			FROM products 
			WHERE is_active = true AND is_new = true
			ORDER BY created_at DESC
			LIMIT 10
		`

		rows, err := db.Query(query)
		if err != nil {
			log.Printf("New arrivals query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Mahsulotlarni olishda xatolik",
			})
			return
		}
		defer rows.Close()

		products := []models.Product{}
		for rows.Next() {
			var p models.Product
			err := rows.Scan(
				&p.ID, &p.CategoryID, &p.Name, &p.Description, &p.Price, &p.DiscountPrice,
				&p.Images, &p.Specs, &p.Variants,
				&p.Rating, &p.IsNew, &p.IsPopular, &p.IsActive, &p.CreatedAt,
			)
			if err != nil {
				log.Printf("Product scan xatosi: %v", err)
				continue
			}
			products = append(products, p)
		}

		log.Printf("✅ %d ta yangi mahsulot topildi", len(products))

		writeJSON(w, http.StatusOK, models.ProductsResponse{
			Success:  true,
			Message:  "Yangi mahsulotlar",
			Products: products,
			Count:    len(products),
		})
	}
}

// GetPopularProducts godoc
// @Summary      Mashhur mahsulotlarni olish
// @Description  Mashhur (is_popular=true) mahsulotlarni qaytaradi
// @Tags         products
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.ProductsResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /products/popular [get]
func GetPopularProducts(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		query := `
			SELECT 
				id, category_id, name, description, price, discount_price,
				COALESCE(images, '{}'), COALESCE(specs::text, '{}')::jsonb, 
				COALESCE(variants::text, '[]')::jsonb,
				rating, is_new, is_popular, is_active, created_at
			FROM products 
			WHERE is_active = true AND is_popular = true
			ORDER BY rating DESC
			LIMIT 10
		`

		rows, err := db.Query(query)
		if err != nil {
			log.Printf("Popular products query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Mahsulotlarni olishda xatolik",
			})
			return
		}
		defer rows.Close()

		products := []models.Product{}
		for rows.Next() {
			var p models.Product
			err := rows.Scan(
				&p.ID, &p.CategoryID, &p.Name, &p.Description, &p.Price, &p.DiscountPrice,
				&p.Images, &p.Specs, &p.Variants,
				&p.Rating, &p.IsNew, &p.IsPopular, &p.IsActive, &p.CreatedAt,
			)
			if err != nil {
				log.Printf("Product scan xatosi: %v", err)
				continue
			}
			products = append(products, p)
		}

		log.Printf("✅ %d ta mashhur mahsulot topildi", len(products))

		writeJSON(w, http.StatusOK, models.ProductsResponse{
			Success:  true,
			Message:  "Mashhur mahsulotlar",
			Products: products,
			Count:    len(products),
		})
	}
}

// CreateProduct godoc
// @Summary      Yangi mahsulot yaratish (Seller)
// @Description  Seller uchun yangi mahsulot qo'shish. Multipart form-data kerak.
// @Tags         seller-products
// @Accept       multipart/form-data
// @Produce      json
// @Param        X-Shop-ID header string true "Do'kon ID"
// @Param        name formData string true "Mahsulot nomi"
// @Param        description formData string false "Tavsif"
// @Param        category_id formData string false "Kategoriya ID"
// @Param        price formData number true "Narx"
// @Param        discount_price formData number false "Chegirma narxi"
// @Param        is_new formData bool false "Yangi mahsulot belgisi"
// @Param        is_popular formData bool false "Mashhur mahsulot belgisi"
// @Param        specs formData string false "Xususiyatlar (JSON)"
// @Param        variants formData string false "Variantlar (JSON)"
// @Param        delivery_settings formData string false "Yetkazib berish sozlamalari (JSON)"
// @Param        images formData file false "Mahsulot rasmlari"
// @Success      201  {object}  models.ProductResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      401  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /seller/products [post]
func CreateProduct(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Get shop ID from header
		shopID := r.Header.Get("X-Shop-ID")
		if shopID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "X-Shop-ID header kerak",
			})
			return
		}

		// Parse multipart form (32MB max)
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			log.Printf("ParseMultipartForm xatosi: %v", err)
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Form ma'lumotlarini o'qishda xatolik",
			})
			return
		}

		// Get form values
		name := r.FormValue("name")
		description := r.FormValue("description")
		categoryID := r.FormValue("category_id")
		priceStr := r.FormValue("price")
		discountPriceStr := r.FormValue("discount_price")
		isNewStr := r.FormValue("is_new")
		isPopularStr := r.FormValue("is_popular")
		specsJSON := r.FormValue("specs")
		variantsJSON := r.FormValue("variants")
		deliverySettingsJSON := r.FormValue("delivery_settings")

		// Validate required fields
		if name == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Mahsulot nomi kerak",
			})
			return
		}

		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || price <= 0 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Narx noto'g'ri",
			})
			return
		}

		// Validate price range (DECIMAL(15,2) max: 9999999999999.99)
		const maxPrice = 9999999999999.99
		if price > maxPrice {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: fmt.Sprintf("Narx juda katta. Maksimal qiymat: %.2f", maxPrice),
			})
			return
		}

		// Parse optional fields
		var discountPrice *float64
		if discountPriceStr != "" {
			dp, err := strconv.ParseFloat(discountPriceStr, 64)
			if err == nil && dp > 0 {
				// Validate discount price range
				if dp > maxPrice {
					writeJSON(w, http.StatusBadRequest, models.AuthResponse{
						Success: false,
						Message: fmt.Sprintf("Chegirma narxi juda katta. Maksimal qiymat: %.2f", maxPrice),
					})
					return
				}
				// Discount price should be less than regular price
				if dp >= price {
					writeJSON(w, http.StatusBadRequest, models.AuthResponse{
						Success: false,
						Message: "Chegirma narxi oddiy narxdan kichik bo'lishi kerak",
					})
					return
				}
				discountPrice = &dp
			}
		}

		isNew := isNewStr == "true"
		isPopular := isPopularStr == "true"

		// Parse JSON fields
		var specs models.JSONB
		if specsJSON != "" {
			if err := json.Unmarshal([]byte(specsJSON), &specs); err != nil {
				log.Printf("Specs JSON parse xatosi: %v", err)
				specs = models.JSONB{}
			}
		}

		var variants models.JSONBArray
		if variantsJSON != "" {
			if err := json.Unmarshal([]byte(variantsJSON), &variants); err != nil {
				log.Printf("Variants JSON parse xatosi: %v", err)
				variants = models.JSONBArray{}
			}
		}

		var deliverySettings models.DeliverySettings
		if deliverySettingsJSON != "" {
			if err := json.Unmarshal([]byte(deliverySettingsJSON), &deliverySettings); err != nil {
				log.Printf("DeliverySettings JSON parse xatosi: %v", err)
				deliverySettings = models.DeliverySettings{
					Default: models.RegionSettings{DeliveryDays: "3-5"},
				}
			} else {
				// Validate delivery settings prices (DECIMAL(15,2) max: 9999999999999.99)
				const maxDeliveryPrice = 9999999999999.99
				if deliverySettings.Default.DeliveryPrice > maxDeliveryPrice {
					writeJSON(w, http.StatusBadRequest, models.AuthResponse{
						Success: false,
						Message: fmt.Sprintf("Yetkazib berish narxi juda katta. Maksimal: %.2f", maxDeliveryPrice),
					})
					return
				}
				if deliverySettings.Default.InstallationPrice > maxDeliveryPrice {
					writeJSON(w, http.StatusBadRequest, models.AuthResponse{
						Success: false,
						Message: fmt.Sprintf("O'rnatish narxi juda katta. Maksimal: %.2f", maxDeliveryPrice),
					})
					return
				}
				// Validate overrides
				for _, override := range deliverySettings.Overrides {
					if override.DeliveryPrice > maxDeliveryPrice {
						writeJSON(w, http.StatusBadRequest, models.AuthResponse{
							Success: false,
							Message: fmt.Sprintf("Hudud uchun yetkazib berish narxi juda katta. Maksimal: %.2f", maxDeliveryPrice),
						})
						return
					}
					if override.InstallationPrice > maxDeliveryPrice {
						writeJSON(w, http.StatusBadRequest, models.AuthResponse{
							Success: false,
							Message: fmt.Sprintf("Hudud uchun o'rnatish narxi juda katta. Maksimal: %.2f", maxDeliveryPrice),
						})
						return
					}
				}
			}
		}

		// Handle image uploads
		var imageURLs []string
		files := r.MultipartForm.File["images"]
		if len(files) > 0 {
			// Create uploads directory if not exists
			uploadDir := "./uploads/products"
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				log.Printf("Upload dir yaratishda xatolik: %v", err)
			}

			for i, fileHeader := range files {
				if i >= 5 { // Max 5 images
					break
				}

				file, err := fileHeader.Open()
				if err != nil {
					log.Printf("File ochishda xatolik: %v", err)
					continue
				}
				defer file.Close()

				// Generate unique filename
				ext := filepath.Ext(fileHeader.Filename)
				if ext == "" {
					ext = ".jpg"
				}
				filename := fmt.Sprintf("%s_%d%s", uuid.New().String(), i, ext)
				filePath := filepath.Join(uploadDir, filename)

				// Save file
				dst, err := os.Create(filePath)
				if err != nil {
					log.Printf("File yaratishda xatolik: %v", err)
					continue
				}
				defer dst.Close()

				if _, err := io.Copy(dst, file); err != nil {
					log.Printf("File saqlashda xatolik: %v", err)
					continue
				}

				// Add URL to list (adjust based on your server config)
				imageURL := fmt.Sprintf("/uploads/products/%s", filename)
				imageURLs = append(imageURLs, imageURL)
			}
		}

		// Generate product ID
		productID := uuid.New().String()

		// Insert into database
		query := `
			INSERT INTO products (
				id, shop_id, category_id, name, description, price, discount_price,
				images, specs, variants, delivery_settings,
				is_new, is_popular, is_active, created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7,
				$8, $9, $10, $11,
				$12, $13, true, $14
			)
			RETURNING id
		`

		var categoryIDPtr *string
		if categoryID != "" {
			categoryIDPtr = &categoryID
		}

		specsValue, _ := specs.Value()
		variantsValue, _ := variants.Value()
		deliveryValue, _ := deliverySettings.Value()

		var insertedID string
		err = db.QueryRow(
			query,
			productID,
			shopID,
			categoryIDPtr,
			name,
			description,
			price,
			discountPrice,
			fmt.Sprintf("{%s}", strings.Join(quoteStrings(imageURLs), ",")),
			specsValue,
			variantsValue,
			deliveryValue,
			isNew,
			isPopular,
			time.Now(),
		).Scan(&insertedID)

		if err != nil {
			log.Printf("❌ Product insert xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Mahsulot yaratishda xatolik: " + err.Error(),
			})
			return
		}

		log.Printf("✅ Mahsulot yaratildi: %s - %s", insertedID, name)

		// Return created product
		product := models.Product{
			ID:               insertedID,
			CategoryID:       categoryIDPtr,
			Name:             name,
			Description:      description,
			Price:            price,
			DiscountPrice:    discountPrice,
			Images:           imageURLs,
			Specs:            specs,
			Variants:         variants,
			DeliverySettings: deliverySettings,
			IsNew:            isNew,
			IsPopular:        isPopular,
			IsActive:         true,
			CreatedAt:        time.Now(),
		}

		writeJSON(w, http.StatusCreated, models.ProductResponse{
			Success: true,
			Message: "Mahsulot muvaffaqiyatli yaratildi",
			Product: &product,
		})
	}
}

// quoteStrings wraps each string in quotes for PostgreSQL array
func quoteStrings(strs []string) []string {
	result := make([]string, len(strs))
	for i, s := range strs {
		result[i] = fmt.Sprintf("\"%s\"", s)
	}
	return result
}

// SellerProductsResponse - Seller mahsulotlari javobi (pagination bilan)
type SellerProductsResponse struct {
	Success  bool             `json:"success"`
	Message  string           `json:"message,omitempty"`
	Products []models.Product `json:"products"`
	Total    int              `json:"total"`
	Page     int              `json:"page"`
	Limit    int              `json:"limit"`
}

// GetSellerProducts godoc
// @Summary      Seller mahsulotlarini olish
// @Description  Joriy do'konning mahsulotlari ro'yxatini qaytaradi (pagination bilan)
// @Tags         seller-products
// @Accept       json
// @Produce      json
// @Param        X-Shop-ID header string true "Do'kon ID"
// @Param        page query int false "Sahifa raqami (default: 1)"
// @Param        limit query int false "Har sahifadagi mahsulotlar soni (default: 10)"
// @Param        is_active query bool false "Faol/Nofaol filtr"
// @Success      200  {object}  SellerProductsResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      401  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /seller/products [get]
func GetSellerProducts(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Get shop ID from header
		shopID := r.Header.Get("X-Shop-ID")
		if shopID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "X-Shop-ID header kerak",
			})
			return
		}

		// Parse query params
		pageStr := r.URL.Query().Get("page")
		limitStr := r.URL.Query().Get("limit")
		sortBy := r.URL.Query().Get("sort")
		filterBy := r.URL.Query().Get("filter")

		page := 1
		limit := 10

		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}

		offset := (page - 1) * limit

		// Build query with analytics columns
		countQuery := `SELECT COUNT(*) FROM products WHERE shop_id = $1`
		dataQuery := `
			SELECT 
				id, category_id, name, description, price, discount_price,
				COALESCE(images, '{}'), COALESCE(specs::text, '{}')::jsonb, 
				COALESCE(variants::text, '[]')::jsonb,
				COALESCE(delivery_settings::text, '{}')::jsonb,
				rating, is_new, is_popular, is_active, created_at,
				COALESCE(view_count, 0), COALESCE(sold_count, 0)
			FROM products 
			WHERE shop_id = $1
		`

		args := []interface{}{shopID}
		argIndex := 2

		// Apply filter
		switch filterBy {
		case "active":
			countQuery += " AND is_active = true"
			dataQuery += " AND is_active = true"
		case "inactive":
			countQuery += " AND is_active = false"
			dataQuery += " AND is_active = false"
		case "discount":
			countQuery += " AND discount_price IS NOT NULL AND discount_price > 0"
			dataQuery += " AND discount_price IS NOT NULL AND discount_price > 0"
		}

		// Apply sorting
		var orderBy string
		switch sortBy {
		case "price_asc":
			orderBy = " ORDER BY price ASC"
		case "price_desc":
			orderBy = " ORDER BY price DESC"
		case "popular":
			orderBy = " ORDER BY COALESCE(view_count, 0) DESC"
		case "bestseller":
			orderBy = " ORDER BY COALESCE(sold_count, 0) DESC"
		case "rating":
			orderBy = " ORDER BY rating DESC"
		default: // "newest" or empty
			orderBy = " ORDER BY created_at DESC"
		}
		dataQuery += orderBy
		dataQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)

		// Get total count
		var total int
		err := db.QueryRow(countQuery, args...).Scan(&total)
		if err != nil {
			log.Printf("❌ Products count xatosi: %v", err)
			total = 0
		}

		// Get products
		args = append(args, limit, offset)
		rows, err := db.Query(dataQuery, args...)
		if err != nil {
			log.Printf("❌ Seller products query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Mahsulotlarni olishda xatolik",
			})
			return
		}
		defer rows.Close()

		products := []models.Product{}
		for rows.Next() {
			var p models.Product
			err := rows.Scan(
				&p.ID, &p.CategoryID, &p.Name, &p.Description, &p.Price, &p.DiscountPrice,
				&p.Images, &p.Specs, &p.Variants, &p.DeliverySettings,
				&p.Rating, &p.IsNew, &p.IsPopular, &p.IsActive, &p.CreatedAt,
				&p.ViewCount, &p.SoldCount,
			)
			if err != nil {
				log.Printf("Product scan xatosi: %v", err)
				continue
			}
			products = append(products, p)
		}

		log.Printf("✅ Seller %s: %d ta mahsulot topildi (sahifa %d, sort=%s, filter=%s)", 
			shopID, len(products), page, sortBy, filterBy)

		writeJSON(w, http.StatusOK, SellerProductsResponse{
			Success:  true,
			Products: products,
			Total:    total,
			Page:     page,
			Limit:    limit,
		})
	}
}

// SellerProductsHandler - GET va POST uchun handler
func SellerProductsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			GetSellerProducts(db)(w, r)
		case http.MethodPost:
			CreateProduct(db)(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET yoki POST metodi qo'llab-quvvatlanadi",
			})
		}
	}
}

// UpdateProduct godoc
// @Summary      Mahsulotni yangilash
// @Description  Mavjud mahsulot ma'lumotlarini yangilash
// @Tags         seller-products
// @Accept       multipart/form-data
// @Produce      json
// @Param        id path string true "Mahsulot ID"
// @Param        X-Shop-ID header string true "Do'kon ID"
// @Success      200  {object}  models.ProductResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /seller/products/{id} [put]
func UpdateProduct(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat PUT metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Get product ID from URL path
		path := r.URL.Path
		parts := strings.Split(path, "/")
		if len(parts) < 4 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Mahsulot ID kerak",
			})
			return
		}
		productID := parts[len(parts)-1]

		// Validate UUID
		if _, err := uuid.Parse(productID); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri mahsulot ID formati",
			})
			return
		}

		// Get shop ID from header
		shopID := r.Header.Get("X-Shop-ID")
		if shopID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "X-Shop-ID header kerak",
			})
			return
		}

		// Check if product belongs to this shop
		var existingShopID string
		var existingImages pq.StringArray
		err := db.QueryRow(
			"SELECT COALESCE(shop_id::text, ''), COALESCE(images, '{}') FROM products WHERE id = $1",
			productID,
		).Scan(&existingShopID, &existingImages)

		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Mahsulot topilmadi",
			})
			return
		}
		if err != nil {
			log.Printf("❌ Product check xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Mahsulotni tekshirishda xatolik",
			})
			return
		}

		// Security check: product must belong to this shop
		if existingShopID != shopID {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Bu mahsulot sizga tegishli emas",
			})
			return
		}

		// Parse multipart form
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Form ma'lumotlarini o'qishda xatolik: " + err.Error(),
			})
			return
		}

		// Get form values
		name := strings.TrimSpace(r.FormValue("name"))
		description := strings.TrimSpace(r.FormValue("description"))
		priceStr := r.FormValue("price")
		discountPriceStr := r.FormValue("discount_price")
		categoryID := r.FormValue("category_id")
		specsJSON := r.FormValue("specs")
		variantsJSON := r.FormValue("variants")
		deliverySettingsJSON := r.FormValue("delivery_settings")
		isNewStr := r.FormValue("is_new")
		isPopularStr := r.FormValue("is_popular")
		isActiveStr := r.FormValue("is_active")
		keepExistingImagesStr := r.FormValue("keep_existing_images") // "true" or "false"

		// Validate required fields
		if name == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Mahsulot nomi kiritilishi shart",
			})
			return
		}

		// Parse price
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || price <= 0 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Narx noto'g'ri formatda",
			})
			return
		}

		// Parse discount price
		var discountPrice float64
		if discountPriceStr != "" {
			discountPrice, _ = strconv.ParseFloat(discountPriceStr, 64)
		}

		// Category ID (optional for update)
		var categoryIDPtr *string
		if categoryID != "" {
			categoryIDPtr = &categoryID
		}

		// Parse specs
		specs := make(map[string]interface{})
		if specsJSON != "" {
			if err := json.Unmarshal([]byte(specsJSON), &specs); err != nil {
				log.Printf("⚠️ Specs parse xatosi: %v", err)
			}
		}

		// Parse variants
		variants := []map[string]interface{}{}
		if variantsJSON != "" {
			if err := json.Unmarshal([]byte(variantsJSON), &variants); err != nil {
				log.Printf("⚠️ Variants parse xatosi: %v", err)
			}
		}

		// Parse delivery settings
		deliverySettings := models.DeliverySettings{}
		if deliverySettingsJSON != "" {
			if err := json.Unmarshal([]byte(deliverySettingsJSON), &deliverySettings); err != nil {
				log.Printf("⚠️ DeliverySettings parse xatosi: %v", err)
			}
		}

		// Boolean flags
		isNew := isNewStr == "true"
		isPopular := isPopularStr == "true"
		isActive := isActiveStr != "false" // Default: keep active (preserve existing state)
		keepExistingImages := keepExistingImagesStr != "false" // Default: keep existing

		// Handle images
		imageURLs := []string{}

		// Keep existing images if requested
		if keepExistingImages {
			imageURLs = append(imageURLs, existingImages...)
		}

		// Handle new image uploads
		files := r.MultipartForm.File["images"]
		if len(files) > 0 {
			uploadDir := "./uploads/products"
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				log.Printf("❌ Upload dir yaratishda xatolik: %v", err)
			}

			for i, fileHeader := range files {
				file, err := fileHeader.Open()
				if err != nil {
					log.Printf("⚠️ File open xatosi: %v", err)
					continue
				}
				defer file.Close()

				// Generate unique filename
				ext := filepath.Ext(fileHeader.Filename)
				if ext == "" {
					ext = ".jpg"
				}
				newFilename := fmt.Sprintf("%s_%d%s", uuid.New().String(), i, ext)
				filePath := filepath.Join(uploadDir, newFilename)

				// Save file
				dst, err := os.Create(filePath)
				if err != nil {
					log.Printf("⚠️ File create xatosi: %v", err)
					continue
				}
				defer dst.Close()

				if _, err := io.Copy(dst, file); err != nil {
					log.Printf("⚠️ File copy xatosi: %v", err)
					continue
				}

				imageURLs = append(imageURLs, "/uploads/products/"+newFilename)
				log.Printf("✅ Yangi rasm saqlandi: %s", newFilename)
			}
		}

		// Convert specs and variants to JSONB values
		specsValue, _ := json.Marshal(specs)
		variantsValue, _ := json.Marshal(variants)
		deliveryValue, _ := json.Marshal(deliverySettings)

		// Update product (including is_active to preserve state)
		query := `
			UPDATE products SET
				name = $1,
				description = $2,
				price = $3,
				discount_price = $4,
				category_id = $5,
				images = $6,
				specs = $7,
				variants = $8,
				delivery_settings = $9,
				is_new = $10,
				is_popular = $11,
				is_active = $12
			WHERE id = $13
			RETURNING id
		`

		var updatedID string
		err = db.QueryRow(
			query,
			name,
			description,
			price,
			discountPrice,
			categoryIDPtr,
			fmt.Sprintf("{%s}", strings.Join(quoteStrings(imageURLs), ",")),
			specsValue,
			variantsValue,
			deliveryValue,
			isNew,
			isPopular,
			isActive,
			productID,
		).Scan(&updatedID)

		if err != nil {
			log.Printf("❌ Product update xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Mahsulotni yangilashda xatolik: " + err.Error(),
			})
			return
		}

		log.Printf("✅ Mahsulot yangilandi: %s - %s", updatedID, name)

		// Return updated product
		var discountPricePtr *float64
		if discountPrice > 0 {
			discountPricePtr = &discountPrice
		}

		product := models.Product{
			ID:               updatedID,
			CategoryID:       categoryIDPtr,
			Name:             name,
			Description:      description,
			Price:            price,
			DiscountPrice:    discountPricePtr,
			Images:           pq.StringArray(imageURLs),
			Specs:            specs,
			Variants:         variants,
			DeliverySettings: deliverySettings,
			IsNew:            isNew,
			IsPopular:        isPopular,
			IsActive:         isActive, // Preserve the actual state
		}

		writeJSON(w, http.StatusOK, models.ProductResponse{
			Success: true,
			Message: "Mahsulot muvaffaqiyatli yangilandi",
			Product: &product,
		})
	}
}

// DeleteProduct godoc
// @Summary      Mahsulotni o'chirish
// @Description  Mahsulotni bazadan o'chirish (hard delete)
// @Tags         seller-products
// @Produce      json
// @Param        id path string true "Mahsulot ID"
// @Param        X-Shop-ID header string true "Do'kon ID"
// @Success      200  {object}  models.AuthResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /seller/products/{id} [delete]
func DeleteProduct(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat DELETE metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Get product ID from URL path
		path := r.URL.Path
		parts := strings.Split(path, "/")
		if len(parts) < 4 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Mahsulot ID kerak",
			})
			return
		}
		productID := parts[len(parts)-1]

		// Validate UUID
		if _, err := uuid.Parse(productID); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri mahsulot ID formati",
			})
			return
		}

		// Get shop ID from header
		shopID := r.Header.Get("X-Shop-ID")
		if shopID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "X-Shop-ID header kerak",
			})
			return
		}

		// Check if product belongs to this shop and get images
		var existingShopID string
		var existingImages pq.StringArray
		err := db.QueryRow(
			"SELECT COALESCE(shop_id::text, ''), COALESCE(images, '{}') FROM products WHERE id = $1",
			productID,
		).Scan(&existingShopID, &existingImages)

		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Mahsulot topilmadi",
			})
			return
		}
		if err != nil {
			log.Printf("❌ Product check xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Mahsulotni tekshirishda xatolik",
			})
			return
		}

		// Security check: product must belong to this shop
		if existingShopID != shopID {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Bu mahsulot sizga tegishli emas",
			})
			return
		}

		// Delete product from database
		result, err := db.Exec("DELETE FROM products WHERE id = $1", productID)
		if err != nil {
			log.Printf("❌ Product delete xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Mahsulotni o'chirishda xatolik: " + err.Error(),
			})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Mahsulot topilmadi yoki allaqachon o'chirilgan",
			})
			return
		}

		// Delete associated images from disk
		for _, imgPath := range existingImages {
			if strings.HasPrefix(imgPath, "/uploads/") {
				fullPath := "." + imgPath
				if err := os.Remove(fullPath); err != nil {
					log.Printf("⚠️ Rasm o'chirishda xatolik: %s - %v", fullPath, err)
				} else {
					log.Printf("✅ Rasm o'chirildi: %s", fullPath)
				}
			}
		}

		log.Printf("✅ Mahsulot o'chirildi: %s", productID)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: "Mahsulot muvaffaqiyatli o'chirildi",
		})
	}
}

// SellerProductItemHandler - PUT va DELETE uchun handler (/api/seller/products/{id})
func SellerProductItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			UpdateProduct(db)(w, r)
		case http.MethodDelete:
			DeleteProduct(db)(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat PUT yoki DELETE metodi qo'llab-quvvatlanadi",
			})
		}
	}
}

// writeJSON is defined in auth.go - reusing it here
