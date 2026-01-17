package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"mebellar-backend/models"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// ============================================
// SELLER PROFILE ENDPOINT (Aggregated)
// ============================================

// SellerProfileData - Seller profile ma'lumotlari (user + shop stats)
type SellerProfileData struct {
	User *UserProfileData `json:"user"`
	Shop *ShopStatsData   `json:"shop"`
}

// UserProfileData - Foydalanuvchi ma'lumotlari
type UserProfileData struct {
	ID        string `json:"id"`
	FullName  string `json:"full_name"`
	Phone     string `json:"phone"`
	Email     string `json:"email,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Role      string `json:"role"`
}

// ShopStatsData - Do'kon statistikasi
type ShopStatsData struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	LogoURL       string  `json:"logo_url,omitempty"`
	Rating        float64 `json:"rating"`
	ProductsCount int     `json:"products_count"`
	OrdersCount   int     `json:"orders_count"`
	IsVerified    bool    `json:"is_verified"`
}

// GetSellerProfile godoc
// @Summary      Sotuvchi profili (aggregated)
// @Description  Foydalanuvchi ma'lumotlari va tanlangan do'kon statistikasini qaytaradi
// @Tags         seller
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Shop-ID header string true "Joriy do'kon ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /seller/profile [get]
func GetSellerProfile(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
				Success: false,
				Message: "Avtorizatsiya talab qilinadi",
			})
			return
		}

		shopID := r.Header.Get("X-Shop-ID")
		if shopID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "X-Shop-ID header kiritilishi shart",
			})
			return
		}

		// 1. Foydalanuvchi ma'lumotlarini olish
		var userData UserProfileData
		err := db.QueryRow(`
			SELECT id, full_name, phone, COALESCE(email, ''), COALESCE(avatar_url, ''), COALESCE(role, 'seller')
			FROM users WHERE id = $1
		`, userID).Scan(&userData.ID, &userData.FullName, &userData.Phone, &userData.Email, &userData.AvatarURL, &userData.Role)

		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Foydalanuvchi topilmadi",
			})
			return
		}
		if err != nil {
			log.Printf("GetSellerProfile user query error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Foydalanuvchi ma'lumotlarini olishda xatolik",
			})
			return
		}

		// 2. Do'kon ma'lumotlarini olish
		var shopData ShopStatsData
		err = db.QueryRow(`
			SELECT id, shop_name, COALESCE(logo_url, ''), rating, is_verified
			FROM seller_profiles 
			WHERE id = $1 AND user_id = $2
		`, shopID, userID).Scan(&shopData.ID, &shopData.Name, &shopData.LogoURL, &shopData.Rating, &shopData.IsVerified)

		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Do'kon topilmadi yoki sizga tegishli emas",
			})
			return
		}
		if err != nil {
			log.Printf("GetSellerProfile shop query error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Do'kon ma'lumotlarini olishda xatolik",
			})
			return
		}

		// 3. Aktiv mahsulotlar sonini olish
		err = db.QueryRow(`
			SELECT COUNT(*) FROM products 
			WHERE shop_id = $1 AND is_active = true
		`, shopID).Scan(&shopData.ProductsCount)
		if err != nil {
			log.Printf("GetSellerProfile products count error: %v", err)
			shopData.ProductsCount = 0
		}

		// 4. Bajarilgan buyurtmalar sonini olish (completed)
		err = db.QueryRow(`
			SELECT COUNT(*) FROM orders 
			WHERE shop_id = $1 AND status = 'completed'
		`, shopID).Scan(&shopData.OrdersCount)
		if err != nil {
			log.Printf("GetSellerProfile orders count error: %v", err)
			shopData.OrdersCount = 0
		}

		log.Printf("✅ Seller profile fetched: user=%s, shop=%s, products=%d, orders=%d",
			userData.FullName, shopData.Name, shopData.ProductsCount, shopData.OrdersCount)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"user":    userData,
			"shop":    shopData,
		})
	}
}

// UpdateSellerProfileRequest - Profil yangilash so'rovi
type UpdateSellerProfileRequest struct {
	FullName    *string `json:"full_name,omitempty"`
	OldPassword *string `json:"old_password,omitempty"`
	NewPassword *string `json:"new_password,omitempty"`
}

// UpdateSellerProfile godoc
// @Summary      Sotuvchi profilini yangilash
// @Description  Foydalanuvchi ismini va/yoki parolini yangilaydi
// @Tags         seller
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body UpdateSellerProfileRequest true "Yangilash ma'lumotlari"
// @Success      200  {object}  models.AuthResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      401  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /seller/profile [put]
func UpdateSellerProfile(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat PUT metodi qo'llab-quvvatlanadi",
			})
			return
		}

		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
				Success: false,
				Message: "Avtorizatsiya talab qilinadi",
			})
			return
		}

		var req UpdateSellerProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		// Check if at least one field to update
		if req.FullName == nil && req.NewPassword == nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Yangilanadigan ma'lumot yo'q",
			})
			return
		}

		// If changing password, verify old password first
		if req.NewPassword != nil {
			if req.OldPassword == nil || *req.OldPassword == "" {
				writeJSON(w, http.StatusBadRequest, models.AuthResponse{
					Success: false,
					Message: "Eski parol kiritilishi shart",
				})
				return
			}

			if len(*req.NewPassword) < 6 {
				writeJSON(w, http.StatusBadRequest, models.AuthResponse{
					Success: false,
					Message: "Yangi parol kamida 6 ta belgidan iborat bo'lishi kerak",
				})
				return
			}

			// Verify old password
			var currentPasswordHash string
			err := db.QueryRow(`SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&currentPasswordHash)
			if err != nil {
				log.Printf("UpdateSellerProfile password fetch error: %v", err)
				writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
					Success: false,
					Message: "Server xatosi",
				})
				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(currentPasswordHash), []byte(*req.OldPassword)); err != nil {
				writeJSON(w, http.StatusBadRequest, models.AuthResponse{
					Success: false,
					Message: "Eski parol noto'g'ri",
				})
				return
			}
		}

		// Build dynamic update query
		updates := []string{}
		args := []interface{}{}
		argIndex := 1

		if req.FullName != nil && strings.TrimSpace(*req.FullName) != "" {
			updates = append(updates, "full_name = $"+string(rune('0'+argIndex)))
			args = append(args, strings.TrimSpace(*req.FullName))
			argIndex++
		}

		if req.NewPassword != nil {
			// Hash new password
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.NewPassword), bcrypt.DefaultCost)
			if err != nil {
				log.Printf("UpdateSellerProfile password hash error: %v", err)
				writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
					Success: false,
					Message: "Server xatosi",
				})
				return
			}
			updates = append(updates, "password_hash = $"+string(rune('0'+argIndex)))
			args = append(args, string(hashedPassword))
			argIndex++
		}

		if len(updates) == 0 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Yangilanadigan ma'lumot yo'q",
			})
			return
		}

		// Add updated_at and user_id
		updates = append(updates, "updated_at = NOW()")
		args = append(args, userID)

		query := "UPDATE users SET " + strings.Join(updates, ", ") + " WHERE id = $" + string(rune('0'+argIndex))

		_, err := db.Exec(query, args...)
		if err != nil {
			log.Printf("UpdateSellerProfile update error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Profilni yangilashda xatolik",
			})
			return
		}

		log.Printf("✅ Seller profile updated: user_id=%s", userID)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: "Profil muvaffaqiyatli yangilandi",
		})
	}
}

