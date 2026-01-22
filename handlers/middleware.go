package handlers

import (
	"database/sql"
	"log"
	"mebellar-backend/models"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// DeviceInfo - qurilma ma'lumotlarini saqlash uchun struct
type DeviceInfo struct {
	DeviceID   string
	AppType    string
	DeviceOS   string // iOS, Android
	OSVersion  string // 17.2, 14.0
	AppVersion string // 1.0.0, 1.0.0+12
}

// ExtractDeviceInfo - request headerlaridan device_id, app_type, device_os, os_version, app_version ni o'qish
// Bu middleware barcha so'rovlarda ishlaydi va keyingi handlerlar uchun ma'lumotlarni tayyorlaydi
func ExtractDeviceInfo(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// x-device-id headerini o'qish
		deviceID := r.Header.Get("x-device-id")
		if deviceID == "" {
			deviceID = r.Header.Get("X-Device-ID") // Case-insensitive fallback
		}

		// x-app-type headerini o'qish
		appType := r.Header.Get("x-app-type")
		if appType == "" {
			appType = r.Header.Get("X-App-Type") // Case-insensitive fallback
		}

		// Default qiymat - agar app_type berilmagan bo'lsa
		if appType == "" {
			appType = "client"
		}

		// Faqat ruxsat etilgan qiymatlar
		if appType != "client" && appType != "seller" && appType != "admin" {
			appType = "client"
		}

		// x-device-os headerini o'qish (iOS, Android)
		deviceOS := r.Header.Get("x-device-os")
		if deviceOS == "" {
			deviceOS = r.Header.Get("X-Device-OS")
		}

		// x-os-version headerini o'qish (17.2, 14.0)
		osVersion := r.Header.Get("x-os-version")
		if osVersion == "" {
			osVersion = r.Header.Get("X-OS-Version")
		}

		// x-app-version headerini o'qish (1.0.0, 1.0.0+12)
		appVersion := r.Header.Get("x-app-version")
		if appVersion == "" {
			appVersion = r.Header.Get("X-App-Version")
		}

		// Headerlar orqali keyingi handlerlarga uzatish
		r.Header.Set("X-Device-ID", deviceID)
		r.Header.Set("X-App-Type", appType)
		r.Header.Set("X-Device-OS", deviceOS)
		r.Header.Set("X-OS-Version", osVersion)
		r.Header.Set("X-App-Version", appVersion)

		next(w, r)
	}
}

// GetDeviceInfoFromRequest - requestdan device ma'lumotlarini olish (helper funksiya)
func GetDeviceInfoFromRequest(r *http.Request) DeviceInfo {
	return DeviceInfo{
		DeviceID:   r.Header.Get("X-Device-ID"),
		AppType:    r.Header.Get("X-App-Type"),
		DeviceOS:   r.Header.Get("X-Device-OS"),
		OSVersion:  r.Header.Get("X-OS-Version"),
		AppVersion: r.Header.Get("X-App-Version"),
	}
}

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
