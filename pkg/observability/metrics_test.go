// Package observability tests
package observability

import (
	"testing"
	"time"
)

func TestNewMetricsCollector(t *testing.T) {
	config := MetricConfig{
		Enabled:        true,
		FlushInterval:  100 * time.Millisecond,
		MaxSamples:     10,
	}

	m := NewMetricsCollector(config)
	if m == nil {
		t.Fatal("NewMetricsCollector returned nil")
	}

	if !m.enabled {
		t.Error("Metrics should be enabled")
	}
}

func TestCounter(t *testing.T) {
	m := NewMetricsCollector(MetricConfig{Enabled: true})
	labels := map[string]string{"env": "test"}

	m.Counter("test_counter", 1.0, labels)
	if val := m.CounterGet("test_counter", 0); val != 1.0 {
		t.Errorf("Expected counter value 1.0, got %f", val)
	}

	m.Counter("test_counter", 2.0, labels)
	if val := m.CounterGet("test_counter", 0); val != 3.0 {
		t.Errorf("Expected counter value 3.0, got %f", val)
	}
}

func TestGauge(t *testing.T) {
	m := NewMetricsCollector(MetricConfig{Enabled: true})
	labels := map[string]string{"env": "test"}

	m.Gauge("test_gauge", 42.0, labels)

	snapshot := m.GetSnapshot()
	key := "gauge.test_gauge.env:test"
	if val, ok := snapshot[key]; !ok {
		t.Errorf("Gauge not found in snapshot: %s", key)
	} else if val != 42.0 {
		t.Errorf("Expected gauge value 42.0, got %v", val)
	}
}

func TestHistogram(t *testing.T) {
	m := NewMetricsCollector(MetricConfig{Enabled: true})
	labels := map[string]string{"env": "test"}

	m.Histogram("test_hist", 100.0, labels)
	m.Histogram("test_hist", 200.0, labels)

	snapshot := m.GetSnapshot()
	key := "histogram.test_hist.env:test"
	if val, ok := snapshot[key]; !ok {
		t.Error("Histogram not found in snapshot")
	} else {
		samples := val.([]float64)
		if len(samples) != 2 {
			t.Errorf("Expected 2 samples, got %d", len(samples))
		}
	}
}

func TestTiming(t *testing.T) {
	m := NewMetricsCollector(MetricConfig{Enabled: true})
	labels := map[string]string{"operation": "test"}

	m.Timing("operation", 100*time.Millisecond, labels)

	snapshot := m.GetSnapshot()
	countKey := "counter.operation.calls.operation:test"
	if _, ok := snapshot[countKey]; !ok {
		t.Error("Counter not incremented after timing")
	}

	durationKey := "histogram.operation.duration_ms.operation:test"
	if val, ok := snapshot[durationKey]; !ok {
		t.Error("Histogram not created after timing")
	} else {
		if samples, ok := val.([]float64); !ok || len(samples) != 1 {
			t.Error("Duration sample not recorded")
		}
	}
}

func TestSkillExecution(t *testing.T) {
	m := NewMetricsCollector(MetricConfig{Enabled: true})

	m.RecordSkillExecution("code-reviewer", 5*time.Second, true, 1000)
	m.RecordSkillExecution("test-generator", 2*time.Second, false, 500)

	if m.CounterGet("skill.tokens", 0) != 1500 {
		t.Errorf("Expected total tokens 1500, got %f", m.CounterGet("skill.tokens", 0))
	}

	if m.CounterGet("skill.errors", 0) != 1 {
		t.Errorf("Expected 1 error, got %f", m.CounterGet("skill.errors", 0))
	}
}

func TestCacheMetrics(t *testing.T) {
	m := NewMetricsCollector(MetricConfig{Enabled: true})

	m.RecordCacheOperation(true, "get")
	m.RecordCacheOperation(true, "get")
	m.RecordCacheOperation(false, "get")
	m.RecordCacheOperation(false, "set")

	if rate := m.GetCacheHitRate(); rate != 0.5 {
		t.Errorf("Expected cache hit rate 0.5, got %f", rate)
	}
}

