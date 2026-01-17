package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mebellar-backend/models"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ============================================
// OTP STORE - Telefon va Email o'zgartirish uchun
// ============================================

// OTPData - OTP ma'lumotlari
type OTPData struct {
	Code      string
	ExpiresAt time.Time
	UserID    string
}

var (
	phoneOTPStore = make(map[string]OTPData) // key: new_phone
	emailOTPStore = make(map[string]OTPData) // key: new_email
	otpMutex      sync.RWMutex
)

// generateOTP is defined in auth.go - reusing it here

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
			SELECT id, full_name, phone, COALESCE(email, ''), COALESCE(avatar_url, ''), COALESCE(role, 'customer'), COALESCE(onesignal_id, ''), created_at, updated_at
			FROM users WHERE id = $1
		`, userID).Scan(&user.ID, &user.FullName, &user.Phone, &user.Email, &user.AvatarURL, &user.Role, &user.OneSignalID, &user.CreatedAt, &user.UpdatedAt)

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
// UPDATE PROFILE (Multipart with Avatar)
// ============================================

// UpdateProfile godoc
// @Summary      Profil ma'lumotlarini yangilash (ism va avatar)
// @Description  Foydalanuvchining ismini va rasmini yangilaydi. multipart/form-data formatida
// @Tags         user
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        full_name formData string false "To'liq ism"
// @Param        avatar formData file false "Avatar rasmi"
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

		// Parse multipart form (10MB limit)
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			log.Printf("ParseMultipartForm xatosi: %v", err)
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "So'rovni o'qib bo'lmadi",
			})
			return
		}

		// Form fieldlarini olish
		fullName := r.FormValue("full_name")
		oneSignalID := r.FormValue("onesignal_id")

		// Avatar faylini tekshirish
		var avatarURL *string
		file, handler, err := r.FormFile("avatar")
		if err == nil {
			defer file.Close()

			// uploads/avatars papkasini yaratish
			uploadDir := "uploads/avatars"
			if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
				log.Printf("Papka yaratishda xatolik: %v", err)
				writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
					Success: false,
					Message: "Server xatosi",
				})
				return
			}

			// Unikal fayl nomi yaratish
			ext := filepath.Ext(handler.Filename)
			newFileName := fmt.Sprintf("%s_%d%s", userID, time.Now().UnixNano(), ext)
			filePath := filepath.Join(uploadDir, newFileName)

			// Faylni saqlash
			dst, err := os.Create(filePath)
			if err != nil {
				log.Printf("Fayl yaratishda xatolik: %v", err)
				writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
					Success: false,
					Message: "Fayl saqlashda xatolik",
				})
				return
			}
			defer dst.Close()

			if _, err := io.Copy(dst, file); err != nil {
				log.Printf("Fayl nusxalashda xatolik: %v", err)
				writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
					Success: false,
					Message: "Fayl saqlashda xatolik",
				})
				return
			}

			// Avatar URL yaratish
			url := "/" + filePath
			avatarURL = &url
			log.Printf("✅ Avatar saqlandi: %s", filePath)
		}

		// SQL so'rov yaratish
		var query string
		var args []interface{}
		updateFields := []string{}
		argIndex := 1

		if fullName != "" {
			updateFields = append(updateFields, fmt.Sprintf("full_name = $%d", argIndex))
			args = append(args, fullName)
			argIndex++
		}

		if avatarURL != nil {
			updateFields = append(updateFields, fmt.Sprintf("avatar_url = $%d", argIndex))
			args = append(args, *avatarURL)
			argIndex++
		}

		if oneSignalID != "" {
			updateFields = append(updateFields, fmt.Sprintf("onesignal_id = $%d", argIndex))
			args = append(args, oneSignalID)
			argIndex++
		}

		if len(updateFields) == 0 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Hech narsa o'zgartirilmadi",
			})
			return
		}

		updateFields = append(updateFields, fmt.Sprintf("updated_at = NOW()"))
		args = append(args, userID)
		query = fmt.Sprintf("UPDATE users SET %s WHERE id = $%d", strings.Join(updateFields, ", "), argIndex)

		// Profilni yangilash
		_, err = db.Exec(query, args...)
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
			SELECT id, full_name, phone, COALESCE(email, ''), COALESCE(avatar_url, ''), COALESCE(onesignal_id, ''), created_at, updated_at
			FROM users WHERE id = $1
		`, userID).Scan(&user.ID, &user.FullName, &user.Phone, &user.Email, &user.AvatarURL, &user.OneSignalID, &user.CreatedAt, &user.UpdatedAt)

		if err != nil {
			log.Printf("Profile fetch xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Server xatosi",
			})
			return
		}

		log.Printf("✅ Profil yangilandi: %s", user.FullName)

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
// CHANGE PHONE - OTP bilan telefon o'zgartirish
// ============================================

