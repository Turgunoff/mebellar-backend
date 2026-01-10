package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"mebellar-backend/models"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// ============================================
// JWT MIDDLEWARE
// ============================================

// JWTMiddleware - JWT token ni tekshiradi va user_id ni context ga qo'shadi
func JWTMiddleware(db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// OPTIONS so'rovlarini o'tkazib yuborish
		if r.Method == http.MethodOptions {
			next(w, r)
			return
		}

		// Authorization header dan token olish
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
				Success: false,
				Message: "Authorization header topilmadi",
			})
			return
		}

		// "Bearer " prefiksini olib tashlash
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri token formati. 'Bearer {token}' formatida kiriting",
			})
			return
		}

		// Token ni tekshirish
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Signing method tekshirish
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return jwtSecretKey, nil
		})

		if err != nil {
			log.Printf("JWT parse xatosi: %v", err)
			writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri yoki muddati o'tgan token",
			})
			return
		}

		// Claims dan user_id olish
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			userID, ok := claims["user_id"].(string)
			if !ok {
				writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
					Success: false,
					Message: "Token ichida user_id topilmadi",
				})
				return
			}

			// User ID ni header orqali keyingi handler ga uzatish
			r.Header.Set("X-User-ID", userID)
			next(w, r)
		} else {
			writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
				Success: false,
				Message: "Token yaroqsiz",
			})
		}
	}
}

// ============================================
// GET PROFILE
// ============================================

// ProfileResponse - profil javobi
type ProfileResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message,omitempty"`
	User    *models.User `json:"user,omitempty"`
}

// GetProfile godoc
// @Summary      Foydalanuvchi profilini olish
// @Description  JWT token orqali autentifikatsiya qilingan foydalanuvchining ma'lumotlarini qaytaradi
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  ProfileResponse
// @Failure      401  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /user/me [get]
func GetProfile(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Middleware dan user_id olish
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
				Success: false,
				Message: "Foydalanuvchi autentifikatsiya qilinmagan",
			})
			return
		}

		// Bazadan foydalanuvchini olish
		var user models.User
		err := db.QueryRow(`
			SELECT id, full_name, phone, created_at, updated_at
			FROM users WHERE id = $1
		`, userID).Scan(&user.ID, &user.FullName, &user.Phone, &user.CreatedAt, &user.UpdatedAt)

		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Foydalanuvchi topilmadi",
			})
			return
		}

		if err != nil {
			log.Printf("Profile query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Server xatosi",
			})
			return
		}

		writeJSON(w, http.StatusOK, ProfileResponse{
			Success: true,
			User:    &user,
		})
	}
}

// ============================================
// UPDATE PROFILE
// ============================================

// UpdateProfileRequest - profil yangilash so'rovi
type UpdateProfileRequest struct {
	FullName string `json:"full_name"`
}

// UpdateProfile godoc
// @Summary      Profil ma'lumotlarini yangilash
// @Description  Foydalanuvchining ismini yangilaydi
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body UpdateProfileRequest true "Yangi ma'lumotlar"
// @Success      200  {object}  ProfileResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      401  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /user/me [put]
func UpdateProfile(db *sql.DB) http.HandlerFunc {
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
				Message: "Foydalanuvchi autentifikatsiya qilinmagan",
			})
			return
		}

		var req UpdateProfileRequest
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		if req.FullName == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Ism kiritilishi shart",
			})
			return
		}

		// Profilni yangilash
		_, err := db.Exec(`
			UPDATE users SET full_name = $1, updated_at = NOW()
			WHERE id = $2
		`, req.FullName, userID)

		if err != nil {
			log.Printf("Profile update xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Profilni yangilashda xatolik",
			})
			return
		}

		// Yangilangan profilni qaytarish
		var user models.User
		err = db.QueryRow(`
			SELECT id, full_name, phone, created_at, updated_at
			FROM users WHERE id = $1
		`, userID).Scan(&user.ID, &user.FullName, &user.Phone, &user.CreatedAt, &user.UpdatedAt)

		if err != nil {
			log.Printf("Profile fetch xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Server xatosi",
			})
			return
		}

		writeJSON(w, http.StatusOK, ProfileResponse{
			Success: true,
			Message: "Profil muvaffaqiyatli yangilandi",
			User:    &user,
		})
	}
}

// ============================================
// DELETE ACCOUNT
// ============================================

// DeleteAccount godoc
// @Summary      Hisobni o'chirish
// @Description  Foydalanuvchi hisobini butunlay o'chiradi (App Store talabi)
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  models.AuthResponse
// @Failure      401  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /user/me [delete]
func DeleteAccount(db *sql.DB) http.HandlerFunc {
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
				Message: "Foydalanuvchi autentifikatsiya qilinmagan",
			})
			return
		}

		// Foydalanuvchini o'chirish
		result, err := db.Exec("DELETE FROM users WHERE id = $1", userID)
		if err != nil {
			log.Printf("Delete account xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Hisobni o'chirishda xatolik",
			})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Foydalanuvchi topilmadi",
			})
			return
		}

		log.Printf("🗑️ User deleted: %s", userID)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: "Hisobingiz muvaffaqiyatli o'chirildi",
		})
	}
}

// ============================================
// HELPER FUNCTIONS
// ============================================

// decodeJSON - JSON ni decode qiladi
func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