func TestGetAverageDuration(t *testing.T) {
	m := NewMetricsCollector(MetricConfig{Enabled: true})
	labels := map[string]string{"operation": "test_op"}

	m.Timing("test_op", 100*time.Millisecond, labels)
	m.Timing("test_op", 200*time.Millisecond, labels)

	avg := m.GetAverageDuration("test_op")
	expected := 150 * time.Millisecond
	if avg != expected {
		t.Errorf("Expected avg duration %v, got %v", expected, avg)
	}
}

func TestFlushMetrics(t *testing.T) {
	m := NewMetricsCollector(MetricConfig{
		Enabled:        true,
		FlushInterval:  50 * time.Millisecond,
		MaxSamples:     5,
	})

	m.Counter("test", 1, nil)

	// Flush should work without error
	if err := m.FlushMetrics(); err != nil {
		t.Errorf("FlushMetrics failed: %v", err)
	}

	// Should have samples - use GetSamples() for safe concurrent access
	samples := m.GetSamples()
	if len(samples) != 1 {
		t.Errorf("Expected 1 sample, got %d", len(samples))
	}
}

func TestNewAuditLogger(t *testing.T) {
	log, err := NewAuditLogger("")
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}

	if log == nil {
		t.Fatal("AuditLogger is nil")
	}

	if log.logger == nil {
		t.Error("Logger not initialized")
	}
}

func TestAuditLogEvent(t *testing.T) {
	log, _ := NewAuditLogger("")

	log.LogEvent("info", "test_event", "test_action", nil)

	entries := log.GetRecentEntries(10)
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].Event != "test_event" {
		t.Errorf("Expected event 'test_event', got '%s'", entries[0].Event)
	}
}

func TestAuditAuthEvent(t *testing.T) {
	log, _ := NewAuditLogger("")

	log.LogAuthEvent("login", "user1", "resource1", true)
	log.LogAuthEvent("login", "user2", "resource2", false)

	entries := log.GetRecentEntries(10)
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}
}

func TestAuditSkillExecution(t *testing.T) {
	log, _ := NewAuditLogger("")

	log.LogSkillExecution("code-reviewer", "user1", 123, true, 5*time.Second)

	entries := log.GetRecentEntries(10)
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].Event != "skill_execution" {
		t.Errorf("Unexpected event: %s", entries[0].Event)
	}
}

func TestNewTracer(t *testing.T) {
	tracer := NewTracer("test-service", true)

	if tracer == nil {
		t.Fatal("NewTracer returned nil")
	}

	if !tracer.enabled {
		t.Error("Tracer should be enabled")
	}
}

func TestTracerSpans(t *testing.T) {
	tracer := NewTracer("test-service", true)

	span1 := tracer.StartSpan("operation1", "", nil)
	span2 := tracer.StartSpan("operation2", span1.ID, nil)

	tracer.EndSpan(span1)
	tracer.EndSpan(span2)

	spans := tracer.GetSpans()
	if len(spans) != 2 {
		t.Errorf("Expected 2 spans, got %d", len(spans))
	}

	if spans[1].ParentID != span1.ID {
		t.Error("Second span should have first span as parent")
	}
}

func TestTracerCurrentSpan(t *testing.T) {
	tracer := NewTracer("test-service", true)

	span := tracer.GetCurrentSpan()
	if span != nil {
		t.Error("Expected no current span initially")
	}

	span1 := tracer.StartSpan("test", "", nil)
	current := tracer.GetCurrentSpan()
	if current != span1 {
		t.Error("Current span should be the one we just started")
	}

	tracer.EndSpan(span1)

	current = tracer.GetCurrentSpan()
	if current != nil {
		t.Error("Expected no current span after ending")
	}
}

func TestMetricsCollectorDoubleClose(t *testing.T) {
	m := NewMetricsCollector(MetricConfig{
		Enabled:        true,
		FlushInterval:  50 * time.Millisecond,
		MaxSamples:     5,
	})

	m.Counter("test", 1, nil)

	// First close should work
	err := m.Close()
	if err != nil {
		t.Fatalf("First Close failed: %v", err)
	}

	// Second close should be safe (not panic)
	err = m.Close()
	if err != nil {
		t.Fatalf("Second Close failed: %v", err)
	}
}
