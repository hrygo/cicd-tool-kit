// Package runner provides caching for review results
package runner

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/ai"
)

const (
	// DefaultCacheTTLHours is the default cache TTL in hours
	DefaultCacheTTLHours = 24

	// CacheFilePermissions is the file permissions for cache files
	CacheFilePermissions = 0600
)

// Cache provides caching for review results
type Cache struct {
	dir      string
	enabled  bool
	mu       sync.RWMutex
	ttl      time.Duration
}

// CachedReview represents a cached review result
type CachedReview struct {
	Summary  ReviewSummary
	Issues   []ai.Issue
	Comment  string
	CachedAt time.Time
	Duration time.Duration // Original execution duration
}

// NewCache creates a new cache instance
func NewCache(dir string, enabled bool) (*Cache, error) {
	if enabled {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create cache directory: %w", err)
		}
	}

	return &Cache{
		dir:     dir,
		enabled: enabled,
		ttl:     time.Duration(DefaultCacheTTLHours) * time.Hour,
	}, nil
}

// GetReview retrieves a cached review
func (c *Cache) GetReview(prID int) (CachedReview, bool) {
	if !c.enabled {
		return CachedReview{}, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	path := c.reviewPath(prID)

	// Stat first to avoid reading deleted files
	// This check is kept under lock to prevent race with cache invalidation
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return CachedReview{}, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return CachedReview{}, false
	}

	var cached CachedReview
	if err := json.Unmarshal(data, &cached); err != nil {
		return CachedReview{}, false
	}

	// Check TTL under lock to prevent race with cache invalidation
	if time.Since(cached.CachedAt) > c.ttl {
		// Expired - remove atomically
		_ = os.Remove(path)
		return CachedReview{}, false
	}

	return cached, true
}

// SetReview stores a review in cache
func (c *Cache) SetReview(prID int, review CachedReview) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	review.CachedAt = time.Now()

	data, err := json.Marshal(review)
	if err != nil {
		log.Printf("Warning: failed to marshal review data for PR %d: %v", prID, err)
		return
	}

	path := c.reviewPath(prID)
	// Use CacheFilePermissions - cache files may contain sensitive code snippets
	if err := os.WriteFile(path, data, CacheFilePermissions); err != nil {
		log.Printf("Warning: failed to write cache file %s: %v", path, err)
	}
}

// Invalidate removes a cached review
func (c *Cache) Invalidate(prID int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	path := c.reviewPath(prID)
	_ = os.Remove(path)
}

// Clear clears all cached reviews
func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.enabled {
		return nil
	}

	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(c.dir, entry.Name())
		if err := os.Remove(path); err != nil {
			// Log but continue - cache cleanup is not critical
			log.Printf("Warning: failed to delete cache file %s: %v", path, err)
		}
	}

	return nil
}

// reviewPath returns the cache file path for a PR.
// The filename includes both the PR ID and a hash suffix to:
// 1. Prevent key collisions (the PR ID prefix ensures uniqueness)
// 2. Provide human-readable filenames for debugging
// 3. Maintain a fixed, predictable filename structure
// MD5 is used for filename generation only, not for security.
func (c *Cache) reviewPath(prID int) string {
	key := fmt.Sprintf("pr-%d", prID)
	hash := md5.Sum([]byte(key))
	// Use "pr-{id}-{hash}.json" format to prevent collisions
	// The PR ID prefix makes each filename unique per PR
	return filepath.Join(c.dir, fmt.Sprintf("pr-%d-%x.json", prID, hash))
}

// GetDiffHash returns a hash of the diff content for caching
// MD5 is used for cache key generation only, not for security purposes.
func GetDiffHash(diff string) string {
	hash := md5.Sum([]byte(diff))
	return fmt.Sprintf("%x", hash)
}

// SetTTL sets the cache TTL
func (c *Cache) SetTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ttl = ttl
}
