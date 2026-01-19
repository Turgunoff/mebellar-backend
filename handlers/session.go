package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"mebellar-backend/models"
	"net/http"
	"strings"
)

// SessionsHandler godoc
// @Summary      Faol sessiyalar ro'yxati
// @Description  Joriy foydalanuvchining barcha faol sessiyalarini qaytaradi
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  models.SessionsResponse
// @Failure      401  {object}  models.SessionsResponse
// @Failure      500  {object}  models.SessionsResponse
// @Router       /auth/sessions [get]
func SessionsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.SessionsResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Context'dan user_id ni olish (JWTMiddleware orqali)
		userID := r.Context().Value("user_id")
		if userID == nil {
			writeJSON(w, http.StatusUnauthorized, models.SessionsResponse{
				Success: false,
				Message: "Avtorizatsiya talab qilinadi",
			})
			return
		}

		// Sessiyalarni olish
		rows, err := db.Query(`
			SELECT id, user_id, device_name, device_id, COALESCE(ip_address, ''), 
			       last_active, is_current, created_at
			FROM user_sessions 
			WHERE user_id = $1 
			ORDER BY last_active DESC
		`, userID)
		if err != nil {
			log.Println("Sessions query xatosi:", err)
			writeJSON(w, http.StatusInternalServerError, models.SessionsResponse{
				Success: false,
				Message: "Server xatosi",
			})
			return
		}
		defer rows.Close()

		var sessions []models.UserSession
		for rows.Next() {
			var session models.UserSession
			if err := rows.Scan(
				&session.ID,
				&session.UserID,
				&session.DeviceName,
				&session.DeviceID,
				&session.IPAddress,
				&session.LastActive,
				&session.IsCurrent,
				&session.CreatedAt,
			); err != nil {
				log.Println("Session scan xatosi:", err)
				continue
			}
			sessions = append(sessions, session)
		}

		writeJSON(w, http.StatusOK, models.SessionsResponse{
			Success:  true,
			Sessions: sessions,
		})
	}
}

// RevokeSessionHandler godoc
// @Summary      Sessiyani bekor qilish
// @Description  Ma'lum bir sessiyani o'chiradi (boshqa qurilmadan chiqarish)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Session ID"
// @Success      200  {object}  models.RevokeSessionResponse
// @Failure      401  {object}  models.RevokeSessionResponse
// @Failure      403  {object}  models.RevokeSessionResponse
// @Failure      404  {object}  models.RevokeSessionResponse
// @Failure      500  {object}  models.RevokeSessionResponse
// @Router       /auth/sessions/{id} [delete]
func RevokeSessionHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeJSON(w, http.StatusMethodNotAllowed, models.RevokeSessionResponse{
				Success: false,
				Message: "Faqat DELETE metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Context'dan user_id ni olish
		userID := r.Context().Value("user_id")
		if userID == nil {
			writeJSON(w, http.StatusUnauthorized, models.RevokeSessionResponse{
				Success: false,
				Message: "Avtorizatsiya talab qilinadi",
			})
			return
		}

		// URL dan session ID ni olish
		path := strings.TrimPrefix(r.URL.Path, "/api/auth/sessions/")
		sessionID := strings.TrimSuffix(path, "/")

		if sessionID == "" {
			writeJSON(w, http.StatusBadRequest, models.RevokeSessionResponse{
				Success: false,
				Message: "Session ID kiritilmagan",
			})
			return
		}

		// "all" bo'lsa - barcha boshqa sessiyalarni o'chirish
		if sessionID == "all" {
			// Joriy sessiyadan tashqari barchasini o'chirish
			result, err := db.Exec(`
				DELETE FROM user_sessions 
				WHERE user_id = $1 AND is_current = false
			`, userID)
			if err != nil {
				log.Println("Delete all sessions xatosi:", err)
				writeJSON(w, http.StatusInternalServerError, models.RevokeSessionResponse{
					Success: false,
					Message: "Sessiyalarni o'chirishda xatolik",
				})
				return
			}

			rowsAffected, _ := result.RowsAffected()
			log.Printf("✅ Deleted %d sessions for user %v", rowsAffected, userID)

			writeJSON(w, http.StatusOK, models.RevokeSessionResponse{
				Success: true,
				Message: "Barcha boshqa qurilmalardan chiqildi",
			})
			return
		}

		// Sessiyani tekshirish - foydalanuvchiga tegishlimi
		var sessionUserID string
		err := db.QueryRow(`
			SELECT user_id FROM user_sessions WHERE id = $1
		`, sessionID).Scan(&sessionUserID)

		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.RevokeSessionResponse{
				Success: false,
				Message: "Sessiya topilmadi",
			})
			return
		}

		if err != nil {
			log.Println("Session check xatosi:", err)
			writeJSON(w, http.StatusInternalServerError, models.RevokeSessionResponse{
				Success: false,
				Message: "Server xatosi",
			})
			return
		}

		// Foydalanuvchiga tegishli emasligini tekshirish
		if sessionUserID != userID.(string) {
			writeJSON(w, http.StatusForbidden, models.RevokeSessionResponse{
				Success: false,
				Message: "Bu sessiya sizga tegishli emas",
			})
			return
		}

		// Sessiyani o'chirish
		_, err = db.Exec(`DELETE FROM user_sessions WHERE id = $1`, sessionID)
		if err != nil {
			log.Println("Session delete xatosi:", err)
			writeJSON(w, http.StatusInternalServerError, models.RevokeSessionResponse{
				Success: false,
				Message: "Sessiyani o'chirishda xatolik",
			})
			return
		}

		log.Printf("✅ Session %s revoked for user %v", sessionID, userID)

		writeJSON(w, http.StatusOK, models.RevokeSessionResponse{
			Success: true,
			Message: "Sessiya bekor qilindi",
		})
	}
}

// SetPinHandler godoc
// @Summary      PIN holatini o'rnatish
// @Description  Foydalanuvchi PIN kod o'rnatganligini belgilaydi
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body models.SetPinRequest true "PIN holati"
// @Success      200  {object}  models.SetPinResponse
// @Failure      400  {object}  models.SetPinResponse
// @Failure      401  {object}  models.SetPinResponse
// @Failure      500  {object}  models.SetPinResponse
// @Router       /auth/set-pin [post]
func SetPinHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.SetPinResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Context'dan user_id ni olish
		userID := r.Context().Value("user_id")
		if userID == nil {
			writeJSON(w, http.StatusUnauthorized, models.SetPinResponse{
				Success: false,
				Message: "Avtorizatsiya talab qilinadi",
			})
			return
		}

		var req models.SetPinRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, models.SetPinResponse{
				Success: false,
				Message: "Noto'g'ri so'rov formati",
			})
			return
		}

		// has_pin ni yangilash
		_, err := db.Exec(`
			UPDATE users SET has_pin = $1, updated_at = NOW() WHERE id = $2
		`, req.HasPin, userID)

		if err != nil {
			log.Println("Set PIN xatosi:", err)
			writeJSON(w, http.StatusInternalServerError, models.SetPinResponse{
				Success: false,
				Message: "PIN holatini saqlashda xatolik",
			})
			return
		}

		message := "PIN kod o'rnatildi"
		if !req.HasPin {
			message = "PIN kod o'chirildi"
		}

		log.Printf("✅ PIN status updated for user %v: has_pin=%v", userID, req.HasPin)

		writeJSON(w, http.StatusOK, models.SetPinResponse{
			Success: true,
			Message: message,
		})
	}
}
