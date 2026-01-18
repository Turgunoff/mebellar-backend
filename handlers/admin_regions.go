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
// ADMIN REGION MANAGEMENT
// ============================================

// GetAdminRegions godoc
// @Summary      Barcha hududlarni olish (Admin)
// @Description  Admin panel uchun barcha hududlar ro'yxatini qaytaradi (faol va nofaol)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        is_active query bool false "Faol/Nofaol filter"
// @Success      200  {object}  models.RegionsResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/regions [get]
func GetAdminRegions(db *sql.DB) http.HandlerFunc {
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

		// Filter
		isActiveStr := r.URL.Query().Get("is_active")

		query := `
			SELECT id, COALESCE(name::text, '{}')::jsonb, COALESCE(code, ''), 
				   is_active, ordering, created_at, COALESCE(updated_at, created_at)
			FROM regions 
			WHERE 1=1
		`
		args := []interface{}{}
		argIndex := 1

		if isActiveStr == "true" {
			query += fmt.Sprintf(" AND is_active = $%d", argIndex)
			args = append(args, true)
			argIndex++
		} else if isActiveStr == "false" {
			query += fmt.Sprintf(" AND is_active = $%d", argIndex)
			args = append(args, false)
			argIndex++
		}

		query += " ORDER BY ordering ASC"

		rows, err := db.Query(query, args...)
		if err != nil {
			log.Printf("❌ Admin regions query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Hududlarni olishda xatolik",
			})
			return
		}
		defer rows.Close()

		regions := []models.Region{}
		for rows.Next() {
			var r models.Region
			var nameJSONB models.StringMap
			err := rows.Scan(&r.ID, &nameJSONB, &r.Code, &r.IsActive, &r.Ordering, &r.CreatedAt, &r.UpdatedAt)
			if err != nil {
				log.Printf("Region scan xatosi: %v", err)
				continue
			}
			r.NameJSONB = nameJSONB
			// Extract legacy name from JSONB (prefer uz, then first available)
			if nameJSONB != nil {
				if uzName, ok := nameJSONB["uz"]; ok && uzName != "" {
					r.Name = uzName
				} else {
					// Get first available name
					for _, name := range nameJSONB {
						r.Name = name
						break
					}
				}
			}
			regions = append(regions, r)
		}

		log.Printf("✅ Admin: %d ta hudud topildi", len(regions))

		writeJSON(w, http.StatusOK, models.RegionsResponse{
			Success: true,
			Regions: regions,
			Count:   len(regions),
		})
	}
}

// CreateRegion godoc
// @Summary      Yangi hudud yaratish (Admin)
// @Description  Admin tomonidan yangi hudud yaratish
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body models.CreateRegionRequest true "Hudud ma'lumotlari"
// @Success      201  {object}  models.RegionResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/regions [post]
func CreateRegion(db *sql.DB) http.HandlerFunc {
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
		if userRole != "admin" {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Faqat admin hudud yaratishi mumkin",
			})
			return
		}

		var req models.CreateRegionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		// Validatsiya
		if req.Name == nil || len(req.Name) == 0 || req.Name["uz"] == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Hudud nomi (name.uz) kiritilishi shart",
			})
			return
		}

		if req.Code == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Hudud kodi (code) kiritilishi shart",
			})
			return
		}

		// Code unikal ekanligini tekshirish
		var existingCode string
		err := db.QueryRow(`SELECT code FROM regions WHERE code = $1`, req.Code).Scan(&existingCode)
		if err == nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Bu kod allaqachon mavjud",
			})
			return
		}

		// Default qiymatlar
		isActive := true
		if req.IsActive != nil {
			isActive = *req.IsActive
		}

		// JSONB maydonlarni tayyorlash
		nameValue, _ := json.Marshal(req.Name)

		// Insert (name column is JSONB type)
		var region models.Region
		var nameJSONB models.StringMap
		query := `
			INSERT INTO regions (name, code, is_active, ordering, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
			RETURNING id, COALESCE(name::text, '{}')::jsonb, COALESCE(code, ''), 
					  is_active, ordering, created_at, updated_at
		`

		err = db.QueryRow(query, nameValue, req.Code, isActive, req.Ordering).Scan(
			&region.ID, &nameJSONB, &region.Code,
			&region.IsActive, &region.Ordering, &region.CreatedAt, &region.UpdatedAt,
		)
		if err != nil {
			log.Printf("CreateRegion error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Hudud yaratishda xatolik: " + err.Error(),
			})
			return
		}
		region.NameJSONB = nameJSONB
		// Extract legacy name from JSONB
		if nameJSONB != nil {
			if uzName, ok := nameJSONB["uz"]; ok && uzName != "" {
				region.Name = uzName
			} else {
				for _, name := range nameJSONB {
					region.Name = name
					break
				}
			}
		}

		log.Printf("✅ Yangi hudud yaratildi: %s (ID: %d, Code: %s)", region.Name, region.ID, region.Code)

		writeJSON(w, http.StatusCreated, models.RegionResponse{
			Success: true,
			Message: "Hudud muvaffaqiyatli yaratildi",
			Region:  &region,
		})
	}
}

