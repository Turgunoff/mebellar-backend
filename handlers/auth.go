package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"mebellar-backend/models"
	"net/http"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// OTP storage - telefon raqami -> kod (MVP uchun oddiy global o'zgaruvchi)
var otpStore = make(map[string]string)

// Verified phones - OTP tasdiqlangan telefonlar
var verifiedPhones = make(map[string]bool)

// JWT secret key (production'da environment variable dan oling!)
var jwtSecretKey = []byte("mebellar-super-secret-key-2024")

// generateOTP - 5 xonali tasodifiy kod yaratadi
func generateOTP() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%05d", rand.Intn(100000))
}

// isValidPhone - telefon raqamini tekshiradi (+998 bilan boshlanishi kerak)
func isValidPhone(phone string) bool {
	// +998 bilan boshlanib, 12 ta raqamdan iborat bo'lishi kerak
	pattern := `^\+998[0-9]{9}$`
	matched, _ := regexp.MatchString(pattern, phone)
	return matched
}

// hashPassword - parolni xavfsiz hash qiladi
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// checkPassword - parolni tekshiradi
func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// generateJWT - JWT token yaratadi (role bilan)
func generateJWT(userID string, phone string, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"phone":   phone,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 kun amal qiladi
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecretKey)
}

// writeJSON - JSON javob qaytarish uchun helper
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// SendOTP godoc
// @Summary      OTP kod yuborish
// @Description  Telefon raqamiga 5 xonali tasdiqlash kodi yuboradi (Mock SMS - konsolga chiqadi)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body models.SendOTPRequest true "Telefon raqami"
// @Success      200  {object}  models.AuthResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      405  {object}  models.AuthResponse
// @Router       /auth/send-otp [post]
func SendOTP(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
			})
			return
		}

		var req models.SendOTPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		// Telefon raqamini tekshirish
		if !isValidPhone(req.Phone) {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri telefon raqam formati. +998XXXXXXXXX formatida kiriting",
			})
			return
		}

		// 5 xonali OTP yaratish
		code := generateOTP()

		// OTP'ni saqlash
		otpStore[req.Phone] = code

		// MOCK SMS - konsolga chiqarish
		fmt.Printf("📱 MOCK SMS to %s: %s\n", req.Phone, code)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: "Tasdiqlash kodi yuborildi",
		})
	}
}

// VerifyOTP godoc
// @Summary      OTP kodni tasdiqlash
// @Description  Yuborilgan OTP kodni tekshiradi va telefon raqamini tasdiqlaydi
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body models.VerifyOTPRequest true "Telefon va OTP kod"
// @Success      200  {object}  models.AuthResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      405  {object}  models.AuthResponse
// @Router       /auth/verify-otp [post]
func VerifyOTP(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
			})
			return
		}

		var req models.VerifyOTPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		// OTP tekshirish
		storedCode, exists := otpStore[req.Phone]
		if !exists {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Bu telefon raqamiga kod yuborilmagan",
			})
			return
		}

		if storedCode != req.Code {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri tasdiqlash kodi",
			})
			return
		}

		// OTP tasdiqlandi - telefon raqamini verified qilamiz
		verifiedPhones[req.Phone] = true
		delete(otpStore, req.Phone) // Ishlatilgan OTP'ni o'chirish

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: "Telefon raqami tasdiqlandi",
		})
	}
}

// Register godoc
// @Summary      Ro'yxatdan o'tish
// @Description  Yangi foydalanuvchi yaratadi. Avval telefon raqami OTP orqali tasdiqlanishi kerak.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body models.RegisterRequest true "Ro'yxatdan o'tish ma'lumotlari"
// @Success      201  {object}  models.AuthResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      409  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /auth/register [post]
func Register(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
			})
			return
		}

		var req models.RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		// Validatsiya
		if req.FullName == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Ism kiritilishi shart",
			})
			return
		}

		if !isValidPhone(req.Phone) {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri telefon raqam formati",
			})
			return
		}

		if len(req.Password) < 6 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Parol kamida 6 ta belgidan iborat bo'lishi kerak",
			})
			return
		}

		// Telefon tasdiqlangan yoki yo'qligini tekshirish (ixtiyoriy)
		if !verifiedPhones[req.Phone] {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Avval telefon raqamini tasdiqlang",
			})
			return
		}

		// Telefon raqami mavjudligini tekshirish
		var existingID string
		err := db.QueryRow("SELECT id FROM users WHERE phone = $1", req.Phone).Scan(&existingID)
		if err == nil {
			writeJSON(w, http.StatusConflict, models.AuthResponse{
				Success: false,
				Message: "Bu telefon raqami allaqachon ro'yxatdan o'tgan",
			})
			return
		}

		// Parolni hash qilish
		passwordHash, err := hashPassword(req.Password)
		if err != nil {
			log.Println("Parol hash xatosi:", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Server xatosi",
			})
			return
		}

	// Role ni aniqlash - agar bo'sh bo'lsa "customer"
	role := req.Role
	if role == "" {
		role = "customer"
	}
	// Faqat ruxsat etilgan role'lar
	if role != "customer" && role != "seller" && role != "admin" {
		role = "customer"
	}

	// Foydalanuvchini bazaga qo'shish
	var userID string
	err = db.QueryRow(`
		INSERT INTO users (full_name, phone, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id
	`, req.FullName, req.Phone, passwordHash, role).Scan(&userID)

	if err != nil {
		log.Println("User insert xatosi:", err)
		writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
			Success: false,
			Message: "Foydalanuvchi yaratishda xatolik",
		})
		return
	}

	// Verified holatni o'chirish (faqat bir marta ishlatish uchun)
	delete(verifiedPhones, req.Phone)

	// JWT token yaratish (role bilan)
	token, err := generateJWT(userID, req.Phone, role)
		if err != nil {
			log.Println("JWT xatosi:", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Token yaratishda xatolik",
			})
			return
		}

	writeJSON(w, http.StatusCreated, models.AuthResponse{
		Success: true,
		Message: "Ro'yxatdan o'tish muvaffaqiyatli",
		Token:   token,
		User: &models.User{
			ID:       userID,
			FullName: req.FullName,
			Phone:    req.Phone,
			Role:     role,
		},
	})
	}
}

