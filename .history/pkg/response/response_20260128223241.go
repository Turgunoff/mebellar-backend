package response

import (
	"encoding/json"
	"net/http"

	"mebellar-backend/pkg/apperror"
	"mebellar-backend/pkg/logger"

	"go.uber.org/zap"
)

// Response стандартный формат ответа API
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo информация об ошибке для клиента
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Success отправляет успешный ответ
func Success(w http.ResponseWriter, data interface{}, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Created отправляет ответ о создании ресурса (201)
func Created(w http.ResponseWriter, data interface{}, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Error отправляет ответ об ошибке
func Error(w http.ResponseWriter, r *http.Request, err error) {
	appErr := apperror.FromError(err)

	// Логируем внутреннюю ошибку
	if appErr.Internal != nil {
		logger.Error("Request error",
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method),
			zap.String("error_code", appErr.Code),
			zap.Error(appErr.Internal),
			zap.String("user_agent", r.UserAgent()),
			zap.String("remote_addr", r.RemoteAddr),
		)
	} else {
		logger.Warn("Client error",
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method),
			zap.String("error_code", appErr.Code),
			zap.String("message", appErr.Message),
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.HTTPStatus)

	json.NewEncoder(w).Encode(Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
	})
}

// NoContent отправляет пустой успешный ответ (204)
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