// DeleteSellerAccount godoc
// @Summary      Hisobni o'chirish (soft delete)
// @Description  Foydalanuvchi hisobini o'chiradi (is_active = false)
// @Tags         seller
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  models.AuthResponse
// @Failure      401  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /seller/account [delete]
func DeleteSellerAccount(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat DELETE metodi qo'llab-quvvatlanadi",
			})
			return
		}

		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
				Success: false,
				Message: "Avtorizatsiya talab qilinadi",
			})
			return
		}

		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("DeleteSellerAccount tx begin error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Server xatosi",
			})
			return
		}
		defer tx.Rollback()

		// 1. Soft delete user (set is_active = false)
		_, err = tx.Exec(`
			UPDATE users 
			SET is_active = false, updated_at = NOW() 
			WHERE id = $1
		`, userID)
		if err != nil {
			log.Printf("DeleteSellerAccount user update error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Hisobni o'chirishda xatolik",
			})
			return
		}

		// 2. Deactivate all shops owned by this user
		_, err = tx.Exec(`
			UPDATE seller_profiles 
			SET is_verified = false, updated_at = NOW() 
			WHERE user_id = $1
		`, userID)
		if err != nil {
			log.Printf("DeleteSellerAccount shops update error: %v", err)
			// Continue anyway - not critical
		}

		// 3. Deactivate all products from user's shops
		_, err = tx.Exec(`
			UPDATE products 
			SET is_active = false, updated_at = NOW() 
			WHERE shop_id IN (SELECT id FROM seller_profiles WHERE user_id = $1)
		`, userID)
		if err != nil {
			log.Printf("DeleteSellerAccount products update error: %v", err)
			// Continue anyway - not critical
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			log.Printf("DeleteSellerAccount tx commit error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Server xatosi",
			})
			return
		}

		log.Printf("🗑️ Seller account soft deleted: user_id=%s", userID)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: "Hisobingiz muvaffaqiyatli o'chirildi",
		})
	}
}

