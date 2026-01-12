package handlers

import (
	"database/sql"
	"log"
	"net/http"

	"mebellar-backend/models"
)

// GetRegions godoc
// @Summary      Barcha hududlarni olish
// @Description  Faol hududlar ro'yxatini qaytaradi (ordering bo'yicha tartiblangan)
// @Tags         regions
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.RegionsResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /regions [get]
func GetRegions(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		query := `
			SELECT id, name, COALESCE(code, ''), is_active, ordering
			FROM regions 
			WHERE is_active = true
			ORDER BY ordering ASC, name ASC
		`

		rows, err := db.Query(query)
		if err != nil {
			log.Printf("❌ Regions query xatosi: %v", err)
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
			err := rows.Scan(&r.ID, &r.Name, &r.Code, &r.IsActive, &r.Ordering)
			if err != nil {
				log.Printf("Region scan xatosi: %v", err)
				continue
			}
			regions = append(regions, r)
		}

		log.Printf("✅ %d ta hudud topildi", len(regions))

		writeJSON(w, http.StatusOK, models.RegionsResponse{
			Success: true,
			Regions: regions,
			Count:   len(regions),
		})
	}
}
