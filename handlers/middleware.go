package handlers

import (
	"database/sql"
	"log"
	"mebellar-backend/models"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// RequireRole - Role-based access control middleware
// Checks if the user's role matches one of the allowed roles
func RequireRole(db *sql.DB, allowedRoles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
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

			// Claims dan user_id va role olish
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				userID, ok := claims["user_id"].(string)
				if !ok {
					writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
						Success: false,
						Message: "Token ichida user_id topilmadi",
					})
					return
				}

				// Role ni olish (token dan yoki bazadan)
				userRole, ok := claims["role"].(string)
				if !ok || userRole == "" {
					// Agar token da role yo'q bo'lsa, bazadan olish
					err := db.QueryRow("SELECT COALESCE(role, 'customer') FROM users WHERE id = $1", userID).Scan(&userRole)
					if err != nil {
						log.Printf("Role query xatosi: %v", err)
						writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
							Success: false,
							Message: "Foydalanuvchi topilmadi",
						})
						return
					}
				}

				// Role ni tekshirish
				hasPermission := false
				for _, allowedRole := range allowedRoles {
					if userRole == allowedRole {
						hasPermission = true
						break
					}
				}

				if !hasPermission {
					writeJSON(w, http.StatusForbidden, models.AuthResponse{
						Success: false,
						Message: "Bu sahifaga kirish huquqingiz yo'q",
					})
					return
				}

				// User ID va Role ni header orqali keyingi handler ga uzatish
				r.Header.Set("X-User-ID", userID)
				r.Header.Set("X-User-Role", userRole)
				next(w, r)
			} else {
				writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
					Success: false,
					Message: "Token yaroqsiz",
				})
			}
		}
	}
}
