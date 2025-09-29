package interfaces

import (
	"context"
	"time"
)

// CacheRepository defines the interface for caching operations
type CacheRepository interface {
	// Key-Value operations
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Bulk operations
	SetMany(ctx context.Context, items map[string]interface{}, ttl time.Duration) error
	GetMany(ctx context.Context, keys []string) (map[string]interface{}, error)
	DeleteMany(ctx context.Context, keys []string) error

	// Pattern operations
	GetKeysByPattern(ctx context.Context, pattern string) ([]string, error)
	DeleteByPattern(ctx context.Context, pattern string) (int64, error)

	// Expiration
	SetTTL(ctx context.Context, key string, ttl time.Duration) error
	GetTTL(ctx context.Context, key string) (time.Duration, error)

	// Health and stats
	Ping(ctx context.Context) error
	GetStats(ctx context.Context) (map[string]interface{}, error)
}