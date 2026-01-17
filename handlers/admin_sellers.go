package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"mebellar-backend/models"
	"net/http"
	"strconv"
	"strings"
)

// ============================================
// ADMIN SELLER MANAGEMENT
// ============================================

// SellersResponse - Sotuvchilar ro'yxati javobi
type SellersResponse struct {
	Success bool                `json:"success"`
	Message string              `json:"message,omitempty"`
	Sellers []SellerListItem    `json:"sellers"`
	Total   int                 `json:"total"`
	Page    int                 `json:"page,omitempty"`
	Limit   int                 `json:"limit,omitempty"`
}

// SellerListItem - Sotuvchi ro'yxat elementi
type SellerListItem struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	FullName    string `json:"full_name"`
	Phone       string `json:"phone"`
	LegalName   string `json:"legal_name"`
	TaxID       string `json:"tax_id"`
	BankAccount string `json:"bank_account,omitempty"`
	BankName    string `json:"bank_name,omitempty"`
	IsVerified  bool   `json:"is_verified"`
	ShopsCount  int    `json:"shops_count"`
	CreatedAt   string `json:"created_at"`
}

// SellerDetailResponse - Bitta sotuvchi to'liq ma'lumotlari
type SellerDetailResponse struct {
	Success bool                `json:"success"`
	Message string              `json:"message,omitempty"`
	Seller   *SellerDetail      `json:"seller,omitempty"`
}

// SellerDetail - Sotuvchi to'liq ma'lumotlari
type SellerDetail struct {
	SellerProfile *models.SellerProfile `json:"seller_profile"`
	User          *models.User           `json:"user"`
	Shops         []models.Shop          `json:"shops"`
	ShopsCount    int                   `json:"shops_count"`
}

