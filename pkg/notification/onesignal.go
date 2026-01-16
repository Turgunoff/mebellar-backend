package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// OneSignalService - OneSignal push notification xizmati
type OneSignalService struct {
	AppID     string
	RestAPIKey string
	Client    *http.Client
}

// NotificationPayload - OneSignal notification payload
type NotificationPayload struct {
	AppID            string                 `json:"app_id"`
	IncludePlayerIDs []string               `json:"include_player_ids,omitempty"`
	Headings         map[string]string      `json:"headings,omitempty"`
	Contents         map[string]string      `json:"contents,omitempty"`
	Data             map[string]interface{} `json:"data,omitempty"`
	Sound            string                 `json:"sound,omitempty"`
	Priority         int                    `json:"priority,omitempty"`
}

// NewOneSignalService - OneSignal servisini yaratish
func NewOneSignalService() *OneSignalService {
	appID := os.Getenv("ONESIGNAL_APP_ID")
	restAPIKey := os.Getenv("ONESIGNAL_REST_API_KEY")

	if appID == "" {
		appID = "a81db172-c5b3-4616-a014-42f8a05d8ca3" // Default fallback
	}

	if restAPIKey == "" {
		restAPIKey = "os_v2_app_vao3c4wfwndbniauil4kaxmmuntr3rust62ed4mtyjhj373tdyyi6lajvjvi5qt2aptzffic3tqx4vuavs4mft732ymqje3gn22s2ea" // Default fallback
	}

	return &OneSignalService{
		AppID:      appID,
		RestAPIKey: restAPIKey,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendNotification - Push notification yuborish
func (s *OneSignalService) SendNotification(playerID, title, content string, data map[string]interface{}) error {
	if playerID == "" {
		log.Println("⚠️ OneSignal: player_id bo'sh, notification yuborilmaydi")
		return fmt.Errorf("player_id bo'sh")
	}

	payload := NotificationPayload{
		AppID:            s.AppID,
		IncludePlayerIDs: []string{playerID},
		Headings: map[string]string{
			"en": title,
		},
		Contents: map[string]string{
			"en": content,
		},
		Data:     data,
		Sound:    "default",
		Priority: 10,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("❌ OneSignal: JSON marshal xatosi: %v", err)
		return err
	}

	req, err := http.NewRequest("POST", "https://onesignal.com/api/v1/notifications", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("❌ OneSignal: Request yaratishda xatolik: %v", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", s.RestAPIKey))

	resp, err := s.Client.Do(req)
	if err != nil {
		log.Printf("❌ OneSignal: Request yuborishda xatolik: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			log.Printf("❌ OneSignal: API xatosi (status: %d): %v", resp.StatusCode, errorResp)
		} else {
			log.Printf("❌ OneSignal: API xatosi (status: %d)", resp.StatusCode)
		}
		return fmt.Errorf("OneSignal API xatosi: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
		log.Printf("✅ OneSignal: Notification yuborildi (id: %v)", result["id"])
	}

	return nil
}

// SendNotificationAsync - Push notification yuborish (goroutine da)
func (s *OneSignalService) SendNotificationAsync(playerID, title, content string, data map[string]interface{}) {
	go func() {
		if err := s.SendNotification(playerID, title, content, data); err != nil {
			log.Printf("⚠️ OneSignal: Async notification yuborishda xatolik: %v", err)
		}
	}()
}