// ChangePhoneRequest - telefon o'zgartirish so'rovi
type ChangePhoneRequest struct {
	NewPhone string `json:"new_phone"`
}

// VerifyPhoneChangeRequest - telefon tasdiqlash so'rovi
type VerifyPhoneChangeRequest struct {
	NewPhone string `json:"new_phone"`
	Code     string `json:"code"`
}

// RequestPhoneChange godoc
// @Summary      Telefon o'zgartirish - OTP yuborish
// @Description  Yangi telefon raqamiga OTP yuboradi
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body ChangePhoneRequest true "Yangi telefon raqami"
// @Success      200  {object}  models.AuthResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      401  {object}  models.AuthResponse
// @Router       /user/change-phone/request [post]
func RequestPhoneChange(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
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

		var req ChangePhoneRequest
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		if req.NewPhone == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Telefon raqam kiritilishi shart",
			})
			return
		}

		// Telefon allaqachon ro'yxatdan o'tganmi tekshirish
		var existingID string
		err := db.QueryRow("SELECT id FROM users WHERE phone = $1", req.NewPhone).Scan(&existingID)
		if err == nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Bu telefon raqam allaqachon ro'yxatdan o'tgan",
			})
			return
		}

		// OTP yaratish
		code := generateOTP()
		expiresAt := time.Now().Add(5 * time.Minute)

		otpMutex.Lock()
		phoneOTPStore[req.NewPhone] = OTPData{
			Code:      code,
			ExpiresAt: expiresAt,
			UserID:    userID,
		}
		otpMutex.Unlock()

		// Mock SMS - konsolga chiqarish
		log.Printf("📱 [MOCK SMS] Telefon o'zgartirish kodi: %s -> %s", req.NewPhone, code)
		log.Printf("⏰ Kod amal qilish muddati: %s", expiresAt.Format("15:04:05"))

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: fmt.Sprintf("Tasdiqlash kodi %s raqamiga yuborildi", req.NewPhone),
		})
	}
}

// VerifyPhoneChange godoc
// @Summary      Telefon o'zgartirish - OTP tasdiqlash
// @Description  OTP kodi orqali yangi telefonni tasdiqlash
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body VerifyPhoneChangeRequest true "Yangi telefon va kod"
// @Success      200  {object}  ProfileResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      401  {object}  models.AuthResponse
// @Router       /user/change-phone/verify [post]
func VerifyPhoneChange(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
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

		var req VerifyPhoneChangeRequest
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		// OTP ni tekshirish
		otpMutex.RLock()
		otpData, exists := phoneOTPStore[req.NewPhone]
		otpMutex.RUnlock()

		if !exists {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Avval tasdiqlash kodi so'rang",
			})
			return
		}

		// Muddati o'tganmi
		if time.Now().After(otpData.ExpiresAt) {
			otpMutex.Lock()
			delete(phoneOTPStore, req.NewPhone)
			otpMutex.Unlock()
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Tasdiqlash kodi muddati o'tgan",
			})
			return
		}

		// Kod to'g'rimi
		if otpData.Code != req.Code {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri tasdiqlash kodi",
			})
			return
		}

		// UserID mos kelishini tekshirish
		if otpData.UserID != userID {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov",
			})
			return
		}

		// Telefonni yangilash
		_, err := db.Exec(`UPDATE users SET phone = $1, updated_at = NOW() WHERE id = $2`, req.NewPhone, userID)
		if err != nil {
			log.Printf("Phone update xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Telefon yangilashda xatolik",
			})
			return
		}

		// OTP ni o'chirish
		otpMutex.Lock()
		delete(phoneOTPStore, req.NewPhone)
		otpMutex.Unlock()

		// Yangilangan profilni qaytarish
		var user models.User
		err = db.QueryRow(`
			SELECT id, full_name, phone, COALESCE(email, ''), COALESCE(avatar_url, ''), created_at, updated_at
			FROM users WHERE id = $1
		`, userID).Scan(&user.ID, &user.FullName, &user.Phone, &user.Email, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)

		if err != nil {
			log.Printf("Profile fetch xatosi: %v", err)
		}

		log.Printf("✅ Telefon o'zgartirildi: %s", req.NewPhone)

		writeJSON(w, http.StatusOK, ProfileResponse{
			Success: true,
			Message: "Telefon raqam muvaffaqiyatli o'zgartirildi",
			User:    &user,
		})
	}
}

// ============================================
// CHANGE EMAIL - OTP bilan email o'zgartirish
// ============================================

// ChangeEmailRequest - email o'zgartirish so'rovi
type ChangeEmailRequest struct {
	NewEmail string `json:"new_email"`
}

// VerifyEmailChangeRequest - email tasdiqlash so'rovi
type VerifyEmailChangeRequest struct {
	NewEmail string `json:"new_email"`
	Code     string `json:"code"`
}