// SellerProfileHandler - /api/seller/profile uchun method router
func SellerProfileHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			GetSellerProfile(db)(w, r)
		case http.MethodPut:
			UpdateSellerProfile(db)(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Bu metod qo'llab-quvvatlanmaydi",
			})
		}
	}
}

// ============================================
// SHOP HANDLERS (Multi-Shop Architecture)
// ============================================

// GetMyShops godoc
// @Summary      Mening do'konlarim
// @Description  Joriy foydalanuvchining barcha do'konlarini qaytaradi
// @Tags         seller
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /seller/shops [get]
func GetMyShops(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
				Success: false,
				Message: "Avtorizatsiya talab qilinadi",
			})
			return
		}

		query := `
			SELECT 
				id, user_id, shop_name, COALESCE(slug, ''), COALESCE(description, ''),
				COALESCE(logo_url, ''), COALESCE(banner_url, ''),
				COALESCE(support_phone, ''), COALESCE(address::text, '{}')::jsonb,
				latitude, longitude,
				COALESCE(social_links::text, '{}')::jsonb,
				COALESCE(working_hours::text, '{}')::jsonb,
				is_verified, rating, created_at, updated_at
			FROM seller_profiles
			WHERE user_id = $1
			ORDER BY created_at DESC
		`

		rows, err := db.Query(query, userID)
		if err != nil {
			log.Printf("GetMyShops query error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Do'konlarni olishda xatolik",
			})
			return
		}
		defer rows.Close()

		var shops []models.SellerProfile
		for rows.Next() {
			var shop models.SellerProfile
			var addressJSONB models.StringMap
			err := rows.Scan(
				&shop.ID, &shop.UserID, &shop.ShopName, &shop.Slug, &shop.Description,
				&shop.LogoURL, &shop.BannerURL,
				&shop.SupportPhone, &addressJSONB,
				&shop.Latitude, &shop.Longitude,
				&shop.SocialLinks, &shop.WorkingHours,
				&shop.IsVerified, &shop.Rating, &shop.CreatedAt, &shop.UpdatedAt,
			)
			if err == nil {
				shop.Address = addressJSONB
			}
			if err != nil {
				log.Printf("GetMyShops scan error: %v", err)
				continue
			}
			shops = append(shops, shop)
		}

		log.Printf("✅ User %s has %d shops", userID, len(shops))

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"shops":   shops,
			"count":   len(shops),
		})
	}
}

