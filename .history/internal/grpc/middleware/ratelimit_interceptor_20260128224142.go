package middleware

import (
	"context"
	"fmt"

	"mebellar-backend/pkg/logger"
	"mebellar-backend/pkg/ratelimit"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// RateLimitInterceptor создает unary interceptor для rate limiting
func RateLimitInterceptor(limiter ratelimit.Limiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Получаем идентификатор клиента (IP или user_id)
		key := getClientIdentifier(ctx, info.FullMethod)

		// Проверяем rate limit
		allowed, err := limiter.Allow(key)
		if err != nil || !allowed {
			logger.Warn("Rate limit exceeded",
				zap.String("method", info.FullMethod),
				zap.String("client_key", key),
				zap.Error(err),
			)

			return nil, status.Error(codes.ResourceExhausted, "Too many requests. Please try again later.")
		}

		// Продолжаем обработку
		return handler(ctx, req)
	}
}

// getClientIdentifier извлекает идентификатор клиента из контекста
func getClientIdentifier(ctx context.Context, method string) string {
	// Попытка получить user_id из auth context
	if authCtx := GetAuthContext(ctx); authCtx != nil && authCtx.UserID != "" {
		return "user:" + authCtx.UserID
	}

	// Получаем IP из metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if xff := md.Get("x-forwarded-for"); len(xff) > 0 {
			return "ip:" + xff[0]
		}
		if realIP := md.Get("x-real-ip"); len(realIP) > 0 {
			return "ip:" + realIP[0]
		}
	}

	// Fallback - используем метод как ключ (глобальный лимит)
	return "method:" + method
}

// MethodSpecificRateLimits конфигурация лимитов для разных методов
var MethodSpecificRateLimits = map[string]int{
	"/auth.AuthService/Login":    5,  // 5 попыток входа в минуту
	"/auth.AuthService/SendOTP":  3,  // 3 OTP в минуту
	"/auth.AuthService/Register": 3,  // 3 регистрации в минуту
	"default":                    60, // 60 запросов в минуту для остального
}

// AdaptiveRateLimitInterceptor с разными лимитами для разных методов
func AdaptiveRateLimitInterceptor(limiters map[string]ratelimit.Limiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Определяем лимитер для метода
		limiter, exists := limiters[info.FullMethod]
		if !exists {
			limiter = limiters["default"]
		}

		// Если лимитер не найден, пропускаем проверку
		if limiter == nil {
			return handler(ctx, req)
		}

		key := getClientIdentifier(ctx, info.FullMethod)

		allowed, err := limiter.Allow(key)
		if err != nil || !allowed {
			logger.Warn("Rate limit exceeded",
				zap.String("method", info.FullMethod),
				zap.String("client_key", key),
			)

			// Получаем информацию о времени ожидания из ошибки
			retryMessage := "Too many requests. Please slow down and try again later."
			if err != nil {
				retryMessage = fmt.Sprintf("Rate limit exceeded: %v", err)
			}

			return nil, status.Error(codes.ResourceExhausted, retryMessage)
		}

		return handler(ctx, req)
	}
}
