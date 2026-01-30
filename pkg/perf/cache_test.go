// Package perf tests
package perf

import (
	"strconv"
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	cache := NewCache[string, int](5, 0)

	cache.Set("key1", 100)
	val, ok := cache.Get("key1")

	if !ok {
		t.Fatal("Key not found")
	}

	if val != 100 {
		t.Errorf("Expected 100, got %d", val)
	}
}

func TestCacheTTL(t *testing.T) {
	cache := NewCache[string, int](5, 50*time.Millisecond)

	cache.Set("key1", 100)

	// Should exist immediately
	val, ok := cache.Get("key1")
	if !ok || val != 100 {
		t.Error("Key should exist immediately")
	}

	// Wait for expiration
	time.Sleep(60 * time.Millisecond)

	_, ok = cache.Get("key1")
	if ok {
		t.Error("Key should have expired")
	}
}

func TestCacheLRU(t *testing.T) {
	cache := NewCache[string, int](3, 0)

	cache.Set("key1", 1)
	cache.Set("key2", 2)
	cache.Set("key3", 3)

	// All should exist
	for i := 1; i <= 3; i++ {
		if _, ok := cache.Get("key" + strconv.Itoa(i)); !ok {
			t.Errorf("key%d should exist", i)
		}
	}

	// Add 4th item, should evict key1
	cache.Set("key4", 4)

	if _, ok := cache.Get("key1"); ok {
		t.Error("key1 should have been evicted")
	}

	// Others should still exist
	for i := 2; i <= 4; i++ {
		if _, ok := cache.Get("key" + strconv.Itoa(i)); !ok {
			t.Errorf("key%d should exist", i)
		}
	}
}

func TestCacheDelete(t *testing.T) {
	cache := NewCache[string, int](5, 0)

	cache.Set("key1", 100)
	cache.Delete("key1")

	if _, ok := cache.Get("key1"); ok {
		t.Error("Key should have been deleted")
	}
}

func TestCacheContains(t *testing.T) {
	cache := NewCache[string, int](5, 0)

	if cache.Contains("key1") {
		t.Error("Key should not exist")
	}

	cache.Set("key1", 100)

	if !cache.Contains("key1") {
		t.Error("Key should exist")
	}
}

func TestCacheLen(t *testing.T) {
	cache := NewCache[string, int](5, 0)

	if cache.Len() != 0 {
		t.Errorf("Expected length 0, got %d", cache.Len())
	}

	cache.Set("key1", 100)
	cache.Set("key2", 200)

	if cache.Len() != 2 {
		t.Errorf("Expected length 2, got %d", cache.Len())
	}
}

func TestCacheClear(t *testing.T) {
	cache := NewCache[string, int](5, 0)

	cache.Set("key1", 100)
	cache.Set("key2", 200)

	cache.Clear()

	if cache.Len() != 0 {
		t.Errorf("Expected length 0 after clear, got %d", cache.Len())
	}

	if _, ok := cache.Get("key1"); ok {
		t.Error("Key should not exist after clear")
	}
}

func TestCacheKeys(t *testing.T) {
	cache := NewCache[string, int](5, 0)

	cache.Set("key1", 100)
	cache.Set("key2", 200)
	cache.Set("key3", 300)

	keys := cache.Keys()
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}
}

func TestCacheRange(t *testing.T) {
	cache := NewCache[string, int](5, 0)

	cache.Set("key1", 100)
	cache.Set("key2", 200)
	cache.Set("key3", 300)

	count := 0
	cache.Range(func(key string, value int) bool {
		count++
		return true
	})

	if count != 3 {
		t.Errorf("Expected to range over 3 items, got %d", count)
	}
}

func TestCacheOnEvicted(t *testing.T) {
	evictedKeys := make([]string, 0)
	cache := NewCache[string, int](2, 0)

	cache.SetOnEvicted(func(key string, value int) {
		evictedKeys = append(evictedKeys, key)
	})

	cache.Set("key1", 100)
	cache.Set("key2", 200)
	cache.Set("key3", 300) // Should evict key1

	if len(evictedKeys) != 1 {
		t.Errorf("Expected 1 evicted key, got %d", len(evictedKeys))
	}

	if evictedKeys[0] != "key1" {
		t.Errorf("Expected evicted key 'key1', got '%s'", evictedKeys[0])
	}
}

func TestCacheClose(t *testing.T) {
	cache := NewCache[string, int](5, 50*time.Millisecond)

	cache.Set("key1", 100)
	cache.Close()

	// After close, operations should be no-ops
	cache.Set("key2", 200)
	if _, ok := cache.Get("key1"); ok {
		t.Error("Get should return false after close")
	}

	if cache.Len() != 0 {
		t.Errorf("Expected length 0 after close, got %d", cache.Len())
	}
}

func TestStatsCache(t *testing.T) {
	cache := NewStatsCache[string, int](5, 0)

	// Miss
	cache.Get("key1")

	// Hit
	cache.Set("key1", 100)
	cache.Get("key1")

	stats := cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	hitRate := cache.HitRate()
	if hitRate != 0.5 {
		t.Errorf("Expected hit rate 0.5, got %f", hitRate)
	}
}

func TestStatsCacheResetStats(t *testing.T) {
	cache := NewStatsCache[string, int](5, 0)

	cache.Set("key1", 100)
	cache.Get("key1")
	cache.Get("key2")

	cache.ResetStats()

	stats := cache.Stats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Error("Stats should be reset")
	}
}

func TestCacheDoubleClose(t *testing.T) {
	cache := NewCache[string, int](5, 50*time.Millisecond)
	cache.Set("key1", 100)

	// First close should work
	cache.Close()

	// Second close should be a no-op (not panic)
	cache.Close()

	// Third close should also be safe
	cache.Close()

	if cache.Len() != 0 {
		t.Errorf("Expected length 0 after close, got %d", cache.Len())
	}
}