// CreateShop godoc
// @Summary      Yangi do'kon yaratish
// @Description  Sotuvchi uchun yangi do'kon yaratadi (bir foydalanuvchi ko'p do'kon ochishi mumkin)
// @Tags         seller
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body models.CreateSellerProfileRequest true "Do'kon ma'lumotlari"
// @Success      201  {object}  models.SellerProfileResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      401  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /seller/shops [post]
func CreateShop(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
			})
			return
		}

		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
				Success: false,
				Message: "Avtorizatsiya talab qilinadi",
			})
			return
		}

		var req models.CreateSellerProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		// Validatsiya
		if strings.TrimSpace(req.ShopName) == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Do'kon nomi kiritilishi shart",
			})
			return
		}

		// Slug yaratish
		slug := models.GenerateSlug(req.ShopName)

		// Slug unikal ekanligini tekshirish
		var existingSlug string
		slugCheckErr := db.QueryRow(`SELECT slug FROM seller_profiles WHERE slug = $1`, slug).Scan(&existingSlug)
		if slugCheckErr == nil {
			// Slug mavjud, unikal qilish
			var count int
			db.QueryRow(`SELECT COUNT(*) FROM seller_profiles WHERE slug LIKE $1`, slug+"%").Scan(&count)
			slug = slug + "-" + string(rune('0'+count+1))
		}

		// Social links va working hours ni JSON ga aylantirish
		socialLinksJSON, _ := json.Marshal(req.SocialLinks)
		workingHoursJSON, _ := json.Marshal(req.WorkingHours)
		
		// Address ni JSONB ga aylantirish
		var addressValue []byte
		if req.Address != nil {
			addressValue, _ = json.Marshal(*req.Address)
		} else {
			addressValue = []byte("{}")
		}

		// Do'konni yaratish
		var shop models.SellerProfile
		var addressJSONB models.StringMap
		query := `
			INSERT INTO seller_profiles (
				user_id, shop_name, slug, description,
				support_phone, address, social_links, working_hours
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id, user_id, shop_name, slug, COALESCE(description, ''),
				COALESCE(logo_url, ''), COALESCE(banner_url, ''),
				COALESCE(support_phone, ''), COALESCE(address::text, '{}')::jsonb,
				latitude, longitude,
				COALESCE(social_links::text, '{}')::jsonb,
				COALESCE(working_hours::text, '{}')::jsonb,
				is_verified, rating, created_at, updated_at
		`

		err := db.QueryRow(
			query,
			userID, req.ShopName, slug, req.Description,
			req.SupportPhone, addressValue, socialLinksJSON, workingHoursJSON,
		).Scan(
			&shop.ID, &shop.UserID, &shop.ShopName, &shop.Slug, &shop.Description,
			&shop.LogoURL, &shop.BannerURL,
			&shop.SupportPhone, &addressJSONB,
			&shop.Latitude, &shop.Longitude,
			&shop.SocialLinks, &shop.WorkingHours,
			&shop.IsVerified, &shop.Rating, &shop.CreatedAt, &shop.UpdatedAt,
		)
		if err == nil {
			shop.Address = addressJSONB
		}

		if err != nil {
			log.Printf("CreateShop error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Do'kon yaratishda xatolik",
			})
			return
		}

		log.Printf("🏪 New shop created: %s (ID: %s) by user %s", shop.ShopName, shop.ID, userID)

		writeJSON(w, http.StatusCreated, models.SellerProfileResponse{
			Success: true,
			Message: "Do'kon muvaffaqiyatli yaratildi",
			Profile: &shop,
		})
	}
}

