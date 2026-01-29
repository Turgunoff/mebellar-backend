package apperror

import (
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AppError представляет ошибку приложения
type AppError struct {
	Code       string     `json:"code"`        // Код ошибки для клиента
	Message    string     `json:"message"`     // Сообщение для пользователя
	HTTPStatus int        `json:"-"`           // HTTP статус код
	GRPCCode   codes.Code `json:"-"`           // gRPC статус код
	Internal   error      `json:"-"`           // Внутренняя ошибка (не показывается клиенту)
}

func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// ToGRPCError конвертирует в gRPC ошибку
func (e *AppError) ToGRPCError() error {
	return status.Error(e.GRPCCode, e.Message)
}

// NewNotFoundError - ресурс не найден
func NewNotFoundError(message string) *AppError {
	return &AppError{
		Code:       "NOT_FOUND",
		Message:    message,
		HTTPStatus: http.StatusNotFound,
		GRPCCode:   codes.NotFound,
	}
}

// NewValidationError - ошибка валидации входных данных
func NewValidationError(message string) *AppError {
	return &AppError{
		Code:       "VALIDATION_ERROR",
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
		GRPCCode:   codes.InvalidArgument,
	}
}

// NewUnauthorizedError - не авторизован
func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Code:       "UNAUTHORIZED",
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
		GRPCCode:   codes.Unauthenticated,
	}
}

// NewForbiddenError - доступ запрещен
func NewForbiddenError(message string) *AppError {
	return &AppError{
		Code:       "FORBIDDEN",
		Message:    message,
		HTTPStatus: http.StatusForbidden,
		GRPCCode:   codes.PermissionDenied,
	}
}

// NewInternalError - внутренняя ошибка сервера
func NewInternalError(message string, internal error) *AppError {
	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		GRPCCode:   codes.Internal,
		Internal:   internal,
	}
}

// NewConflictError - конфликт (например, дубликат)
func NewConflictError(message string) *AppError {
	return &AppError{
		Code:       "CONFLICT",
		Message:    message,
		HTTPStatus: http.StatusConflict,
		GRPCCode:   codes.AlreadyExists,
	}
}

// NewDatabaseError - ошибка базы данных
func NewDatabaseError(operation string, internal error) *AppError {
	return &AppError{
		Code:       "DATABASE_ERROR",
		Message:    fmt.Sprintf("Ошибка базы данных при выполнении операции: %s", operation),
		HTTPStatus: http.StatusInternalServerError,
		GRPCCode:   codes.Internal,
		Internal:   internal,
	}
}

// NewRateLimitError - превышен лимит запросов
func NewRateLimitError(message string) *AppError {
	return &AppError{
		Code:       "RATE_LIMIT_EXCEEDED",
		Message:    message,
		HTTPStatus: http.StatusTooManyRequests,
		GRPCCode:   codes.ResourceExhausted,
	}
}

// IsAppError проверяет, является ли ошибка AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// FromError пытается преобразовать error в AppError
func FromError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	// Если не AppError, создаем внутреннюю ошибку
	return NewInternalError("Внутренняя ошибка сервера", err)
}