// UpdateRegion godoc
// @Summary      Hududni yangilash (Admin)
// @Description  Hudud ma'lumotlarini yangilash
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "Hudud ID"
// @Param        request body models.UpdateRegionRequest true "Yangilash ma'lumotlari"
// @Success      200  {object}  models.RegionResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/regions/{id} [put]
func UpdateRegion(db *sql.DB) http.HandlerFunc {
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
		if userRole != "admin" {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Faqat admin hududni yangilashi mumkin",
			})
			return
		}

		// URL dan region ID olish
		path := strings.TrimPrefix(r.URL.Path, "/api/admin/regions/")
		path = strings.TrimSuffix(path, "/status")
		path = strings.TrimSuffix(path, "/")
		regionIDStr := path

		regionID, err := strconv.Atoi(regionIDStr)
		if err != nil || regionID <= 0 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri hudud ID",
			})
			return
		}

		// Region mavjudligini tekshirish
		var exists bool
		err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM regions WHERE id = $1)`, regionID).Scan(&exists)
		if err != nil || !exists {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Hudud topilmadi",
			})
			return
		}

		var req models.UpdateRegionRequest
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
			// name column is JSONB type, so update it directly
			updates = append(updates, fmt.Sprintf("name = $%d", argIndex))
			args = append(args, nameValue)
			argIndex++
		}
		if req.Code != nil {
			// Code unikal ekanligini tekshirish
			var existingID int
			err := db.QueryRow(`SELECT id FROM regions WHERE code = $1 AND id != $2`, *req.Code, regionID).Scan(&existingID)
			if err == nil {
				writeJSON(w, http.StatusBadRequest, models.AuthResponse{
					Success: false,
					Message: "Bu kod boshqa hududda mavjud",
				})
				return
			}
			updates = append(updates, fmt.Sprintf("code = $%d", argIndex))
			args = append(args, *req.Code)
			argIndex++
		}
		if req.Ordering != nil {
			updates = append(updates, fmt.Sprintf("ordering = $%d", argIndex))
			args = append(args, *req.Ordering)
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
		args = append(args, regionID)

		query := "UPDATE regions SET " + strings.Join(updates, ", ") + " WHERE id = $" + strconv.Itoa(argIndex)

		_, err = db.Exec(query, args...)
		if err != nil {
			log.Printf("UpdateRegion error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Hududni yangilashda xatolik: " + err.Error(),
			})
			return
		}

		log.Printf("✅ Hudud yangilandi: ID=%d", regionID)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: "Hudud muvaffaqiyatli yangilandi",
		})
	}
}

// DeleteRegion godoc
// @Summary      Hududni o'chirish (Admin)
// @Description  Hududni o'chirish (bog'langan do'konlar bo'lsa xatolik)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "Hudud ID"
// @Success      200  {object}  models.AuthResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      409  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/regions/{id} [delete]
func DeleteRegion(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat DELETE metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Role tekshirish
		userRole := r.Header.Get("X-User-Role")
		if userRole != "admin" {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Faqat admin hududni o'chirishi mumkin",
			})
			return
		}

		// URL dan region ID olish
		path := strings.TrimPrefix(r.URL.Path, "/api/admin/regions/")
		path = strings.TrimSuffix(path, "/")
		regionIDStr := path

		regionID, err := strconv.Atoi(regionIDStr)
		if err != nil || regionID <= 0 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri hudud ID",
			})
			return
		}

		// Region mavjudligini tekshirish
		var exists bool
		err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM regions WHERE id = $1)`, regionID).Scan(&exists)
		if err != nil || !exists {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Hudud topilmadi",
			})
			return
		}

		// Bog'langan do'konlar borligini tekshirish
		var shopsCount int
		err = db.QueryRow(`SELECT COUNT(*) FROM shops WHERE region_id = $1`, regionID).Scan(&shopsCount)
		if err == nil && shopsCount > 0 {
			writeJSON(w, http.StatusConflict, models.AuthResponse{
				Success: false,
				Message: fmt.Sprintf("Bu hududga %d ta do'kon bog'langan. Avval do'konlarni boshqa hududga o'tkazing.", shopsCount),
			})
			return
		}

		// O'chirish
		_, err = db.Exec(`DELETE FROM regions WHERE id = $1`, regionID)
		if err != nil {
			log.Printf("DeleteRegion error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Hududni o'chirishda xatolik",
			})
			return
		}

		log.Printf("🗑️ Hudud o'chirildi: ID=%d", regionID)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: "Hudud muvaffaqiyatli o'chirildi",
		})
	}
}