// GetShopByID godoc
// @Summary      Do'kon ma'lumotlari
// @Description  Do'kon ID bo'yicha ma'lumotlarni qaytaradi
// @Tags         seller
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Do'kon ID"
// @Success      200  {object}  models.SellerProfileResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Router       /seller/shops/{id} [get]
func GetShopByID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
				Success: false,
				Message: "Avtorizatsiya talab qilinadi",
			})
			return
		}

		// URL dan shop ID olish: /api/seller/shops/{id}
		path := strings.TrimPrefix(r.URL.Path, "/api/seller/shops/")
		shopID := strings.TrimSuffix(path, "/")

		if shopID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Do'kon ID kiritilishi shart",
			})
			return
		}

		var shop models.SellerProfile
		var addressJSONB models.StringMap
		query := `
			SELECT 
				id, user_id, shop_name, COALESCE(slug, ''), COALESCE(description, ''),
				COALESCE(logo_url, ''), COALESCE(banner_url, ''),
				COALESCE(legal_name, ''), COALESCE(tax_id, ''),
				COALESCE(bank_account, ''), COALESCE(bank_name, ''),
				COALESCE(support_phone, ''), COALESCE(address::text, '{}')::jsonb,
				latitude, longitude,
				COALESCE(social_links::text, '{}')::jsonb,
				COALESCE(working_hours::text, '{}')::jsonb,
				is_verified, rating, created_at, updated_at
			FROM seller_profiles
			WHERE id = $1
		`

		err := db.QueryRow(query, shopID).Scan(
			&shop.ID, &shop.UserID, &shop.ShopName, &shop.Slug, &shop.Description,
			&shop.LogoURL, &shop.BannerURL,
			&shop.LegalName, &shop.TaxID,
			&shop.BankAccount, &shop.BankName,
			&shop.SupportPhone, &addressJSONB,
			&shop.Latitude, &shop.Longitude,
			&shop.SocialLinks, &shop.WorkingHours,
			&shop.IsVerified, &shop.Rating, &shop.CreatedAt, &shop.UpdatedAt,
		)
		if err == nil {
			shop.Address = addressJSONB
		}

		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Do'kon topilmadi",
			})
			return
		}

		if err != nil {
			log.Printf("GetShopByID error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Do'konni olishda xatolik",
			})
			return
		}

		// Faqat egasi ko'ra oladi (yoki admin)
		if shop.UserID != userID {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Bu do'konga kirish huquqi yo'q",
			})
			return
		}

		writeJSON(w, http.StatusOK, models.SellerProfileResponse{
			Success: true,
			Profile: &shop,
		})
	}
}

