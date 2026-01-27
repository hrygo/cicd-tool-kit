// Package perf tests
package perf

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPool(t *testing.T) {
	pool := NewWorkerPool(4)
	pool.Start()
	defer pool.Stop()

	counter := atomic.Int32{}

	for i := 0; i < 5; i++ {
		if !pool.Submit(func() {
			counter.Add(1)
		}) {
			t.Fatal("Failed to submit task")
		}
	}

	// Wait a bit for tasks to complete
	time.Sleep(200 * time.Millisecond)

	if counter.Load() != 5 {
		t.Errorf("Expected counter 5, got %d", counter.Load())
	}
}

func TestWorkerPoolSubmitWait(t *testing.T) {
	pool := NewWorkerPool(2)
	pool.Start()
	defer pool.Stop()

	result := ""
	err := pool.SubmitWait(func() {
		result = "done"
	})

	if err != nil {
		t.Fatalf("SubmitWait failed: %v", err)
	}

	if result != "done" {
		t.Errorf("Expected result 'done', got '%s'", result)
	}
}

func TestWorkerPoolBatch(t *testing.T) {
	pool := NewWorkerPool(8)
	pool.Start()
	defer pool.Stop()

	counter := atomic.Int32{}

	tasks := make([]func(), 10)
	for i := range tasks {
		tasks[i] = func() {
			counter.Add(1)
		}
	}

	err := pool.Batch(tasks)
	if err != nil {
		t.Fatalf("Batch failed: %v", err)
	}

	if counter.Load() != 10 {
		t.Errorf("Expected counter 10, got %d", counter.Load())
	}
}

func TestMap(t *testing.T) {
	ctx := context.Background()
	items := []int{1, 2, 3, 4, 5}

	results, err := Map(ctx, items, func(n int) (int, error) {
		return n * 2, nil
	}, 2)

	if err != nil {
		t.Fatalf("Map failed: %v", err)
	}

	expected := []int{2, 4, 6, 8, 10}
	if len(results) != len(expected) {
		t.Fatalf("Expected %d results, got %d", len(expected), len(results))
	}

	for i, v := range results {
		if v != expected[i] {
			t.Errorf("Expected %d at index %d, got %d", expected[i], i, v)
		}
	}
}

func TestMapError(t *testing.T) {
	ctx := context.Background()
	items := []int{1, 2, 3, 4, 5}

	_, err := Map(ctx, items, func(n int) (int, error) {
		if n == 3 {
			return 0, fmt.Errorf("error at %d", n)
		}
		return n * 2, nil
	}, 2)

	if err == nil {
		t.Error("Expected error from Map")
	}
}

func TestFilter(t *testing.T) {
	ctx := context.Background()
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	results, err := Filter(ctx, items, func(n int) (bool, error) {
		return n%2 == 0, nil
	}, 2)

	if err != nil {
		t.Fatalf("Filter failed: %v", err)
	}

	// Results should contain exactly the even numbers (order may vary due to concurrency)
	expectedMap := map[int]bool{2: true, 4: true, 6: true, 8: true, 10: true}
	if len(results) != len(expectedMap) {
		t.Errorf("Expected %d results, got %d", len(expectedMap), len(results))
	}

	for _, v := range results {
		if !expectedMap[v] {
			t.Errorf("Unexpected value %d in results", v)
		}
	}
}

func TestEach(t *testing.T) {
	ctx := context.Background()
	items := []int{1, 2, 3, 4, 5}

	counter := atomic.Int32{}

	err := Each(ctx, items, func(n int) error {
		counter.Add(int32(n))
		return nil
	}, 2)

	if err != nil {
		t.Fatalf("Each failed: %v", err)
	}

	if counter.Load() != 15 {
		t.Errorf("Expected counter 15, got %d", counter.Load())
	}
}

func TestEachError(t *testing.T) {
	ctx := context.Background()
	items := []int{1, 2, 3, 4, 5}

	err := Each(ctx, items, func(n int) error {
		if n == 3 {
			return fmt.Errorf("error at %d", n)
		}
		return nil
	}, 2)

	if err == nil {
		t.Error("Expected error from Each")
	}
}

func TestParallel(t *testing.T) {
	ctx := context.Background()

	results, err := Parallel(ctx,
		func() (int, error) { return 1, nil },
		func() (int, error) { return 2, nil },
		func() (int, error) { return 3, nil },
	)

	if err != nil {
		t.Fatalf("Parallel failed: %v", err)
	}

	expected := []int{1, 2, 3}
	if len(results) != len(expected) {
		t.Errorf("Expected %d results, got %d", len(expected), len(results))
	}

	for i, v := range results {
		if v != expected[i] {
			t.Errorf("Expected %d at index %d, got %d", expected[i], i, v)
		}
	}
}

func TestParallelError(t *testing.T) {
	ctx := context.Background()

	_, err := Parallel(ctx,
		func() (int, error) { return 1, nil },
		func() (int, error) { return 0, fmt.Errorf("error") },
		func() (int, error) { return 3, nil },
	)

	if err == nil {
		t.Error("Expected error from Parallel")
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(2)
	defer limiter.Close()

	counter := atomic.Int32{}
	done := make(chan struct{})

	// Start 4 goroutines
	for i := 0; i < 4; i++ {
		go func() {
			limiter.Do(func() error {
				counter.Add(1)
				time.Sleep(50 * time.Millisecond)
				return nil
			})
			done <- struct{}{}
		}()
	}

	// Wait for all to complete
	for i := 0; i < 4; i++ {
		<-done
	}

	if counter.Load() != 4 {
		t.Errorf("Expected counter 4, got %d", counter.Load())
	}
}

func TestRateLimiterConcurrency(t *testing.T) {
	limiter := NewRateLimiter(1)
	defer limiter.Close()

	start := time.Now()
	concurrent := 3

	done := make(chan struct{})
	for i := 0; i < concurrent; i++ {
		go func() {
			limiter.Do(func() error {
				time.Sleep(50 * time.Millisecond)
				return nil
			})
			done <- struct{}{}
		}()
	}

	for i := 0; i < concurrent; i++ {
		<-done
	}

	elapsed := time.Since(start)
	// With concurrency of 1 and 3 tasks of 50ms each, should take at least 150ms
	if elapsed < 140*time.Millisecond {
		t.Errorf("Expected at least 140ms, got %v", elapsed)
	}
}