// GetSellers godoc
// @Summary      Barcha sotuvchilarni olish (Admin)
// @Description  Admin panel uchun barcha sotuvchilar ro'yxatini qaytaradi
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        page query int false "Sahifa raqami (default: 1)"
// @Param        limit query int false "Har sahifadagi sotuvchilar soni (default: 10, max: 100)"
// @Param        is_verified query bool false "Tasdiqlangan/Tasdiqlanmagan filter"
// @Success      200  {object}  SellersResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/sellers [get]
func GetSellers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Role tekshirish
		userRole := r.Header.Get("X-User-Role")
		if userRole != "admin" && userRole != "moderator" {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Bu sahifaga kirish huquqingiz yo'q",
			})
			return
		}

		// Query parametrlarini olish
		pageStr := r.URL.Query().Get("page")
		limitStr := r.URL.Query().Get("limit")
		isVerifiedStr := r.URL.Query().Get("is_verified")

		// Pagination sozlash
		page := 1
		limit := 10
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
		offset := (page - 1) * limit

		// SQL so'rov yaratish
		baseQuery := `
			SELECT 
				sp.id, sp.user_id, u.full_name, u.phone,
				COALESCE(sp.legal_name, ''), COALESCE(sp.tax_id, ''),
				COALESCE(sp.bank_account, ''), COALESCE(sp.bank_name, ''),
				sp.is_verified, sp.created_at,
				(SELECT COUNT(*) FROM shops WHERE seller_id = sp.id) as shops_count
			FROM seller_profiles sp
			INNER JOIN users u ON sp.user_id = u.id
			WHERE u.is_active = true
		`
		countQuery := `
			SELECT COUNT(*)
			FROM seller_profiles sp
			INNER JOIN users u ON sp.user_id = u.id
			WHERE u.is_active = true
		`
		args := []interface{}{}
		argIndex := 1

		// Verification filtri
		if isVerifiedStr == "true" {
			baseQuery += fmt.Sprintf(" AND sp.is_verified = $%d", argIndex)
			countQuery += fmt.Sprintf(" AND sp.is_verified = $%d", argIndex)
			args = append(args, true)
			argIndex++
		} else if isVerifiedStr == "false" {
			baseQuery += fmt.Sprintf(" AND sp.is_verified = $%d", argIndex)
			countQuery += fmt.Sprintf(" AND sp.is_verified = $%d", argIndex)
			args = append(args, false)
			argIndex++
		}

		// Jami sonni olish
		var total int
		err := db.QueryRow(countQuery, args...).Scan(&total)
		if err != nil {
			log.Printf("GetSellers: Count query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Sotuvchilarni olishda xatolik",
			})
			return
		}

		// Sotuvchilarni olish (pagination bilan)
		dataQuery := baseQuery + ` ORDER BY sp.created_at DESC LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
		args = append(args, limit, offset)

		rows, err := db.Query(dataQuery, args...)
		if err != nil {
			log.Printf("GetSellers: Sellers query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Sotuvchilarni olishda xatolik",
			})
			return
		}
		defer rows.Close()

		sellers := []SellerListItem{}
		for rows.Next() {
			var s SellerListItem
			err := rows.Scan(
				&s.ID, &s.UserID, &s.FullName, &s.Phone,
				&s.LegalName, &s.TaxID,
				&s.BankAccount, &s.BankName,
				&s.IsVerified, &s.CreatedAt, &s.ShopsCount,
			)
			if err != nil {
				log.Printf("GetSellers: Seller scan xatosi: %v", err)
				continue
			}
			sellers = append(sellers, s)
		}

		log.Printf("✅ %d ta sotuvchi topildi (sahifa %d)", len(sellers), page)

		writeJSON(w, http.StatusOK, SellersResponse{
			Success: true,
			Sellers: sellers,
			Total:   total,
			Page:    page,
			Limit:   limit,
		})
	}
}

// GetSellerDetail godoc
// @Summary      Sotuvchi to'liq ma'lumotlari (Admin)
// @Description  Sotuvchi ID bo'yicha to'liq ma'lumotlar va uning do'konlarini qaytaradi
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path string true "Sotuvchi ID"
// @Success      200  {object}  SellerDetailResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/sellers/{id} [get]
func GetSellerDetail(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Role tekshirish
		userRole := r.Header.Get("X-User-Role")
		if userRole != "admin" && userRole != "moderator" {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Bu sahifaga kirish huquqingiz yo'q",
			})
			return
		}

		// URL dan seller ID olish
		path := strings.TrimPrefix(r.URL.Path, "/api/admin/sellers/")
		sellerID := strings.TrimSuffix(path, "/")

		if sellerID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Sotuvchi ID kiritilishi shart",
			})
			return
		}

		// Seller profile ma'lumotlarini olish
		var sellerProfile models.SellerProfile
		var addressJSONB models.StringMap
		err := db.QueryRow(`
			SELECT 
				id, user_id, shop_name, COALESCE(slug, ''), COALESCE(description, ''),
				COALESCE(logo_url, ''), COALESCE(banner_url, ''),
				COALESCE(legal_name, ''), COALESCE(tax_id, ''),
				COALESCE(bank_account, ''), COALESCE(bank_name, ''),
				COALESCE(support_phone, ''), 
				COALESCE(address::text, '{}')::jsonb,
				latitude, longitude,
				COALESCE(social_links::text, '{}')::jsonb,
				COALESCE(working_hours::text, '{}')::jsonb,
				is_verified, rating, created_at, updated_at
			FROM seller_profiles
			WHERE id = $1
		`, sellerID).Scan(
			&sellerProfile.ID, &sellerProfile.UserID, &sellerProfile.ShopName, &sellerProfile.Slug, &sellerProfile.Description,
			&sellerProfile.LogoURL, &sellerProfile.BannerURL,
			&sellerProfile.LegalName, &sellerProfile.TaxID,
			&sellerProfile.BankAccount, &sellerProfile.BankName,
			&sellerProfile.SupportPhone, &addressJSONB,
			&sellerProfile.Latitude, &sellerProfile.Longitude,
			&sellerProfile.SocialLinks, &sellerProfile.WorkingHours,
			&sellerProfile.IsVerified, &sellerProfile.Rating, &sellerProfile.CreatedAt, &sellerProfile.UpdatedAt,
		)

		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Sotuvchi topilmadi",
			})
			return
		}
		if err != nil {
			log.Printf("GetSellerDetail: Seller query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Sotuvchi ma'lumotlarini olishda xatolik",
			})
			return
		}
		sellerProfile.Address = addressJSONB

		// User ma'lumotlarini olish
		var user models.User
		err = db.QueryRow(`
			SELECT id, full_name, phone, COALESCE(email, ''), 
				COALESCE(avatar_url, ''), COALESCE(role, 'seller'),
				created_at, updated_at
			FROM users
			WHERE id = $1
		`, sellerProfile.UserID).Scan(
			&user.ID, &user.FullName, &user.Phone, &user.Email,
			&user.AvatarURL, &user.Role,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			log.Printf("GetSellerDetail: User query xatosi: %v", err)
		}

		// Shops ma'lumotlarini olish
		shopsQuery := `
			SELECT 
				s.id, s.seller_id,
				COALESCE(s.name::text, '{}')::jsonb,
				COALESCE(s.description::text, '{}')::jsonb,
				COALESCE(s.address::text, '{}')::jsonb,
				COALESCE(s.slug, ''), COALESCE(s.logo_url, ''), COALESCE(s.banner_url, ''),
				COALESCE(s.phone, ''), s.latitude, s.longitude, s.region_id,
				COALESCE(s.working_hours::text, '{}')::jsonb,
				s.is_active, s.is_verified, s.is_main, s.rating,
				s.created_at, s.updated_at
			FROM shops s
			WHERE s.seller_id = $1
			ORDER BY s.is_main DESC, s.created_at DESC
		`

		shopsRows, err := db.Query(shopsQuery, sellerID)
		if err != nil {
			log.Printf("GetSellerDetail: Shops query xatosi: %v", err)
		}
		defer shopsRows.Close()

		shops := []models.Shop{}
		for shopsRows.Next() {
			var shop models.Shop
			var nameJSONB, descJSONB, addrJSONB models.StringMap
			err := shopsRows.Scan(
				&shop.ID, &shop.SellerID,
				&nameJSONB, &descJSONB, &addrJSONB,
				&shop.Slug, &shop.LogoURL, &shop.BannerURL,
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
				log.Printf("GetSellerDetail: Shop scan xatosi: %v", err)
				continue
			}
			shops = append(shops, shop)
		}

		writeJSON(w, http.StatusOK, SellerDetailResponse{
			Success: true,
			Seller: &SellerDetail{
				SellerProfile: &sellerProfile,
				User:          &user,
				Shops:         shops,
				ShopsCount:    len(shops),
			},
		})
	}
}

// UpdateSellerStatusRequest - Sotuvchi statusini yangilash so'rovi
type UpdateSellerStatusRequest struct {
	IsVerified bool `json:"is_verified"`
}

// UpdateSellerStatus godoc
// @Summary      Sotuvchi statusini yangilash (Admin)
// @Description  Sotuvchini tasdiqlash/tasdiqlashni bekor qilish
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path string true "Sotuvchi ID"
// @Param        request body UpdateSellerStatusRequest true "Status ma'lumotlari"
// @Success      200  {object}  models.AuthResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/sellers/{id}/status [put]
func UpdateSellerStatus(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat PUT metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Role tekshirish
		userRole := r.Header.Get("X-User-Role")
		if userRole != "admin" && userRole != "moderator" {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Bu sahifaga kirish huquqingiz yo'q",
			})
			return
		}

		// URL dan seller ID olish
		path := strings.TrimPrefix(r.URL.Path, "/api/admin/sellers/")
		path = strings.TrimSuffix(path, "/status")
		path = strings.TrimSuffix(path, "/")
		sellerID := path

		if sellerID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Sotuvchi ID kiritilishi shart",
			})
			return
		}

		var req UpdateSellerStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		// Seller mavjudligini tekshirish
		var exists bool
		err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM seller_profiles WHERE id = $1)`, sellerID).Scan(&exists)
		if err != nil || !exists {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Sotuvchi topilmadi",
			})
			return
		}

		// Statusni yangilash
		_, err = db.Exec(`
			UPDATE seller_profiles 
			SET is_verified = $1, updated_at = NOW()
			WHERE id = $2
		`, req.IsVerified, sellerID)

		if err != nil {
			log.Printf("UpdateSellerStatus: Update xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Statusni yangilashda xatolik",
			})
			return
		}

		log.Printf("✅ Sotuvchi statusi yangilandi: %s -> is_verified=%v", sellerID, req.IsVerified)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: fmt.Sprintf("Sotuvchi %s", map[bool]string{true: "tasdiqlandi", false: "tasdiqlash bekor qilindi"}[req.IsVerified]),
		})
	}
}

// AdminSellerHandler - /api/admin/sellers/{id} uchun method router
func AdminSellerHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/admin/sellers/")
		path = strings.TrimSuffix(path, "/")
		
		if strings.HasSuffix(r.URL.Path, "/status") {
			UpdateSellerStatus(db)(w, r)
		} else if path != "" {
			GetSellerDetail(db)(w, r)
		} else {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov",
			})
		}
	}
}
