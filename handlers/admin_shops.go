package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"mebellar-backend/models"
	"mebellar-backend/pkg/translator"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// ============================================
// ADMIN SHOP MANAGEMENT
// ============================================

// ListShops godoc
// @Summary      Barcha do'konlarni olish (Admin)
// @Description  Admin panel uchun barcha do'konlar ro'yxatini qaytaradi (filter bilan)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        page query int false "Sahifa raqami (default: 1)"
// @Param        limit query int false "Har sahifadagi do'konlar soni (default: 10, max: 100)"
// @Param        seller_id query string false "Sotuvchi ID bo'yicha filter"
// @Param        region_id query string false "Region ID bo'yicha filter"
// @Param        is_active query bool false "Faol/Nofaol filter"
// @Param        is_verified query bool false "Tasdiqlangan/Tasdiqlanmagan filter"
// @Success      200  {object}  models.ShopsResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/shops [get]
func ListShops(db *sql.DB) http.HandlerFunc {
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
		sellerID := r.URL.Query().Get("seller_id")
		regionID := r.URL.Query().Get("region_id")
		isActiveStr := r.URL.Query().Get("is_active")
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
				s.id, s.seller_id,
				COALESCE(s.name::text, '{}')::jsonb,
				COALESCE(s.description::text, '{}')::jsonb,
				COALESCE(s.address::text, '{}')::jsonb,
				COALESCE(s.slug, ''), COALESCE(s.logo_url, ''), COALESCE(s.banner_url, ''),
				COALESCE(s.phone, ''), s.latitude, s.longitude, 
				s.region_id,
				COALESCE(s.working_hours::text, '{}')::jsonb,
				s.is_active, s.is_verified, s.is_main, s.rating,
				s.created_at, s.updated_at,
				COALESCE(r.name, '') as region_name
			FROM shops s
			LEFT JOIN regions r ON s.region_id = r.id
			WHERE 1=1
		`
		countQuery := `SELECT COUNT(*) FROM shops WHERE 1=1`
		args := []interface{}{}
		argIndex := 1

		// Filtrlar
		if sellerID != "" {
			baseQuery += fmt.Sprintf(" AND s.seller_id = $%d", argIndex)
			countQuery += fmt.Sprintf(" AND seller_id = $%d", argIndex)
			args = append(args, sellerID)
			argIndex++
		}
		if regionID != "" {
			regionIDInt, err := strconv.Atoi(regionID)
			if err == nil {
				baseQuery += fmt.Sprintf(" AND s.region_id = $%d", argIndex)
				countQuery += fmt.Sprintf(" AND region_id = $%d", argIndex)
				args = append(args, regionIDInt)
				argIndex++
			}
		}
		if isActiveStr == "true" {
			baseQuery += fmt.Sprintf(" AND s.is_active = $%d", argIndex)
			countQuery += fmt.Sprintf(" AND is_active = $%d", argIndex)
			args = append(args, true)
			argIndex++
		} else if isActiveStr == "false" {
			baseQuery += fmt.Sprintf(" AND s.is_active = $%d", argIndex)
			countQuery += fmt.Sprintf(" AND is_active = $%d", argIndex)
			args = append(args, false)
			argIndex++
		}
		if isVerifiedStr == "true" {
			baseQuery += fmt.Sprintf(" AND s.is_verified = $%d", argIndex)
			countQuery += fmt.Sprintf(" AND is_verified = $%d", argIndex)
			args = append(args, true)
			argIndex++
		} else if isVerifiedStr == "false" {
			baseQuery += fmt.Sprintf(" AND s.is_verified = $%d", argIndex)
			countQuery += fmt.Sprintf(" AND is_verified = $%d", argIndex)
			args = append(args, false)
			argIndex++
		}

		// Jami sonni olish
		var total int
		err := db.QueryRow(countQuery, args...).Scan(&total)
		if err != nil {
			log.Printf("ListShops: Count query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Do'konlarni olishda xatolik",
			})
			return
		}

		// Do'konlarni olish (pagination bilan)
		dataQuery := baseQuery + ` ORDER BY s.created_at DESC LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
		args = append(args, limit, offset)

		rows, err := db.Query(dataQuery, args...)
		if err != nil {
			log.Printf("ListShops: Shops query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Do'konlarni olishda xatolik",
			})
			return
		}
		defer rows.Close()

		shops := []models.Shop{}
		for rows.Next() {
			var shop models.Shop
			var nameJSONB, descJSONB, addrJSONB models.StringMap
			var regionName string
			err := rows.Scan(
				&shop.ID, &shop.SellerID,
				&nameJSONB, &descJSONB, &addrJSONB,
				&shop.Slug, &shop.LogoURL, &shop.BannerURL,
				&shop.Phone, &shop.Latitude, &shop.Longitude, &shop.RegionID,
				&shop.WorkingHours,
				&shop.IsActive, &shop.IsVerified, &shop.IsMain, &shop.Rating,
				&shop.CreatedAt, &shop.UpdatedAt,
				&regionName,
			)
			if err == nil {
				shop.Name = nameJSONB
				shop.Description = descJSONB
				shop.Address = addrJSONB
			}
			if err != nil {
				log.Printf("ListShops: Shop scan xatosi: %v", err)
				continue
			}
			shops = append(shops, shop)
		}

		log.Printf("✅ %d ta do'kon topildi (sahifa %d)", len(shops), page)

		writeJSON(w, http.StatusOK, models.ShopsResponse{
			Success: true,
			Shops:   shops,
			Count:   total,
			Page:    page,
			Limit:   limit,
		})
	}
}