// ToggleRegionStatus godoc
// @Summary      Hudud statusini o'zgartirish (Admin)
// @Description  Hududni faol/nofaol qilish
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "Hudud ID"
// @Success      200  {object}  models.AuthResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/regions/{id}/status [put]
func ToggleRegionStatus(db *sql.DB) http.HandlerFunc {
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

		// URL dan region ID olish
		path := strings.TrimPrefix(r.URL.Path, "/api/admin/regions/")
		path = strings.TrimSuffix(path, "/status")
		path = strings.TrimSuffix(path, "/")
		regionIDStr := path

		regionID, err := strconv.Atoi(regionIDStr)
		if err != nil || regionID <= 0 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri hudud ID",
			})
			return
		}

		// Joriy statusni olish va o'zgartirish
		var currentStatus bool
		err = db.QueryRow(`SELECT is_active FROM regions WHERE id = $1`, regionID).Scan(&currentStatus)
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Hudud topilmadi",
			})
			return
		}
		if err != nil {
			log.Printf("ToggleRegionStatus error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Statusni olishda xatolik",
			})
			return
		}

		// Statusni o'zgartirish
		newStatus := !currentStatus
		_, err = db.Exec(`UPDATE regions SET is_active = $1, updated_at = NOW() WHERE id = $2`, newStatus, regionID)
		if err != nil {
			log.Printf("ToggleRegionStatus update error: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Statusni yangilashda xatolik",
			})
			return
		}

		statusText := "faollashtirildi"
		if !newStatus {
			statusText = "nofaollashtirildi"
		}

		log.Printf("✅ Hudud statusi o'zgartirildi: ID=%d -> is_active=%v", regionID, newStatus)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: fmt.Sprintf("Hudud %s", statusText),
		})
	}
}

// AdminRegionsHandler - /api/admin/regions uchun method router
func AdminRegionsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			GetAdminRegions(db)(w, r)
		case http.MethodPost:
			CreateRegion(db)(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Bu metod qo'llab-quvvatlanmaydi",
			})
		}
	}
}

// AdminRegionItemHandler - /api/admin/regions/{id} uchun method router
func AdminRegionItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/admin/regions/")
		path = strings.TrimSuffix(path, "/")

		if strings.HasSuffix(r.URL.Path, "/status") {
			ToggleRegionStatus(db)(w, r)
		} else if path != "" {
			switch r.Method {
			case http.MethodPut:
				UpdateRegion(db)(w, r)
			case http.MethodDelete:
				DeleteRegion(db)(w, r)
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
