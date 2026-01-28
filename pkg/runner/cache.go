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

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/claude"
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
	Issues   []claude.Issue
	Comment  string
	CachedAt time.Time
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
		ttl:     24 * time.Hour,
	}, nil
}

// GetReview retrieves a cached review
func (c *Cache) GetReview(prID int) (CachedReview, bool) {
	if !c.enabled {
		return CachedReview{}, false
	}

	path := c.reviewPath(prID)

	data, err := os.ReadFile(path)
	if err != nil {
		return CachedReview{}, false
	}

	var cached CachedReview
	if err := json.Unmarshal(data, &cached); err != nil {
		return CachedReview{}, false
	}

	// Check TTL - must hold lock for consistent ttl read
	c.mu.RLock()
	ttl := c.ttl
	c.mu.RUnlock()

	if time.Since(cached.CachedAt) > ttl {
		// Remove expired file - use write lock for mutation
		c.mu.Lock()
		_ = os.Remove(path)
		c.mu.Unlock()
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
	// Use 0600 permissions - cache files may contain sensitive code snippets
	if err := os.WriteFile(path, data, 0600); err != nil {
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

// reviewPath returns the cache file path for a PR
// MD5 is used for filename generation only, not for security.
// The hash provides consistent short filenames from cache keys.
func (c *Cache) reviewPath(prID int) string {
	key := fmt.Sprintf("pr-%d", prID)
	hash := md5.Sum([]byte(key))
	return filepath.Join(c.dir, fmt.Sprintf("%x.json", hash))
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