// CreateShopAdmin godoc
// @Summary      Yangi do'kon yaratish (Admin)
// @Description  Admin tomonidan yangi do'kon yaratish
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body models.CreateShopRequest true "Do'kon ma'lumotlari"
// @Success      201  {object}  models.ShopResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/shops [post]
func CreateShopAdmin(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
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

		var req models.CreateShopRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		// Validatsiya
		if req.Name == nil || len(req.Name) == 0 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Do'kon nomi (name) kiritilishi shart",
			})
			return
		}

		// Seller ID ni olish (query parametridan)
		sellerID := r.URL.Query().Get("seller_id")
		if sellerID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "seller_id parametri kiritilishi shart (query parameter)",
			})
			return
		}
		
		// Validate that name has at least Uzbek value
		if req.Name == nil || req.Name["uz"] == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Do'kon nomi (name.uz) kiritilishi shart",
			})
			return
		}

		// Seller mavjudligini tekshirish
		var sellerExists bool
		err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM seller_profiles WHERE id = $1)`, sellerID).Scan(&sellerExists)
		if err != nil || !sellerExists {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Sotuvchi topilmadi",
			})
			return
		}

		// Slug yaratish (English name dan)
		slug := models.GenerateSlugFromName(req.Name)
		if slug == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Do'kon nomidan slug yaratib bo'lmadi",
			})
			return
		}

		// Slug unikal ekanligini tekshirish
		var existingSlug string
		slugCheckErr := db.QueryRow(`SELECT slug FROM shops WHERE slug = $1`, slug).Scan(&existingSlug)
		if slugCheckErr == nil {
			// Slug mavjud, unikal qilish
			var count int
			db.QueryRow(`SELECT COUNT(*) FROM shops WHERE slug LIKE $1`, slug+"%").Scan(&count)
			slug = fmt.Sprintf("%s-%d", slug, count+1)
		}

		// is_main tekshirish - agar is_main = true bo'lsa, boshqa asosiy do'konlarni o'chirish
		if req.IsMain != nil && *req.IsMain {
			_, err = db.Exec(`
				UPDATE shops 
				SET is_main = false 
				WHERE seller_id = $1 AND is_main = true
			`, sellerID)
			if err != nil {
				log.Printf("CreateShop: Update is_main xatosi: %v", err)
			}
		}

		// Translate shop details using Gemini AI
		// Admin sends only Uzbek, we translate to Russian and English
		nameMap := make(models.StringMap)
		descMap := make(models.StringMap)
		addrMap := make(models.StringMap)
		
		// Set Uzbek values
		nameMap["uz"] = req.Name["uz"]
		if req.Description != nil && (*req.Description)["uz"] != "" {
			descMap["uz"] = (*req.Description)["uz"]
		}
		if req.Address != nil && (*req.Address)["uz"] != "" {
			addrMap["uz"] = (*req.Address)["uz"]
		}

		// Call Gemini translation service
		translatedName, translatedDesc, translatedAddr, err := translator.TranslateShop(
			nameMap["uz"],
			descMap["uz"],
			addrMap["uz"],
		)
		if err != nil {
			log.Printf("⚠️ Shop translation xatosi (fallback to uz only): %v", err)
			// Fallback: use only Uzbek if translation fails
			nameMap["ru"] = nameMap["uz"]
			nameMap["en"] = nameMap["uz"]
			if descMap["uz"] != "" {
				descMap["ru"] = descMap["uz"]
				descMap["en"] = descMap["uz"]
			}
			if addrMap["uz"] != "" {
				addrMap["ru"] = addrMap["uz"]
				addrMap["en"] = addrMap["uz"]
			}
		} else {
			// Use translated values
			nameMap = translatedName
			descMap = translatedDesc
			addrMap = translatedAddr
			log.Printf("✅ Shop tarjima muvaffaqiyatli: %s -> ru:%s, en:%s", nameMap["uz"], nameMap["ru"], nameMap["en"])
		}

		// Default qiymatlar
		isActive := true
		if req.IsActive != nil {
			isActive = *req.IsActive
		}
		isMain := false
		if req.IsMain != nil {
			isMain = *req.IsMain
		}

		// JSONB maydonlarni tayyorlash
		nameValue, _ := json.Marshal(nameMap)
		descValue, _ := json.Marshal(descMap)
		if len(descMap) == 0 {
			descValue = []byte("{}")
		}
		addrValue, _ := json.Marshal(addrMap)
		if len(addrMap) == 0 {
			addrValue = []byte("{}")
		}
		workingHoursValue := []byte("{}")
		if req.WorkingHours != nil {
			workingHoursValue, _ = json.Marshal(req.WorkingHours)
		}

		// Do'konni yaratish
		shopID := uuid.New().String()
		query := `
			INSERT INTO shops (
				id, seller_id, name, description, address, slug,
				phone, latitude, longitude, region_id, working_hours,
				is_active, is_main, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
			RETURNING id, seller_id,
				COALESCE(name::text, '{}')::jsonb,
				COALESCE(description::text, '{}')::jsonb,
				COALESCE(address::text, '{}')::jsonb,
				COALESCE(slug, ''), COALESCE(logo_url, ''), COALESCE(banner_url, ''),
				COALESCE(phone, ''), latitude, longitude, region_id,
				COALESCE(working_hours::text, '{}')::jsonb,
				is_active, is_verified, is_main, rating,
				created_at, updated_at
		`

		var shop models.Shop
		var nameJSONB, descJSONB, addrJSONB models.StringMap
		err = db.QueryRow(
			query,
			shopID, sellerID, nameValue, descValue, addrValue, slug,
			req.Phone, req.Latitude, req.Longitude, req.RegionID, workingHoursValue,
			isActive, isMain,
		).Scan(
			&shop.ID, &shop.SellerID,
			&nameJSONB, &descJSONB, &addrJSONB,
			&shop.Slug, &shop.LogoURL, &shop.BannerURL,
			&shop.Phone, &shop.Latitude, &shop.Longitude, &shop.RegionID,
			&shop.WorkingHours,
			&shop.IsActive, &shop.IsVerified, &shop.IsMain, &shop.Rating,
			&shop.CreatedAt, &shop.UpdatedAt,
		)

		if err != nil {
			log.Printf("CreateShop: Insert xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Do'kon yaratishda xatolik: " + err.Error(),
			})
			return
		}

		shop.Name = nameJSONB
		shop.Description = descJSONB
		shop.Address = addrJSONB

		log.Printf("✅ Do'kon yaratildi: %s (ID: %s)", shop.GetName("uz"), shopID)

		writeJSON(w, http.StatusCreated, models.ShopResponse{
			Success: true,
			Message: "Do'kon muvaffaqiyatli yaratildi",
			Shop:    &shop,
		})
	}
}

// UpdateShopAdmin godoc
// @Summary      Do'konni yangilash (Admin)
// @Description  Do'kon ma'lumotlarini yangilash
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path string true "Do'kon ID"
// @Param        request body models.UpdateShopRequest true "Yangilash ma'lumotlari"
// @Success      200  {object}  models.ShopResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/shops/{id} [put]
func UpdateShopAdmin(db *sql.DB) http.HandlerFunc {
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

		// URL dan shop ID olish
		path := strings.TrimPrefix(r.URL.Path, "/api/admin/shops/")
		shopID := strings.TrimSuffix(path, "/")

		if shopID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Do'kon ID kiritilishi shart",
			})
			return
		}

		// Shop mavjudligini tekshirish
		var sellerID string
		err := db.QueryRow(`SELECT seller_id FROM shops WHERE id = $1`, shopID).Scan(&sellerID)
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Do'kon topilmadi",
			})
			return
		}
		if err != nil {
			log.Printf("UpdateShop: Shop check xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Do'konni tekshirishda xatolik",
			})
			return
		}

		var req models.UpdateShopRequest
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

		if req.Name != nil {
			nameValue, _ := json.Marshal(*req.Name)
			updates = append(updates, fmt.Sprintf("name = $%d", argIndex))
			args = append(args, nameValue)
			argIndex++

			// Slug ni yangilash (agar name o'zgarsa)
			slug := models.GenerateSlugFromName(*req.Name)
			if slug != "" {
				// Slug unikal ekanligini tekshirish
				var existingSlug string
				slugCheckErr := db.QueryRow(`SELECT slug FROM shops WHERE slug = $1 AND id != $2`, slug, shopID).Scan(&existingSlug)
				if slugCheckErr == nil {
					var count int
					db.QueryRow(`SELECT COUNT(*) FROM shops WHERE slug LIKE $1 AND id != $2`, slug+"%", shopID).Scan(&count)
					slug = fmt.Sprintf("%s-%d", slug, count+1)
				}
				updates = append(updates, fmt.Sprintf("slug = $%d", argIndex))
				args = append(args, slug)
				argIndex++
			}
		}
		if req.Description != nil {
			descValue, _ := json.Marshal(*req.Description)
			updates = append(updates, fmt.Sprintf("description = $%d", argIndex))
			args = append(args, descValue)
			argIndex++
		}
		if req.Address != nil {
			addrValue, _ := json.Marshal(*req.Address)
			updates = append(updates, fmt.Sprintf("address = $%d", argIndex))
			args = append(args, addrValue)
			argIndex++
		}
		if req.Phone != nil {
			updates = append(updates, fmt.Sprintf("phone = $%d", argIndex))
			args = append(args, *req.Phone)
			argIndex++
		}
		if req.RegionID != nil {
			updates = append(updates, fmt.Sprintf("region_id = $%d", argIndex))
			args = append(args, *req.RegionID) // Already int type
			argIndex++
		}
		if req.Latitude != nil {
			updates = append(updates, fmt.Sprintf("latitude = $%d", argIndex))
			args = append(args, *req.Latitude)
			argIndex++
		}
		if req.Longitude != nil {
			updates = append(updates, fmt.Sprintf("longitude = $%d", argIndex))
			args = append(args, *req.Longitude)
			argIndex++
		}
		if req.LogoURL != nil {
			updates = append(updates, fmt.Sprintf("logo_url = $%d", argIndex))
			args = append(args, *req.LogoURL)
			argIndex++
		}
		if req.BannerURL != nil {
			updates = append(updates, fmt.Sprintf("banner_url = $%d", argIndex))
			args = append(args, *req.BannerURL)
			argIndex++
		}
		if req.WorkingHours != nil {
			workingHoursValue, _ := json.Marshal(req.WorkingHours)
			updates = append(updates, fmt.Sprintf("working_hours = $%d", argIndex))
			args = append(args, workingHoursValue)
			argIndex++
		}
		if req.IsMain != nil {
			// Agar is_main = true bo'lsa, boshqa asosiy do'konlarni o'chirish
			if *req.IsMain {
				_, err = db.Exec(`
					UPDATE shops 
					SET is_main = false 
					WHERE seller_id = $1 AND is_main = true AND id != $2
				`, sellerID, shopID)
				if err != nil {
					log.Printf("UpdateShop: Update is_main xatosi: %v", err)
				}
			}
			updates = append(updates, fmt.Sprintf("is_main = $%d", argIndex))
			args = append(args, *req.IsMain)
			argIndex++
		}
		if req.IsActive != nil {
			updates = append(updates, fmt.Sprintf("is_active = $%d", argIndex))
			args = append(args, *req.IsActive)
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

		query := "UPDATE shops SET " + strings.Join(updates, ", ") + " WHERE id = $" + strconv.Itoa(argIndex)

		_, err = db.Exec(query, args...)
		if err != nil {
			log.Printf("UpdateShop: Update xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Do'konni yangilashda xatolik: " + err.Error(),
			})
			return
		}

		log.Printf("✅ Do'kon yangilandi: %s", shopID)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: "Do'kon muvaffaqiyatli yangilandi",
		})
	}
}

// VerifyShopRequest - Do'kon tasdiqlash so'rovi
type VerifyShopRequest struct {
	IsVerified bool `json:"is_verified"`
}

// VerifyShop godoc
// @Summary      Do'konni tasdiqlash (Admin)
// @Description  Do'konni tasdiqlash/tasdiqlashni bekor qilish
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path string true "Do'kon ID"
// @Param        request body VerifyShopRequest true "Tasdiqlash ma'lumotlari"
// @Success      200  {object}  models.AuthResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/shops/{id}/verify [put]
func VerifyShop(db *sql.DB) http.HandlerFunc {
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

		// URL dan shop ID olish
		path := strings.TrimPrefix(r.URL.Path, "/api/admin/shops/")
		path = strings.TrimSuffix(path, "/verify")
		path = strings.TrimSuffix(path, "/")
		shopID := path

		if shopID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Do'kon ID kiritilishi shart",
			})
			return
		}

		var req VerifyShopRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		// Shop mavjudligini tekshirish
		var exists bool
		err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM shops WHERE id = $1)`, shopID).Scan(&exists)
		if err != nil || !exists {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Do'kon topilmadi",
			})
			return
		}

		// Statusni yangilash
		_, err = db.Exec(`
			UPDATE shops 
			SET is_verified = $1, updated_at = NOW()
			WHERE id = $2
		`, req.IsVerified, shopID)

		if err != nil {
			log.Printf("VerifyShop: Update xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Statusni yangilashda xatolik",
			})
			return
		}

		log.Printf("✅ Do'kon statusi yangilandi: %s -> is_verified=%v", shopID, req.IsVerified)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: fmt.Sprintf("Do'kon %s", map[bool]string{true: "tasdiqlandi", false: "tasdiqlash bekor qilindi"}[req.IsVerified]),
		})
	}
}

// AdminShopsHandler - /api/admin/shops uchun method router
func AdminShopsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			ListShops(db)(w, r)
		case http.MethodPost:
			CreateShopAdmin(db)(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Bu metod qo'llab-quvvatlanmaydi",
			})
		}
	}
}

// AdminShopItemHandler - /api/admin/shops/{id} uchun method router
func AdminShopItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/admin/shops/")
		path = strings.TrimSuffix(path, "/")
		
		if strings.HasSuffix(r.URL.Path, "/verify") {
			VerifyShop(db)(w, r)
		} else if path != "" {
			switch r.Method {
			case http.MethodPut:
				UpdateShopAdmin(db)(w, r)
			default:
				writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
					Success: false,
					Message: "Bu metod qo'llab-quvvatlanmaydi",
				})
			}
		} else {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov",
			})
		}
	}
}
