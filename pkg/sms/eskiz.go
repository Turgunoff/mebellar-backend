package sms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	EskizBaseURL     = "https://notify.eskiz.uz/api"
	EskizSenderID    = "4546"
	TokenExpireHours = 24 // Token yangilanish vaqti
)

// EskizService - Eskiz.uz SMS Gateway xizmati
type EskizService struct {
	email    string
	password string
	token    string
	tokenExp time.Time
	client   *http.Client
	mu       sync.RWMutex
}

// EskizLoginResponse - Login javobi
type EskizLoginResponse struct {
	Message string `json:"message"`
	Data    struct {
		Token string `json:"token"`
	} `json:"data"`
	TokenType string `json:"token_type"`
}

// EskizSMSResponse - SMS yuborish javobi
type EskizSMSResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// NewEskizService - yangi Eskiz servis yaratish
func NewEskizService(email, password string) *EskizService {
	return &EskizService{
		email:    email,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Login - Eskiz.uz ga kirish va token olish
func (e *EskizService) Login() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	log.Println("📱 [ESKIZ] Logging in to Eskiz.uz...")

	// Form data tayyorlash
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("email", e.email)
	writer.WriteField("password", e.password)
	writer.Close()

	// Request yaratish
	req, err := http.NewRequest("POST", EskizBaseURL+"/auth/login", body)
	if err != nil {
		return fmt.Errorf("request yaratishda xatolik: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// So'rov yuborish
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("login so'rovida xatolik: %v", err)
	}
	defer resp.Body.Close()

	// Javobni o'qish
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("javobni o'qishda xatolik: %v", err)
	}

	log.Printf("📱 [ESKIZ] Login response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login muvaffaqiyatsiz: %s", string(respBody))
	}

	// JSON parse
	var loginResp EskizLoginResponse
	if err := json.Unmarshal(respBody, &loginResp); err != nil {
		return fmt.Errorf("JSON parse xatosi: %v", err)
	}

	if loginResp.Data.Token == "" {
		return fmt.Errorf("token olinmadi: %s", loginResp.Message)
	}

	e.token = loginResp.Data.Token
	e.tokenExp = time.Now().Add(TokenExpireHours * time.Hour)

	log.Println("✅ [ESKIZ] Successfully logged in!")
	return nil
}

// isTokenValid - token hali yaroqlimi
func (e *EskizService) isTokenValid() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.token != "" && time.Now().Before(e.tokenExp)
}

// getToken - token olish (kerak bo'lsa login qilish)
func (e *EskizService) getToken() (string, error) {
	if e.isTokenValid() {
		e.mu.RLock()
		token := e.token
		e.mu.RUnlock()
		return token, nil
	}

	// Token yo'q yoki eskirgan - login qilish
	if err := e.Login(); err != nil {
		return "", err
	}

	e.mu.RLock()
	token := e.token
	e.mu.RUnlock()
	return token, nil
}

// FormatPhone - telefon raqamini formatlash (998901234567)
func FormatPhone(phone string) string {
	// +, bo'shliq, tire va boshqa belgilarni olib tashlash
	phone = strings.ReplaceAll(phone, "+", "")
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")

	// Agar 9 bilan boshlansa, 998 qo'shish
	if strings.HasPrefix(phone, "9") && len(phone) == 9 {
		phone = "998" + phone
	}

	return phone
}

// SendSMS - SMS yuborish
func (e *EskizService) SendSMS(phone, message string) error {
	// Telefon raqamini formatlash
	formattedPhone := FormatPhone(phone)

	log.Printf("📱 [ESKIZ] Sending SMS to %s...", formattedPhone)

	// Tokenni olish
	token, err := e.getToken()
	if err != nil {
		return fmt.Errorf("token olishda xatolik: %v", err)
	}

	// SMS yuborish
	err = e.sendSMSWithToken(formattedPhone, message, token)
	if err != nil {
		// Agar 401 bo'lsa, qayta login qilib urinish
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "Unauthorized") {
			log.Println("📱 [ESKIZ] Token expired, refreshing...")
			if loginErr := e.Login(); loginErr != nil {
				return fmt.Errorf("token yangilashda xatolik: %v", loginErr)
			}

			e.mu.RLock()
			newToken := e.token
			e.mu.RUnlock()

			return e.sendSMSWithToken(formattedPhone, message, newToken)
		}
		return err
	}

	return nil
}

// sendSMSWithToken - token bilan SMS yuborish
func (e *EskizService) sendSMSWithToken(phone, message, token string) error {
	// Form data tayyorlash
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("mobile_phone", phone)
	writer.WriteField("message", message)
	writer.WriteField("from", EskizSenderID)
	writer.Close()

	// Request yaratish
	req, err := http.NewRequest("POST", EskizBaseURL+"/message/sms/send", body)
	if err != nil {
		return fmt.Errorf("request yaratishda xatolik: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	// So'rov yuborish
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("SMS yuborishda xatolik: %v", err)
	}
	defer resp.Body.Close()

	// Javobni o'qish
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("javobni o'qishda xatolik: %v", err)
	}

	log.Printf("📱 [ESKIZ] SMS response: %d - %s", resp.StatusCode, string(respBody))

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("401 Unauthorized")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SMS yuborish muvaffaqiyatsiz: %s", string(respBody))
	}

	// Response parsing
	var smsResp EskizSMSResponse
	if err := json.Unmarshal(respBody, &smsResp); err != nil {
		// Parse xatosi bo'lsa ham, 200 bo'lsa OK deb hisoblaymiz
		log.Printf("📱 [ESKIZ] SMS sent (parse warning): %v", err)
		return nil
	}

	if smsResp.Status == "error" {
		return fmt.Errorf("SMS xatosi: %s", smsResp.Message)
	}

	log.Printf("✅ [ESKIZ] SMS sent successfully! ID: %s", smsResp.ID)
	return nil
}

// SMSService - SMS xizmatlari uchun interfeys
type SMSService interface {
	SendSMS(phone, message string) error
	SendOTP(phone, code string) error
}

// SendOTP - OTP kod yuborish (formatlangan xabar bilan)
func (e *EskizService) SendOTP(phone, code string) error {
	// Using approved template for testing
	message := fmt.Sprintf("Verification code to log in to the Edumate platform: %s. n8SDK1tHd", code)
	return e.SendSMS(phone, message)
}

// RefreshToken - Tokenni yangilash
func (e *EskizService) RefreshToken() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.token == "" {
		// Token yo'q, login qilish kerak
		e.mu.Unlock()
		err := e.Login()
		e.mu.Lock()
		return err
	}

	log.Println("📱 [ESKIZ] Refreshing token...")

	// Refresh endpoint
	req, err := http.NewRequest("PATCH", EskizBaseURL+"/auth/refresh", nil)
	if err != nil {
		return fmt.Errorf("request yaratishda xatolik: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.token)

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("token yangilashda xatolik: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Refresh ishlamadi, qayta login
		e.mu.Unlock()
		err := e.Login()
		e.mu.Lock()
		return err
	}

	respBody, _ := io.ReadAll(resp.Body)
	var loginResp EskizLoginResponse
	if err := json.Unmarshal(respBody, &loginResp); err == nil && loginResp.Data.Token != "" {
		e.token = loginResp.Data.Token
		e.tokenExp = time.Now().Add(TokenExpireHours * time.Hour)
		log.Println("✅ [ESKIZ] Token refreshed!")
	}

	return nil
}
