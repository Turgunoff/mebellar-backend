package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	"mebellar-backend/models"
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

// writeJSON is defined in auth.go - reusing it here
