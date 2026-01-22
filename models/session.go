package models

import "time"

// UserSession - foydalanuvchi sessiyasi modeli (user_sessions jadvali bilan bir xil)
type UserSession struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	DeviceName string     `json:"device_name"`
	DeviceID   string     `json:"device_id"`
	IPAddress  string     `json:"ip_address,omitempty"`
	AppType    string     `json:"app_type"`              // client, seller, admin
	DeviceOS   string     `json:"device_os,omitempty"`   // iOS, Android
	OSVersion  string     `json:"os_version,omitempty"`  // 17.2, 14.0
	AppVersion string     `json:"app_version,omitempty"` // 1.0.0, 1.0.0+12
	IsTrusted  bool       `json:"is_trusted"`            // Qurilma ishonchlimi (2FA uchun)
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`  // Sessiya tugash vaqti
	LastActive time.Time  `json:"last_active"`
	IsCurrent  bool       `json:"is_current"`
	CreatedAt  time.Time  `json:"created_at"`
}

// SessionsResponse - sessiyalar ro'yxati javobi
type SessionsResponse struct {
	Success  bool          `json:"success"`
	Message  string        `json:"message,omitempty"`
	Sessions []UserSession `json:"sessions,omitempty"`
}

// RevokeSessionResponse - sessiyani bekor qilish javobi
type RevokeSessionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// SetPinRequest - PIN o'rnatish so'rovi
type SetPinRequest struct {
	HasPin bool `json:"has_pin"`
}

// SetPinResponse - PIN o'rnatish javobi
type SetPinResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
