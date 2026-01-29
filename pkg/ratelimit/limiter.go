package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

// Limiter интерфейс для rate limiting
type Limiter interface {
	Allow(key string) (bool, error)
	Reset(key string) error
}

// RedisLimiter использует Redis для распределенного rate limiting
type RedisLimiter struct {
	client *redis.Client
	limit  int           // Максимальное количество запросов
	window time.Duration // Временное окно
}

// NewRedisLimiter создает новый Redis-based rate limiter
func NewRedisLimiter(client *redis.Client, limit int, window time.Duration) *RedisLimiter {
	return &RedisLimiter{
		client: client,
		limit:  limit,
		window: window,
	}
}

// Allow проверяет, разрешен ли запрос для данного ключа
func (l *RedisLimiter) Allow(key string) (bool, error) {
	ctx := context.Background()

	// Используем Redis INCR и EXPIRE для sliding window
	redisKey := fmt.Sprintf("ratelimit:%s", key)

	// Получаем текущее количество запросов
	count, err := l.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return false, fmt.Errorf("redis incr error: %w", err)
	}

	// Если это первый запрос, устанавливаем TTL
	if count == 1 {
		if err := l.client.Expire(ctx, redisKey, l.window).Err(); err != nil {
			return false, fmt.Errorf("redis expire error: %w", err)
		}
	}

	// Проверяем, не превышен ли лимит
	if count > int64(l.limit) {
		// Получаем TTL для информации в ошибке
		ttl, _ := l.client.TTL(ctx, redisKey).Result()
		return false, fmt.Errorf("rate limit exceeded, retry after %v", ttl)
	}

	return true, nil
}

// Reset сбрасывает счетчик для ключа
func (l *RedisLimiter) Reset(key string) error {
	ctx := context.Background()
	redisKey := fmt.Sprintf("ratelimit:%s", key)
	return l.client.Del(ctx, redisKey).Err()
}

// MemoryLimiter использует in-memory rate limiting (для single instance)
type MemoryLimiter struct {
	limiters map[string]*rate.Limiter
	limit    rate.Limit
	burst    int
}

// NewMemoryLimiter создает in-memory rate limiter
func NewMemoryLimiter(rps int, burst int) *MemoryLimiter {
	return &MemoryLimiter{
		limiters: make(map[string]*rate.Limiter),
		limit:    rate.Limit(rps),
		burst:    burst,
	}
}

// Allow проверяет, разрешен ли запрос
func (l *MemoryLimiter) Allow(key string) (bool, error) {
	limiter, exists := l.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(l.limit, l.burst)
		l.limiters[key] = limiter
	}

	if !limiter.Allow() {
		return false, fmt.Errorf("rate limit exceeded")
	}

	return true, nil
}

// Reset сбрасывает limiter для ключа
func (l *MemoryLimiter) Reset(key string) error {
	delete(l.limiters, key)
	return nil
}

// Cleanup удаляет старые limiters (вызывать периодически)
func (l *MemoryLimiter) Cleanup() {
	// Простая реализация - очистить все
	// В production нужно отслеживать last access time
	l.limiters = make(map[string]*rate.Limiter)
}
