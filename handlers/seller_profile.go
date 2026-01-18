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

		// 2. Get seller_id from seller_profiles
		var sellerID string
		err = db.QueryRow(`SELECT id FROM seller_profiles WHERE user_id = $1`, userID).Scan(&sellerID)
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Sotuvchi profili topilmadi",
			})
			return
		}
		if err != nil {
			log.Printf("GetSellerProfile seller query error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Sotuvchi ma'lumotlarini olishda xatolik",
			})
			return
		}

		// 3. Do'kon ma'lumotlarini shops jadvalidan olish
		var shopData ShopStatsData
		var shopName models.StringMap
		err = db.QueryRow(`
			SELECT id, name, COALESCE(logo_url, ''), rating, is_verified
			FROM shops 
			WHERE id = $1 AND seller_id = $2
		`, shopID, sellerID).Scan(&shopData.ID, &shopName, &shopData.LogoURL, &shopData.Rating, &shopData.IsVerified)

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
		// Extract shop name from JSONB (prefer "uz" key)
		if name, ok := shopName["uz"]; ok {
			shopData.Name = name
		} else if name, ok := shopName["ru"]; ok {
			shopData.Name = name
		} else if name, ok := shopName["en"]; ok {
			shopData.Name = name
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

		// Get seller_id from seller_profiles first
		var sellerID string
		err := db.QueryRow(`SELECT id FROM seller_profiles WHERE user_id = $1`, userID).Scan(&sellerID)
		if err == sql.ErrNoRows {
			// No seller profile yet, return empty list
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success": true,
				"shops":   []interface{}{},
				"count":   0,
			})
			return
		}
		if err != nil {
			log.Printf("GetMyShops: Error getting seller_id: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Do'konlarni olishda xatolik",
			})
			return
		}

		// Query shops from shops table
		query := `
			SELECT 
				id, seller_id, name, description, address, slug,
				COALESCE(logo_url, ''), COALESCE(banner_url, ''),
				COALESCE(phone, ''), latitude, longitude, region_id,
				COALESCE(working_hours::text, '{}')::jsonb,
				is_active, is_verified, is_main, rating,
				created_at, updated_at
			FROM shops
			WHERE seller_id = $1
			ORDER BY created_at DESC
		`

		rows, err := db.Query(query, sellerID)
		if err != nil {
			log.Printf("GetMyShops query error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Do'konlarni olishda xatolik",
			})
			return
		}
		defer rows.Close()

		var shops []models.Shop
		for rows.Next() {
			var shop models.Shop
			var nameJSONB, descJSONB, addrJSONB models.StringMap
			err := rows.Scan(
				&shop.ID, &shop.SellerID, &nameJSONB, &descJSONB, &addrJSONB, &shop.Slug,
				&shop.LogoURL, &shop.BannerURL,
				&shop.Phone, &shop.Latitude, &shop.Longitude, &shop.RegionID,
				&shop.WorkingHours,
				&shop.IsActive, &shop.IsVerified, &shop.IsMain, &shop.Rating,
				&shop.CreatedAt, &shop.UpdatedAt,
			)
			if err == nil {
				shop.Name = nameJSONB
				shop.Description = descJSONB
				shop.Address = addrJSONB
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

		// Parse request body manually to handle address as both string and JSONB
		var rawReq map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&rawReq); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		// Parse request - accept both old format (shop_name, support_phone) and new format (name, phone)
		var nameStr string
		if nameRaw, ok := rawReq["name"].(string); ok {
			nameStr = nameRaw
		} else if shopNameRaw, ok := rawReq["shop_name"].(string); ok {
			// Backward compatibility
			nameStr = shopNameRaw
		} else {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Do'kon nomi kiritilishi shart",
			})
			return
		}

		if strings.TrimSpace(nameStr) == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Do'kon nomi kiritilishi shart",
			})
			return
		}

		// Get phone - accept both formats
		var phoneStr string
		if phoneRaw, ok := rawReq["phone"].(string); ok {
			phoneStr = phoneRaw
		} else if supportPhoneRaw, ok := rawReq["support_phone"].(string); ok {
			phoneStr = supportPhoneRaw
		}

		// Get description
		var descStr string
		if descRaw, ok := rawReq["description"].(string); ok {
			descStr = descRaw
		}

		// Prepare name as JSONB (convert string to {"uz": "value"})
		nameMap := models.StringMap{"uz": nameStr}
		
		// Prepare description as JSONB
		descMap := models.StringMap{}
		if descStr != "" {
			descMap["uz"] = descStr
		}

		// Handle address - can be string or JSONB object (for backward compatibility)
		addrMap := models.StringMap{}
		if addrRaw, ok := rawReq["address"]; ok && addrRaw != nil {
			switch v := addrRaw.(type) {
			case string:
				// Plain string from Flutter - convert to JSONB with uz key
				if v != "" {
					addrMap["uz"] = v
				}
			case map[string]interface{}:
				// Already JSONB object
				for k, val := range v {
					if str, ok := val.(string); ok {
						addrMap[k] = str
					}
				}
			}
		}

		// Handle working_hours
		var workingHours models.WorkingHours
		if workingHoursRaw, ok := rawReq["working_hours"]; ok {
			if workingHoursMap, ok := workingHoursRaw.(map[string]interface{}); ok {
				dayNames := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
				for _, dayName := range dayNames {
					if dayData, ok := workingHoursMap[dayName].(map[string]interface{}); ok {
						schedule := &models.DaySchedule{}
						if open, ok := dayData["open"].(string); ok {
							schedule.Open = open
						}
						if close, ok := dayData["close"].(string); ok {
							schedule.Close = close
						}
						if closed, ok := dayData["closed"].(bool); ok {
							schedule.Closed = closed
						}
						switch dayName {
						case "monday":
							workingHours.Monday = schedule
						case "tuesday":
							workingHours.Tuesday = schedule
						case "wednesday":
							workingHours.Wednesday = schedule
						case "thursday":
							workingHours.Thursday = schedule
						case "friday":
							workingHours.Friday = schedule
						case "saturday":
							workingHours.Saturday = schedule
						case "sunday":
							workingHours.Sunday = schedule
						}
					}
				}
			}
		}

		// Get seller_id from seller_profiles table, or create if doesn't exist
		var sellerID string
		err := db.QueryRow(`SELECT id FROM seller_profiles WHERE user_id = $1`, userID).Scan(&sellerID)
		if err == sql.ErrNoRows {
			// Auto-create seller profile for new users
			log.Printf("CreateShop: No seller profile found for user %s, creating one...", userID)
			
			// Get user's full_name to use as legal_name
			var userFullName string
			err := db.QueryRow(`SELECT COALESCE(full_name, '') FROM users WHERE id = $1`, userID).Scan(&userFullName)
			if err != nil {
				log.Printf("CreateShop: Error getting user full_name: %v", err)
				writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
					Success: false,
					Message: "Foydalanuvchi ma'lumotlarini olishda xatolik",
				})
				return
			}

			// Create seller profile
			err = db.QueryRow(
				`INSERT INTO seller_profiles (user_id, legal_name, is_verified, created_at, updated_at)
				 VALUES ($1, $2, $3, NOW(), NOW())
				 RETURNING id`,
				userID, userFullName, false,
			).Scan(&sellerID)
			if err != nil {
				log.Printf("CreateShop: Error creating seller profile: %v", err)
				writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
					Success: false,
					Message: "Sotuvchi profilini yaratishda xatolik",
				})
				return
			}
			log.Printf("✅ Created seller profile %s for user %s", sellerID, userID)
		} else if err != nil {
			log.Printf("CreateShop: Error getting seller_id: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Do'kon yaratishda xatolik",
			})
			return
		}

		// Generate slug from name
		slug := models.GenerateSlugFromName(nameMap)
		if slug == "" {
			slug = models.GenerateSlug(nameStr) // Fallback to old method
		}

		// Check if slug is unique, if not make it unique
		var existingSlug string
		slugCheckErr := db.QueryRow(`SELECT slug FROM shops WHERE slug = $1`, slug).Scan(&existingSlug)
		if slugCheckErr == nil {
			// Slug exists, make it unique
			var count int
			db.QueryRow(`SELECT COUNT(*) FROM shops WHERE slug LIKE $1`, slug+"%").Scan(&count)
			slug = slug + "-" + string(rune('0'+count+1))
		}

		// Check if this is the first shop for this seller (to set is_main)
		var shopCount int
		db.QueryRow(`SELECT COUNT(*) FROM shops WHERE seller_id = $1`, sellerID).Scan(&shopCount)
		isMain := shopCount == 0

		// Prepare JSONB values
		nameValue, _ := json.Marshal(nameMap)
		descValue, _ := json.Marshal(descMap)
		if len(descMap) == 0 {
			descValue = []byte("{}")
		}
		addrValue, _ := json.Marshal(addrMap)
		if len(addrMap) == 0 {
			addrValue = []byte("{}")
		}
		workingHoursValue, _ := json.Marshal(workingHours)
		if workingHoursValue == nil || len(workingHoursValue) == 0 {
			workingHoursValue = []byte("{}")
		}

		// Insert into shops table
		var shop models.Shop
		var nameJSONB, descJSONB, addrJSONB models.StringMap
		query := `
			INSERT INTO shops (
				seller_id, name, description, address, slug,
				phone, working_hours, is_active, is_main
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING 
				id, seller_id, name, description, address, slug,
				COALESCE(logo_url, ''), COALESCE(banner_url, ''),
				COALESCE(phone, ''), latitude, longitude, region_id,
				COALESCE(working_hours::text, '{}')::jsonb,
				is_active, is_verified, is_main, rating,
				created_at, updated_at
		`

		err = db.QueryRow(
			query,
			sellerID, nameValue, descValue, addrValue, slug,
			phoneStr, workingHoursValue, true, isMain,
		).Scan(
			&shop.ID, &shop.SellerID, &nameJSONB, &descJSONB, &addrJSONB, &shop.Slug,
			&shop.LogoURL, &shop.BannerURL,
			&shop.Phone, &shop.Latitude, &shop.Longitude, &shop.RegionID,
			&shop.WorkingHours,
			&shop.IsActive, &shop.IsVerified, &shop.IsMain, &shop.Rating,
			&shop.CreatedAt, &shop.UpdatedAt,
		)
		if err == nil {
			shop.Name = nameJSONB
			shop.Description = descJSONB
			shop.Address = addrJSONB
		}

		if err != nil {
			log.Printf("CreateShop error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Do'kon yaratishda xatolik",
			})
			return
		}

		log.Printf("🏪 New shop created: %s (ID: %s) by seller %s", shop.GetName("uz"), shop.ID, sellerID)

		// Return shop in response (convert to format Flutter expects)
		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"success": true,
			"message": "Do'kon muvaffaqiyatli yaratildi",
			"shop":    shop,
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

		// Parse request body manually to handle address as both string and JSONB
		var rawReq map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&rawReq); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		var req models.UpdateSellerProfileRequest
		if shopName, ok := rawReq["shop_name"].(string); ok {
			req.ShopName = &shopName
		}
		if desc, ok := rawReq["description"].(string); ok {
			req.Description = &desc
		}
		if logoURL, ok := rawReq["logo_url"].(string); ok {
			req.LogoURL = &logoURL
		}
		if bannerURL, ok := rawReq["banner_url"].(string); ok {
			req.BannerURL = &bannerURL
		}
		if phone, ok := rawReq["support_phone"].(string); ok {
			req.SupportPhone = &phone
		}
		if lat, ok := rawReq["latitude"].(float64); ok {
			req.Latitude = &lat
		}
		if lon, ok := rawReq["longitude"].(float64); ok {
			req.Longitude = &lon
		}

		// Handle address - can be string or JSONB object
		if addrRaw, ok := rawReq["address"]; ok && addrRaw != nil {
			switch v := addrRaw.(type) {
			case string:
				// Plain string - convert to JSONB with uz key
				if v != "" {
					addrMap := models.StringMap{"uz": v}
					req.Address = &addrMap
				}
			case map[string]interface{}:
				// Already JSONB object
				addrMap := make(models.StringMap)
				for k, val := range v {
					if str, ok := val.(string); ok {
						addrMap[k] = str
					}
				}
				if len(addrMap) > 0 {
					req.Address = &addrMap
				}
			}
		}

		// Handle social_links
		if socialLinksRaw, ok := rawReq["social_links"]; ok {
			if socialLinksMap, ok := socialLinksRaw.(map[string]interface{}); ok {
				var socialLinks models.SocialLinks
				if inst, ok := socialLinksMap["instagram"].(string); ok {
					socialLinks.Instagram = inst
				}
				if tg, ok := socialLinksMap["telegram"].(string); ok {
					socialLinks.Telegram = tg
				}
				if fb, ok := socialLinksMap["facebook"].(string); ok {
					socialLinks.Facebook = fb
				}
				if web, ok := socialLinksMap["website"].(string); ok {
					socialLinks.Website = web
				}
				if yt, ok := socialLinksMap["youtube"].(string); ok {
					socialLinks.YouTube = yt
				}
				req.SocialLinks = &socialLinks
			}
		}

		// Handle working_hours
		if workingHoursRaw, ok := rawReq["working_hours"]; ok {
			if workingHoursMap, ok := workingHoursRaw.(map[string]interface{}); ok {
				var workingHours models.WorkingHours
				dayNames := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
				for _, dayName := range dayNames {
					if dayData, ok := workingHoursMap[dayName].(map[string]interface{}); ok {
						schedule := &models.DaySchedule{}
						if open, ok := dayData["open"].(string); ok {
							schedule.Open = open
						}
						if close, ok := dayData["close"].(string); ok {
							schedule.Close = close
						}
						if closed, ok := dayData["closed"].(bool); ok {
							schedule.Closed = closed
						}
						switch dayName {
						case "monday":
							workingHours.Monday = schedule
						case "tuesday":
							workingHours.Tuesday = schedule
						case "wednesday":
							workingHours.Wednesday = schedule
						case "thursday":
							workingHours.Thursday = schedule
						case "friday":
							workingHours.Friday = schedule
						case "saturday":
							workingHours.Saturday = schedule
						case "sunday":
							workingHours.Sunday = schedule
						}
					}
				}
				req.WorkingHours = &workingHours
			}
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
