package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"mebellar-backend/models"
)

// GetBanners godoc
// @Summary Barcha faol bannerlarni olish
// @Description Home sahifasi uchun barcha faol bannerlarni qaytaradi
// @Tags banners
// @Accept json
// @Produce json
// @Success 200 {object} models.BannersResponse
// @Failure 500 {object} models.BannersResponse
// @Router /banners [get]
func GetBanners(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(models.BannersResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Faqat faol bannerlarni sort_order bo'yicha tartiblangan holda olish
		query := `
			SELECT 
				id, 
				title, 
				subtitle, 
				image_url, 
				target_type, 
				target_id, 
				sort_order, 
				is_active, 
				created_at
			FROM banners 
			WHERE is_active = true 
			ORDER BY sort_order ASC
		`

		rows, err := db.Query(query)
		if err != nil {
			log.Printf("Bannerlarni olishda xatolik: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.BannersResponse{
				Success: false,
				Message: "Bannerlarni olishda xatolik yuz berdi",
			})
			return
		}
		defer rows.Close()

		var banners []models.Banner

		for rows.Next() {
			var banner models.Banner
			var titleJSON, subtitleJSON []byte
			var targetID sql.NullString

			err := rows.Scan(
				&banner.ID,
				&titleJSON,
				&subtitleJSON,
				&banner.ImageURL,
				&banner.TargetType,
				&targetID,
				&banner.SortOrder,
				&banner.IsActive,
				&banner.CreatedAt,
			)
			if err != nil {
				log.Printf("Banner scan xatolik: %v", err)
				continue
			}

			// Parse JSONB fields
			if len(titleJSON) > 0 {
				if err := json.Unmarshal(titleJSON, &banner.Title); err != nil {
					log.Printf("Title JSON parse xatolik: %v", err)
					banner.Title = map[string]string{}
				}
			} else {
				banner.Title = map[string]string{}
			}

			if len(subtitleJSON) > 0 {
				if err := json.Unmarshal(subtitleJSON, &banner.Subtitle); err != nil {
					log.Printf("Subtitle JSON parse xatolik: %v", err)
					banner.Subtitle = map[string]string{}
				}
			} else {
				banner.Subtitle = map[string]string{}
			}

			// Handle nullable target_id
			if targetID.Valid {
				banner.TargetID = &targetID.String
			}

			banners = append(banners, banner)
		}

		if err := rows.Err(); err != nil {
			log.Printf("Rows iteration xatolik: %v", err)
		}

		// Agar bannerlar bo'sh bo'lsa, bo'sh array qaytarish
		if banners == nil {
			banners = []models.Banner{}
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(models.BannersResponse{
			Success: true,
			Banners: banners,
			Count:   len(banners),
		})
	}
}
