package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"mebellar-backend/models" // O'zimiz yaratgan model
)

// GetProducts - Barcha mahsulotlarni qaytaradi
func GetProducts(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Bazaga so'rov yuborish
		rows, err := db.Query("SELECT id, name, description, price, category_id, colors, image_url FROM products WHERE is_active = true")
		if err != nil {
			http.Error(w, "Bazadan o'qishda xatolik", http.StatusInternalServerError)
			log.Println(err)
			return
		}
		defer rows.Close()

		// 2. Ma'lumotlarni yig'ish
		var products []models.Product

		for rows.Next() {
			var p models.Product
			// Bazadagi ustunlarni structga joylash
			if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.CategoryID, &p.Colors, &p.ImageURL); err != nil {
				log.Println("Scan xatosi:", err)
				continue
			}
			products = append(products, p)
		}

		// 3. JSON formatda javob qaytarish
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(products)
	}
}
