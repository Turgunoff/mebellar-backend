package handlers

import (
	"database/sql"
	"encoding/json"
	"mebellar-backend/models"
	"net/http"
	"time"
)

// AdminDashboardStatsResponse - Admin dashboard statistikasi
type AdminDashboardStatsResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Data    *AdminDashboardStats   `json:"data,omitempty"`
}

// AdminDashboardStats - Dashboard statistikasi
type AdminDashboardStats struct {
	TotalUsers      int     `json:"total_users"`
	TotalSellers    int     `json:"total_sellers"`
	TotalProducts   int     `json:"total_products"`
	TotalOrders     int     `json:"total_orders"`
	TotalRevenue    float64 `json:"total_revenue"`
	ActiveUsers     int     `json:"active_users"`
	PendingOrders   int     `json:"pending_orders"`
	CompletedOrders int     `json:"completed_orders"`
	LastUpdated     string  `json:"last_updated"`
}

// GetAdminDashboardStats godoc
// @Summary      Admin dashboard statistikasi
// @Description  Admin panel uchun umumiy statistikalar (test endpoint)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  AdminDashboardStatsResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /admin/dashboard-stats [get]
func GetAdminDashboardStats(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// User ID va Role middleware dan olingan
		userID := r.Header.Get("X-User-ID")
		userRole := r.Header.Get("X-User-Role")

		if userID == "" {
			writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
				Success: false,
				Message: "Foydalanuvchi autentifikatsiya qilinmagan",
			})
			return
		}

		// Role tekshirish (middleware allaqachon tekshirgan, lekin qo'shimcha xavfsizlik)
		if userRole != "admin" && userRole != "moderator" {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Bu sahifaga kirish huquqingiz yo'q",
			})
			return
		}

		// Statistikani bazadan olish (hozircha dummy data)
		// Keyinchalik real statistikaga o'zgartiriladi
		stats := &AdminDashboardStats{
			TotalUsers:      150,
			TotalSellers:    25,
			TotalProducts:   500,
			TotalOrders:     1200,
			TotalRevenue:    125000000.50, // 125 million so'm
			ActiveUsers:     85,
			PendingOrders:   45,
			CompletedOrders: 1100,
			LastUpdated:     time.Now().Format("2006-01-02 15:04:05"),
		}

		// Real statistikani olish (keyinchalik)
		// TODO: Real queries yozish
		var totalUsers, totalSellers, totalProducts, totalOrders int
		var totalRevenue float64

		err := db.QueryRow(`
			SELECT 
				(SELECT COUNT(*) FROM users WHERE is_active = true),
				(SELECT COUNT(*) FROM users WHERE role = 'seller' AND is_active = true),
				(SELECT COUNT(*) FROM products WHERE is_active = true),
				(SELECT COUNT(*) FROM orders),
				(SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE status = 'completed')
		`).Scan(&totalUsers, &totalSellers, &totalProducts, &totalOrders, &totalRevenue)

		if err == nil {
			stats.TotalUsers = totalUsers
			stats.TotalSellers = totalSellers
			stats.TotalProducts = totalProducts
			stats.TotalOrders = totalOrders
			stats.TotalRevenue = totalRevenue
		}

		// Pending va completed orders
		var pendingOrders, completedOrders int
		err = db.QueryRow(`
			SELECT 
				(SELECT COUNT(*) FROM orders WHERE status = 'pending'),
				(SELECT COUNT(*) FROM orders WHERE status = 'completed')
		`).Scan(&pendingOrders, &completedOrders)

		if err == nil {
			stats.PendingOrders = pendingOrders
			stats.CompletedOrders = completedOrders
		}

		// Active users (oxirgi 30 kunda login qilganlar)
		var activeUsers int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM users 
			WHERE is_active = true 
			AND updated_at > NOW() - INTERVAL '30 days'
		`).Scan(&activeUsers)

		if err == nil {
			stats.ActiveUsers = activeUsers
		}

		writeJSON(w, http.StatusOK, AdminDashboardStatsResponse{
			Success: true,
			Data:    stats,
		})
	}
}