// UpdateShop godoc
// @Summary      Do'konni yangilash
// @Description  Do'kon ma'lumotlarini yangilaydi
// @Tags         seller
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Do'kon ID"
// @Param        request body models.UpdateSellerProfileRequest true "Yangilash ma'lumotlari"
// @Success      200  {object}  models.SellerProfileResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Router       /seller/shops/{id} [put]
func UpdateShop(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat PUT metodi qo'llab-quvvatlanadi",
			})
			return
		}

		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
				Success: false,
				Message: "Avtorizatsiya talab qilinadi",
			})
			return
		}

		// URL dan shop ID olish
		path := strings.TrimPrefix(r.URL.Path, "/api/seller/shops/")
		shopID := strings.TrimSuffix(path, "/")

		if shopID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Do'kon ID kiritilishi shart",
			})
			return
		}

		// Do'kon egasini tekshirish
		var ownerID string
		err := db.QueryRow(`SELECT user_id FROM seller_profiles WHERE id = $1`, shopID).Scan(&ownerID)
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Do'kon topilmadi",
			})
			return
		}
		if ownerID != userID {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Bu do'konni yangilash huquqi yo'q",
			})
			return
		}

		var req models.UpdateSellerProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		// Dinamik UPDATE query
		updates := []string{}
		args := []interface{}{}
		argIndex := 1

		if req.ShopName != nil {
			updates = append(updates, "shop_name = $"+string(rune('0'+argIndex)))
			args = append(args, *req.ShopName)
			argIndex++
		}
		if req.Description != nil {
			updates = append(updates, "description = $"+string(rune('0'+argIndex)))
			args = append(args, *req.Description)
			argIndex++
		}
		if req.LogoURL != nil {
			updates = append(updates, "logo_url = $"+string(rune('0'+argIndex)))
			args = append(args, *req.LogoURL)
			argIndex++
		}
		if req.BannerURL != nil {
			updates = append(updates, "banner_url = $"+string(rune('0'+argIndex)))
			args = append(args, *req.BannerURL)
			argIndex++
		}
		if req.SupportPhone != nil {
			updates = append(updates, "support_phone = $"+string(rune('0'+argIndex)))
			args = append(args, *req.SupportPhone)
			argIndex++
		}
		if req.Address != nil {
			addressValue, _ := req.Address.Value()
			updates = append(updates, "address = $"+string(rune('0'+argIndex)))
			args = append(args, addressValue)
			argIndex++
		}
		if req.Latitude != nil {
			updates = append(updates, "latitude = $"+string(rune('0'+argIndex)))
			args = append(args, *req.Latitude)
			argIndex++
		}
		if req.Longitude != nil {
			updates = append(updates, "longitude = $"+string(rune('0'+argIndex)))
			args = append(args, *req.Longitude)
			argIndex++
		}
		if req.SocialLinks != nil {
			socialLinksJSON, _ := json.Marshal(req.SocialLinks)
			updates = append(updates, "social_links = $"+string(rune('0'+argIndex)))
			args = append(args, socialLinksJSON)
			argIndex++
		}
		if req.WorkingHours != nil {
			workingHoursJSON, _ := json.Marshal(req.WorkingHours)
			updates = append(updates, "working_hours = $"+string(rune('0'+argIndex)))
			args = append(args, workingHoursJSON)
			argIndex++
		}

		if len(updates) == 0 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Yangilanadigan ma'lumot yo'q",
			})
			return
		}

		// updated_at ni qo'shish
		updates = append(updates, "updated_at = NOW()")
		args = append(args, shopID)

		query := "UPDATE seller_profiles SET " + strings.Join(updates, ", ") + " WHERE id = $" + string(rune('0'+argIndex))

		_, err = db.Exec(query, args...)
		if err != nil {
			log.Printf("UpdateShop error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Do'konni yangilashda xatolik",
			})
			return
		}

		log.Printf("🏪 Shop updated: %s by user %s", shopID, userID)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: "Do'kon muvaffaqiyatli yangilandi",
		})
	}
}

// DeleteShop godoc
// @Summary      Do'konni o'chirish
// @Description  Do'konni o'chiradi
// @Tags         seller
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Do'kon ID"
// @Success      200  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Router       /seller/shops/{id} [delete]
func DeleteShop(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat DELETE metodi qo'llab-quvvatlanadi",
			})
			return
		}

		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
				Success: false,
				Message: "Avtorizatsiya talab qilinadi",
			})
			return
		}

		// URL dan shop ID olish
		path := strings.TrimPrefix(r.URL.Path, "/api/seller/shops/")
		shopID := strings.TrimSuffix(path, "/")

		if shopID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Do'kon ID kiritilishi shart",
			})
			return
		}

		// Do'kon egasini tekshirish
		var ownerID string
		err := db.QueryRow(`SELECT user_id FROM seller_profiles WHERE id = $1`, shopID).Scan(&ownerID)
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Do'kon topilmadi",
			})
			return
		}
		if ownerID != userID {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Bu do'konni o'chirish huquqi yo'q",
			})
			return
		}

		_, err = db.Exec(`DELETE FROM seller_profiles WHERE id = $1`, shopID)
		if err != nil {
			log.Printf("DeleteShop error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Do'konni o'chirishda xatolik",
			})
			return
		}

		log.Printf("🗑️ Shop deleted: %s by user %s", shopID, userID)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: "Do'kon muvaffaqiyatli o'chirildi",
		})
	}
}

