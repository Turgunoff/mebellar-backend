package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"mebellar-backend/pkg/logger"

	"go.uber.org/zap"
)

// Cache интерфейс для кэширования
type Cache interface {
	Get(key string, dest interface{}) error
	Set(key string, value interface{}, ttl time.Duration) error
	Delete(key string) error
	Clear() error
}

// RedisCache реализация Cache с Redis
type RedisCache struct {
	client *redis.Client
	prefix string
}

// NewRedisCache создает новый Redis cache
func NewRedisCache(client *redis.Client, prefix string) *RedisCache {
	return &RedisCache{
		client: client,
		prefix: prefix,
	}
}

// Get получает значение из кэша
func (c *RedisCache) Get(key string, dest interface{}) error {
	ctx := context.Background()
	fullKey := c.prefix + key

	data, err := c.client.Get(ctx, fullKey).Bytes()
	if err == redis.Nil {
		return fmt.Errorf("cache miss")
	}
	if err != nil {
		logger.Error("Redis get error",
			zap.String("key", fullKey),
			zap.Error(err),
		)
		return err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		logger.Error("Cache unmarshal error",
			zap.String("key", fullKey),
			zap.Error(err),
		)
		return err
	}

	logger.Debug("Cache hit",
		zap.String("key", fullKey),
	)

	return nil
}

// Set сохраняет значение в кэш
func (c *RedisCache) Set(key string, value interface{}, ttl time.Duration) error {
	ctx := context.Background()
	fullKey := c.prefix + key

	data, err := json.Marshal(value)
	if err != nil {
		logger.Error("Cache marshal error",
			zap.String("key", fullKey),
			zap.Error(err),
		)
		return err
	}

	if err := c.client.Set(ctx, fullKey, data, ttl).Err(); err != nil {
		logger.Error("Redis set error",
			zap.String("key", fullKey),
			zap.Error(err),
		)
		return err
	}

	logger.Debug("Cache set",
		zap.String("key", fullKey),
		zap.Duration("ttl", ttl),
	)

	return nil
}

// Delete удаляет значение из кэша
func (c *RedisCache) Delete(key string) error {
	ctx := context.Background()
	fullKey := c.prefix + key

	return c.client.Del(ctx, fullKey).Err()
}

// Clear очищает весь кэш с префиксом
func (c *RedisCache) Clear() error {
	ctx := context.Background()

	iter := c.client.Scan(ctx, 0, c.prefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			logger.Error("Failed to delete cache key",
				zap.String("key", iter.Val()),
				zap.Error(err),
			)
		}
	}

	return iter.Err()
}

// MemoryCache простой in-memory cache (для development)
type MemoryCache struct {
	data map[string]cacheEntry
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// NewMemoryCache создает in-memory cache
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		data: make(map[string]cacheEntry),
	}

	// Запускаем очистку устаревших записей
	go cache.cleanupExpired()

	return cache
}

func (c *MemoryCache) Get(key string, dest interface{}) error {
	entry, exists := c.data[key]
	if !exists {
		return fmt.Errorf("cache miss")
	}

	if time.Now().After(entry.expiresAt) {
		delete(c.data, key)
		return fmt.Errorf("cache expired")
	}

	return json.Unmarshal(entry.value, dest)
}

func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	c.data[key] = cacheEntry{
		value:     data,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

func (c *MemoryCache) Delete(key string) error {
	delete(c.data, key)
	return nil
}

func (c *MemoryCache) Clear() error {
	c.data = make(map[string]cacheEntry)
	return nil
}

func (c *MemoryCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		for key, entry := range c.data {
			if now.After(entry.expiresAt) {
				delete(c.data, key)
			}
		}
	}
}