// Login godoc
// @Summary      Tizimga kirish
// @Description  Telefon raqami va parol orqali tizimga kirish. JWT token qaytaradi.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body models.LoginRequest true "Login ma'lumotlari"
// @Success      200  {object}  models.AuthResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      401  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /auth/login [post]
func Login(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
			})
			return
		}

		var req models.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

	// Foydalanuvchini topish
	var user models.User
	var passwordHash string
	err := db.QueryRow(`
		SELECT id, full_name, phone, COALESCE(role, 'customer'), password_hash, created_at, updated_at
		FROM users WHERE phone = $1
	`, req.Phone).Scan(&user.ID, &user.FullName, &user.Phone, &user.Role, &passwordHash, &user.CreatedAt, &user.UpdatedAt)

		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
				Success: false,
				Message: "Telefon raqami yoki parol noto'g'ri",
			})
			return
		}

		if err != nil {
			log.Println("Login query xatosi:", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Server xatosi",
			})
			return
		}

	// Parolni tekshirish
	if !checkPassword(req.Password, passwordHash) {
		writeJSON(w, http.StatusUnauthorized, models.AuthResponse{
			Success: false,
			Message: "Telefon raqami yoki parol noto'g'ri",
		})
		return
	}

	// JWT token yaratish (role bilan)
	token, err := generateJWT(user.ID, user.Phone, user.Role)
		if err != nil {
			log.Println("JWT xatosi:", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Token yaratishda xatolik",
			})
			return
		}

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: "Kirish muvaffaqiyatli",
			Token:   token,
			User:    &user,
		})
	}
}

// ForgotPassword godoc
// @Summary      Parolni unutdim
// @Description  Telefon raqamiga parolni tiklash uchun OTP kod yuboradi
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body models.ForgotPasswordRequest true "Telefon raqami"
// @Success      200  {object}  models.AuthResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /auth/forgot-password [post]
func ForgotPassword(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
			})
			return
		}

		var req models.ForgotPasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		// Telefon raqamini tekshirish
		if !isValidPhone(req.Phone) {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri telefon raqam formati",
			})
			return
		}

		// Foydalanuvchi mavjudligini tekshirish
		var existingID string
		err := db.QueryRow("SELECT id FROM users WHERE phone = $1", req.Phone).Scan(&existingID)
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Bu telefon raqami ro'yxatdan o'tmagan",
			})
			return
		}

		if err != nil {
			log.Println("Forgot password query xatosi:", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Server xatosi",
			})
			return
		}

		// 5 xonali OTP yaratish
		code := generateOTP()

		// OTP'ni saqlash
		otpStore[req.Phone] = code

		// MOCK SMS - konsolga chiqarish
		fmt.Printf("📱 MOCK SMS (Password Reset) to %s: %s\n", req.Phone, code)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: "Parolni tiklash kodi yuborildi",
		})
	}
}

// ResetPassword godoc
// @Summary      Parolni tiklash
// @Description  OTP kod orqali yangi parol o'rnatadi
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body models.ResetPasswordRequest true "Parolni tiklash ma'lumotlari"
// @Success      200  {object}  models.AuthResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /auth/reset-password [post]
func ResetPassword(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
			})
			return
		}

		var req models.ResetPasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		// Validatsiya
		if !isValidPhone(req.Phone) {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri telefon raqam formati",
			})
			return
		}

		if len(req.NewPassword) < 6 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Yangi parol kamida 6 ta belgidan iborat bo'lishi kerak",
			})
			return
		}

		// OTP tekshirish
		storedCode, exists := otpStore[req.Phone]
		if !exists {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Bu telefon raqamiga kod yuborilmagan",
			})
			return
		}

		if storedCode != req.Code {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri tasdiqlash kodi",
			})
			return
		}

		// Yangi parolni hash qilish
		passwordHash, err := hashPassword(req.NewPassword)
		if err != nil {
			log.Println("Parol hash xatosi:", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Server xatosi",
			})
			return
		}

		// Parolni yangilash
		result, err := db.Exec(`
			UPDATE users SET password_hash = $1, updated_at = NOW()
			WHERE phone = $2
		`, passwordHash, req.Phone)

		if err != nil {
			log.Println("Password update xatosi:", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Parolni yangilashda xatolik",
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

		// OTP'ni o'chirish
		delete(otpStore, req.Phone)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: "Parol muvaffaqiyatli yangilandi",
		})
	}
}
