package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/quantfidential/trading-ecosystem/audit-data-adapter-go/pkg/adapters"
)

// CacheRedisRepository implements CacheRepository using Redis
type CacheRedisRepository struct {
	client *redis.Client
	logger *logrus.Logger
	config *adapters.RepositoryConfig
}

// NewCacheRepository creates a new Redis-based cache repository
func NewCacheRepository(client *redis.Client, logger *logrus.Logger, config *adapters.RepositoryConfig) adapters.CacheRepository {
	return &CacheRedisRepository{
		client: client,
		logger: logger,
		config: config,
	}
}

// Set stores a value in Redis cache with TTL
func (r *CacheRedisRepository) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// Serialize value to JSON
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize cache value: %w", err)
	}

	// Use default TTL if not specified
	if ttl == 0 {
		ttl = r.config.DefaultTTL
	}

	err = r.client.Set(ctx, r.getCacheKey(key), data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache value: %w", err)
	}

	return nil
}

// Get retrieves a value from Redis cache
func (r *CacheRedisRepository) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := r.client.Get(ctx, r.getCacheKey(key)).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("cache key not found: %s", key)
		}
		return fmt.Errorf("failed to get cache value: %w", err)
	}

	// Deserialize from JSON
	err = json.Unmarshal([]byte(data), dest)
	if err != nil {
		return fmt.Errorf("failed to deserialize cache value: %w", err)
	}

	return nil
}

// Delete removes a key from Redis cache
func (r *CacheRedisRepository) Delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, r.getCacheKey(key)).Err()
	if err != nil {
		return fmt.Errorf("failed to delete cache key: %w", err)
	}

	return nil
}

// Exists checks if a key exists in Redis cache
func (r *CacheRedisRepository) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, r.getCacheKey(key)).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check cache key existence: %w", err)
	}

	return count > 0, nil
}

// SetMany stores multiple values in Redis cache
func (r *CacheRedisRepository) SetMany(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	// Use default TTL if not specified
	if ttl == 0 {
		ttl = r.config.DefaultTTL
	}

	// Use pipeline for better performance
	pipe := r.client.Pipeline()

	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to serialize cache value for key %s: %w", key, err)
		}

		pipe.Set(ctx, r.getCacheKey(key), data, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute bulk cache set: %w", err)
	}

	return nil
}

// GetMany retrieves multiple values from Redis cache
func (r *CacheRedisRepository) GetMany(ctx context.Context, keys []string) (map[string]interface{}, error) {
	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}

	// Prepare cache keys
	cacheKeys := make([]string, len(keys))
	for i, key := range keys {
		cacheKeys[i] = r.getCacheKey(key)
	}

	// Use pipeline for better performance
	pipe := r.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(cacheKeys))

	for i, cacheKey := range cacheKeys {
		cmds[i] = pipe.Get(ctx, cacheKey)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to execute bulk cache get: %w", err)
	}

	// Process results
	results := make(map[string]interface{})

	for i, cmd := range cmds {
		data, err := cmd.Result()
		if err != nil {
			if err == redis.Nil {
				// Key not found, skip
				continue
			}
			r.logger.WithError(err).WithField("key", keys[i]).Warn("Failed to get cache value")
			continue
		}

		var value interface{}
		if err := json.Unmarshal([]byte(data), &value); err != nil {
			r.logger.WithError(err).WithField("key", keys[i]).Warn("Failed to deserialize cache value")
			continue
		}

		results[keys[i]] = value
	}

	return results, nil
}

// DeleteMany removes multiple keys from Redis cache
func (r *CacheRedisRepository) DeleteMany(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	// Prepare cache keys
	cacheKeys := make([]string, len(keys))
	for i, key := range keys {
		cacheKeys[i] = r.getCacheKey(key)
	}

	err := r.client.Del(ctx, cacheKeys...).Err()
	if err != nil {
		return fmt.Errorf("failed to delete multiple cache keys: %w", err)
	}

	return nil
}

// GetKeysByPattern retrieves all keys matching a pattern
func (r *CacheRedisRepository) GetKeysByPattern(ctx context.Context, pattern string) ([]string, error) {
	cachePattern := r.getCacheKey(pattern)
	keys, err := r.client.Keys(ctx, cachePattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys by pattern: %w", err)
	}

	// Remove cache prefix from keys
	cachePrefix := "cache:"
	results := make([]string, len(keys))
	for i, key := range keys {
		if len(key) > len(cachePrefix) {
			results[i] = key[len(cachePrefix):]
		} else {
			results[i] = key
		}
	}

	return results, nil
}

// DeleteByPattern removes all keys matching a pattern
func (r *CacheRedisRepository) DeleteByPattern(ctx context.Context, pattern string) (int64, error) {
	keys, err := r.GetKeysByPattern(ctx, pattern)
	if err != nil {
		return 0, err
	}

	if len(keys) == 0 {
		return 0, nil
	}

	// Prepare cache keys
	cacheKeys := make([]string, len(keys))
	for i, key := range keys {
		cacheKeys[i] = r.getCacheKey(key)
	}

	deletedCount, err := r.client.Del(ctx, cacheKeys...).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to delete keys by pattern: %w", err)
	}

	return deletedCount, nil
}

// SetTTL sets the TTL for an existing key
func (r *CacheRedisRepository) SetTTL(ctx context.Context, key string, ttl time.Duration) error {
	success, err := r.client.Expire(ctx, r.getCacheKey(key), ttl).Result()
	if err != nil {
		return fmt.Errorf("failed to set TTL: %w", err)
	}

	if !success {
		return fmt.Errorf("key not found: %s", key)
	}

	return nil
}

// GetTTL retrieves the TTL for a key
func (r *CacheRedisRepository) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := r.client.TTL(ctx, r.getCacheKey(key)).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL: %w", err)
	}

	return ttl, nil
}

// Ping checks Redis connection health
func (r *CacheRedisRepository) Ping(ctx context.Context) error {
	_, err := r.client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	return nil
}

// GetStats returns Redis statistics
func (r *CacheRedisRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := r.client.Info(ctx, "memory", "stats", "keyspace").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	stats := map[string]interface{}{
		"redis_info": info,
	}

	// Get additional stats
	dbSize, err := r.client.DBSize(ctx).Result()
	if err == nil {
		stats["db_size"] = dbSize
	}

	// Get memory usage
	memUsage, err := r.client.MemoryUsage(ctx, "cache:*").Result()
	if err == nil {
		stats["cache_memory_usage"] = memUsage
	}

	return stats, nil
}

// Helper methods
func (r *CacheRedisRepository) getCacheKey(key string) string {
	return fmt.Sprintf("cache:%s", key)
}