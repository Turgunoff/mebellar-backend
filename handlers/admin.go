package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"mebellar-backend/models"
	"net/http"
	"strconv"
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

// UsersResponse - Foydalanuvchilar ro'yxati javobi
type UsersResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message,omitempty"`
	Users   []models.User `json:"users"`
	Total   int          `json:"total"`
	Page    int          `json:"page,omitempty"`
	Limit   int          `json:"limit,omitempty"`
}

// GetUsers godoc
// @Summary      Barcha foydalanuvchilarni olish (Admin)
// @Description  Admin panel uchun barcha foydalanuvchilar ro'yxatini qaytaradi
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        page query int false "Sahifa raqami (default: 1)"
// @Param        limit query int false "Har sahifadagi foydalanuvchilar soni (default: 10, max: 100)"
// @Param        role query string false "Role bo'yicha filter (customer, seller, admin, moderator)"
// @Success      200  {object}  UsersResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /admin/users [get]
func GetUsers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// User ID va Role middleware dan olingan
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
		roleFilter := r.URL.Query().Get("role")

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
		// Barcha foydalanuvchilarni olish (password_hash ni o'zgartirmaslik)
		baseQuery := `
			SELECT 
				id, full_name, phone, COALESCE(email, ''), 
				COALESCE(avatar_url, ''), COALESCE(role, 'customer'),
				COALESCE(onesignal_id, ''), created_at, updated_at
			FROM users
			WHERE is_active = true
		`
		countQuery := `SELECT COUNT(*) FROM users WHERE is_active = true`
		args := []interface{}{}
		argIndex := 1

		// Role filtri
		if roleFilter != "" {
			baseQuery += fmt.Sprintf(" AND COALESCE(role, 'customer') = $%d", argIndex)
			countQuery += fmt.Sprintf(" AND COALESCE(role, 'customer') = $%d", argIndex)
			args = append(args, roleFilter)
			argIndex++
		}

		// Jami sonni olish
		var total int
		err := db.QueryRow(countQuery, args...).Scan(&total)
		if err != nil {
			log.Printf("GetUsers: Count query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Foydalanuvchilarni olishda xatolik",
			})
			return
		}

		// Foydalanuvchilarni olish (pagination bilan)
		dataQuery := baseQuery + ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
		args = append(args, limit, offset)

		rows, err := db.Query(dataQuery, args...)
		if err != nil {
			log.Printf("GetUsers: Users query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Foydalanuvchilarni olishda xatolik",
			})
			return
		}
		defer rows.Close()

		users := []models.User{}
		for rows.Next() {
			var u models.User
			err := rows.Scan(
				&u.ID, &u.FullName, &u.Phone, &u.Email, &u.AvatarURL,
				&u.Role, &u.OneSignalID, &u.CreatedAt, &u.UpdatedAt,
			)
			if err != nil {
				log.Printf("GetUsers: User scan xatosi: %v", err)
				continue
			}
			users = append(users, u)
		}

		log.Printf("✅ %d ta foydalanuvchi topildi (sahifa %d)", len(users), page)

		writeJSON(w, http.StatusOK, UsersResponse{
			Success: true,
			Users:   users,
			Total:   total,
			Page:    page,
			Limit:   limit,
		})
	}
}