// RequestEmailChange godoc
// @Summary      Email o'zgartirish - OTP yuborish
// @Description  Yangi email manziliga OTP yuboradi
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body ChangeEmailRequest true "Yangi email manzili"
// @Success      200  {object}  models.AuthResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      401  {object}  models.AuthResponse
// @Router       /user/change-email/request [post]
func RequestEmailChange(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
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

		var req ChangeEmailRequest
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		if req.NewEmail == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Email manzil kiritilishi shart",
			})
			return
		}

		// Email allaqachon ro'yxatdan o'tganmi tekshirish
		var existingID string
		err := db.QueryRow("SELECT id FROM users WHERE email = $1", req.NewEmail).Scan(&existingID)
		if err == nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Bu email manzil allaqachon ro'yxatdan o'tgan",
			})
			return
		}

		// OTP yaratish
		code := generateOTP()
		expiresAt := time.Now().Add(5 * time.Minute)

		otpMutex.Lock()
		emailOTPStore[req.NewEmail] = OTPData{
			Code:      code,
			ExpiresAt: expiresAt,
			UserID:    userID,
		}
		otpMutex.Unlock()

		// Mock Email - konsolga chiqarish
		log.Printf("📧 [MOCK EMAIL] Email o'zgartirish kodi: %s -> %s", req.NewEmail, code)
		log.Printf("⏰ Kod amal qilish muddati: %s", expiresAt.Format("15:04:05"))

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: fmt.Sprintf("Tasdiqlash kodi %s manziliga yuborildi", req.NewEmail),
		})
	}
}

// VerifyEmailChange godoc
// @Summary      Email o'zgartirish - OTP tasdiqlash
// @Description  OTP kodi orqali yangi emailni tasdiqlash
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body VerifyEmailChangeRequest true "Yangi email va kod"
// @Success      200  {object}  ProfileResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      401  {object}  models.AuthResponse
// @Router       /user/change-email/verify [post]
func VerifyEmailChange(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
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

		var req VerifyEmailChangeRequest
		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		// OTP ni tekshirish
		otpMutex.RLock()
		otpData, exists := emailOTPStore[req.NewEmail]
		otpMutex.RUnlock()

		if !exists {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Avval tasdiqlash kodi so'rang",
			})
			return
		}

		// Muddati o'tganmi
		if time.Now().After(otpData.ExpiresAt) {
			otpMutex.Lock()
			delete(emailOTPStore, req.NewEmail)
			otpMutex.Unlock()
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Tasdiqlash kodi muddati o'tgan",
			})
			return
		}

		// Kod to'g'rimi
		if otpData.Code != req.Code {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri tasdiqlash kodi",
			})
			return
		}

		// UserID mos kelishini tekshirish
		if otpData.UserID != userID {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov",
			})
			return
		}

		// Emailni yangilash
		_, err := db.Exec(`UPDATE users SET email = $1, updated_at = NOW() WHERE id = $2`, req.NewEmail, userID)
		if err != nil {
			log.Printf("Email update xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Email yangilashda xatolik",
			})
			return
		}

		// OTP ni o'chirish
		otpMutex.Lock()
		delete(emailOTPStore, req.NewEmail)
		otpMutex.Unlock()

		// Yangilangan profilni qaytarish
		var user models.User
		err = db.QueryRow(`
			SELECT id, full_name, phone, COALESCE(email, ''), COALESCE(avatar_url, ''), created_at, updated_at
			FROM users WHERE id = $1
		`, userID).Scan(&user.ID, &user.FullName, &user.Phone, &user.Email, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)

		if err != nil {
			log.Printf("Profile fetch xatosi: %v", err)
		}

		log.Printf("✅ Email o'zgartirildi: %s", req.NewEmail)

		writeJSON(w, http.StatusOK, ProfileResponse{
			Success: true,
			Message: "Email manzil muvaffaqiyatli o'zgartirildi",
			User:    &user,
		})
	}
}

// ============================================
// BECOME SELLER - Role ni seller ga o'zgartirish
// ============================================

// BecomeSeller godoc
// @Summary      Sotuvchi bo'lish
// @Description  Customer rolini seller ga o'zgartiradi
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  models.AuthResponse
// @Failure      401  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /user/become-seller [post]
func BecomeSeller(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
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

		// Rolni seller ga o'zgartirish
		result, err := db.Exec(`UPDATE users SET role = 'seller', updated_at = NOW() WHERE id = $1`, userID)
		if err != nil {
			log.Printf("Become seller xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Sotuvchi bo'lishda xatolik yuz berdi",
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

		log.Printf("🎉 User became seller: %s", userID)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: "Tabriklaymiz! Siz endi sotuvchisiz.",
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