// ============================================
// PUBLIC SHOP ENDPOINTS
// ============================================

// GetPublicShopBySlug godoc
// @Summary      Do'kon sahifasi (ommaviy)
// @Description  Slug bo'yicha do'kon ma'lumotlarini qaytaradi (maxfiy ma'lumotlarsiz)
// @Tags         shops
// @Accept       json
// @Produce      json
// @Param        slug path string true "Do'kon slug"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  models.AuthResponse
// @Router       /shops/{slug} [get]
func GetPublicShopBySlug(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// URL dan slug olish: /api/shops/{slug}
		path := strings.TrimPrefix(r.URL.Path, "/api/shops/")
		slug := strings.TrimSuffix(path, "/")

		if slug == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Do'kon slug kiritilishi shart",
			})
			return
		}

		var shop models.SellerProfile
		var addressJSONB models.StringMap
		query := `
			SELECT 
				id, user_id, shop_name, COALESCE(slug, ''), COALESCE(description, ''),
				COALESCE(logo_url, ''), COALESCE(banner_url, ''),
				COALESCE(support_phone, ''), COALESCE(address::text, '{}')::jsonb,
				latitude, longitude,
				COALESCE(social_links::text, '{}')::jsonb,
				COALESCE(working_hours::text, '{}')::jsonb,
				is_verified, rating
			FROM seller_profiles
			WHERE slug = $1
		`

		err := db.QueryRow(query, slug).Scan(
			&shop.ID, &shop.UserID, &shop.ShopName, &shop.Slug, &shop.Description,
			&shop.LogoURL, &shop.BannerURL,
			&shop.SupportPhone, &addressJSONB,
			&shop.Latitude, &shop.Longitude,
			&shop.SocialLinks, &shop.WorkingHours,
			&shop.IsVerified, &shop.Rating,
		)
		if err == nil {
			shop.Address = addressJSONB
		}

		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Do'kon topilmadi",
			})
			return
		}

		if err != nil {
			log.Printf("GetPublicShopBySlug error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Do'konni olishda xatolik",
			})
			return
		}

		// Maxfiy ma'lumotlarsiz qaytarish
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"shop":    shop.ToPublic(),
		})
	}
}

// ============================================
// SHOP CONTEXT MIDDLEWARE
// ============================================

// RequireShopID - X-Shop-ID header ni tekshiruvchi middleware
func RequireShopID(db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
				Success: false,
				Message: "Avtorizatsiya talab qilinadi",
			})
			return
		}

		shopID := r.Header.Get("X-Shop-ID")
		if shopID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "X-Shop-ID header kiritilishi shart",
			})
			return
		}

		// Do'kon mavjudligini va egasini tekshirish
		var ownerID string
		err := db.QueryRow(`SELECT user_id FROM seller_profiles WHERE id = $1`, shopID).Scan(&ownerID)
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Do'kon topilmadi",
			})
			return
		}
		if err != nil {
			log.Printf("RequireShopID error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Do'konni tekshirishda xatolik",
			})
			return
		}

		if ownerID != userID {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Bu do'konga kirish huquqi yo'q",
			})
			return
		}

		// Shop ID ni header ga qo'shish (keyingi handler uchun)
		r.Header.Set("X-Verified-Shop-ID", shopID)

		next(w, r)
	}
}

// ShopsHandler - /api/seller/shops uchun method router
func ShopsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			GetMyShops(db)(w, r)
		case http.MethodPost:
			CreateShop(db)(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Bu metod qo'llab-quvvatlanmaydi",
			})
		}
	}
}

// ShopByIDHandler - /api/seller/shops/{id} uchun method router
func ShopByIDHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			GetShopByID(db)(w, r)
		case http.MethodPut:
			UpdateShop(db)(w, r)
		case http.MethodDelete:
			DeleteShop(db)(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Bu metod qo'llab-quvvatlanmaydi",
			})
		}
	}
}
