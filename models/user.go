package models

import "time"

// User - foydalanuvchi modeli (users jadvali bilan bir xil)
type User struct {
	ID           string    `json:"id"` // UUID sifatida string
	FullName     string    `json:"full_name"`
	Phone        string    `json:"phone"`
	Email        string    `json:"email,omitempty"`      // COALESCE bilan bo'sh string qaytadi
	AvatarURL    string    `json:"avatar_url,omitempty"` // COALESCE bilan bo'sh string qaytadi
	Role         string    `json:"role"`                 // customer, seller, admin
	PasswordHash string    `json:"-"`                    // JSON'da ko'rsatilmasin
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SendOTPRequest - OTP yuborish uchun so'rov
type SendOTPRequest struct {
	Phone string `json:"phone"`
}

// VerifyOTPRequest - OTP tekshirish uchun so'rov
type VerifyOTPRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

// RegisterRequest - ro'yxatdan o'tish uchun so'rov
type RegisterRequest struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	Role     string `json:"role"` // customer, seller - agar bo'sh bo'lsa customer
}

// LoginRequest - kirish uchun so'rov
type LoginRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// ForgotPasswordRequest - parolni unutdim so'rovi
type ForgotPasswordRequest struct {
	Phone string `json:"phone"`
}

// ResetPasswordRequest - parolni tiklash so'rovi
type ResetPasswordRequest struct {
	Phone       string `json:"phone"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

// AuthResponse - autentifikatsiya javobi
type AuthResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
	User    *User  `json:"user,omitempty"`
}
