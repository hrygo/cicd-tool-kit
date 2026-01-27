// Package observability provides metrics and tracing functionality
package observability

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// MetricsCollector collects and reports metrics
type MetricsCollector struct {
	mu                 sync.RWMutex
	metrics            map[string]interface{}
	samples            []*MetricSample
	maxSamples         int
	maxHistogramSamples int
	enabled            bool
	flushInterval      time.Duration
	stopCh             chan struct{}
	closeOnce          sync.Once
}

// MetricSample represents a single metric sample at a point in time
type MetricSample struct {
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics"`
}

// MetricConfig configures metrics collection
type MetricConfig struct {
	Enabled            bool          `json:"enabled" yaml:"enabled"`
	FlushInterval      time.Duration `json:"flush_interval" yaml:"flush_interval"`
	MaxSamples         int           `json:"max_samples" yaml:"max_samples"`
	MaxHistogramSamples int          `json:"max_histogram_samples" yaml:"max_histogram_samples"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(config MetricConfig) *MetricsCollector {
	if config.MaxSamples == 0 {
		config.MaxSamples = 1000
	}
	if config.MaxHistogramSamples == 0 {
		config.MaxHistogramSamples = 10000
	}
	if config.FlushInterval == 0 {
		config.FlushInterval = 30 * time.Second
	}

	m := &MetricsCollector{
		metrics:            make(map[string]interface{}),
		samples:            make([]*MetricSample, 0, config.MaxSamples),
		maxSamples:         config.MaxSamples,
		maxHistogramSamples: config.MaxHistogramSamples,
		enabled:            config.Enabled,
		flushInterval:      config.FlushInterval,
		stopCh:             make(chan struct{}),
	}

	if m.enabled {
		go m.backgroundFlush()
	}

	return m
}

// formatLabels converts labels map to a consistent string format
// e.g., map[string]string{"env": "test", "region": "us"} -> "env:test-region:us"
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	// Sort keys for consistency
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + ":" + labels[k]
	}
	return strings.Join(parts, "-")
}

// Counter increments a counter metric
func (m *MetricsCollector) Counter(name string, value float64, labels map[string]string) {
	if !m.enabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	labelStr := formatLabels(labels)
	key := "counter." + name
	if labelStr != "" {
		key += "." + labelStr
	}
	currentVal := float64(0)
	if val, ok := m.metrics[key]; ok {
		if f, ok := val.(float64); ok {
			currentVal = f
		}
	}
	m.metrics[key] = currentVal + value
}

// CounterGet gets the sum of all counter values matching the name
func (m *MetricsCollector) CounterGet(name string, labelIdx int) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	prefix := "counter." + name
	sum := float64(0)
	for key, val := range m.metrics {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			if f, ok := val.(float64); ok {
				sum += f
			}
		}
	}
	return sum
}

// Gauge sets a gauge metric
func (m *MetricsCollector) Gauge(name string, value float64, labels map[string]string) {
	if !m.enabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	labelStr := formatLabels(labels)
	key := "gauge." + name
	if labelStr != "" {
		key += "." + labelStr
	}
	m.metrics[key] = value
}

// Histogram records a histogram sample
func (m *MetricsCollector) Histogram(name string, value float64, labels map[string]string) {
	if !m.enabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	labelStr := formatLabels(labels)
	key := "histogram." + name
	if labelStr != "" {
		key += "." + labelStr
	}
	var samples []float64
	if val, ok := m.metrics[key]; ok {
		// Type assertion with comma-ok for safety
		if existingSamples, ok := val.([]float64); ok {
			samples = existingSamples
		}
		// If type assertion fails, samples remains nil and we start fresh
	}

	samples = append(samples, value)
	// Limit histogram size to prevent unbounded growth
	if len(samples) > m.maxHistogramSamples {
		// Keep the most recent samples by discarding the oldest
		samples = samples[len(samples)-m.maxHistogramSamples:]
	}
	m.metrics[key] = samples
}

// Timing records the duration of an operation
func (m *MetricsCollector) Timing(name string, duration time.Duration, labels map[string]string) {
	if !m.enabled {
		return
	}

	m.Histogram(name+".duration_ms", float64(duration.Milliseconds()), labels)
	m.Counter(name+".calls", 1, labels)
}

// RecordSkillExecution records metrics for a skill execution
func (m *MetricsCollector) RecordSkillExecution(skillName string, duration time.Duration, success bool, tokensUsed int) {
	labels := map[string]string{
		"skill": skillName,
		"status": func() string {
			if success {
				return "success"
			}
			return "failure"
		}(),
	}

	m.Timing("skill.execution", duration, labels)
	m.Counter("skill.tokens", float64(tokensUsed), labels)

	if !success {
		m.Counter("skill.errors", 1, labels)
	}
}

// RecordCacheOperation records cache metrics
func (m *MetricsCollector) RecordCacheOperation(hit bool, operation string) {
	labels := map[string]string{
		"operation": operation,
		"result": func() string {
			if hit {
				return "hit"
			}
			return "miss"
		}(),
	}

	m.Counter("cache.operations", 1, labels)
}

// GetSnapshot returns a snapshot of current metrics
func (m *MetricsCollector) GetSnapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := make(map[string]interface{})
	for k, v := range m.metrics {
		snapshot[k] = v
	}
	return snapshot
}

// FlushMetrics flushes current metrics to storage/output
func (m *MetricsCollector) FlushMetrics() error {
	if !m.enabled {
		return nil
	}

	// Get snapshot first (with RLock)
	snapshot := m.GetSnapshot()

	// Then Lock for updating samples
	m.mu.Lock()
	sample := &MetricSample{
		Timestamp: time.Now(),
		Metrics:   snapshot,
	}

	m.samples = append(m.samples, sample)
	if len(m.samples) > m.maxSamples {
		m.samples = m.samples[1:]
	}
	m.mu.Unlock()

	// In production, this would send to a metrics backend
	// For now, we'll log if verbose mode is on
	if os.Getenv("VERBOSE") == "true" {
		data, err := json.MarshalIndent(sample, "", "  ")
		if err != nil {
			log.Printf("[Metrics] ERROR: failed to marshal sample: %v\n", err)
		} else {
			log.Printf("[Metrics] %s\n", string(data))
		}
	}

	return nil
}

// backgroundFlush periodically flushes metrics
func (m *MetricsCollector) backgroundFlush() {
	ticker := time.NewTicker(m.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.FlushMetrics()
		case <-m.stopCh:
			return
		}
	}
}

// Close stops the metrics collector
// Safe to call multiple times - subsequent calls are no-ops
func (m *MetricsCollector) Close() error {
	var err error
	m.closeOnce.Do(func() {
		// Signal background flush to stop
		close(m.stopCh)

		// Do a final flush before returning
		// This happens while backgroundFlush goroutine is exiting
		err = m.FlushMetrics()
	})
	return err
}

// GetCacheHitRate calculates the cache hit rate
func (m *MetricsCollector) GetCacheHitRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Sum all cache.operations counters with result=hit or result=miss
	hitSum := float64(0)
	missSum := float64(0)
	prefix := "counter.cache.operations"
	for key, val := range m.metrics {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			if f, ok := val.(float64); ok {
				// Check if key contains "-result:hit" or "-result:miss"
				if strings.Contains(key, "-result:hit") {
					hitSum += f
				} else if strings.Contains(key, "-result:miss") {
					missSum += f
				}
			}
		}
	}

	total := hitSum + missSum
	if total == 0 {
		return 0
	}

	return hitSum / total
}

// GetAverageDuration gets average duration for an operation
func (m *MetricsCollector) GetAverageDuration(operationName string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Search for histogram keys matching the operation name with any labels
	prefix := "histogram." + operationName + ".duration_ms"
	allSamples := make([]float64, 0)
	for key, val := range m.metrics {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			if samples, ok := val.([]float64); ok {
				allSamples = append(allSamples, samples...)
			}
		}
	}

	if len(allSamples) == 0 {
		return 0
	}

	var sum float64
	for _, s := range allSamples {
		sum += s
	}
	avg := sum / float64(len(allSamples))
	return time.Duration(avg) * time.Millisecond
}

// AuditLogger logs security-relevant events for audit trails
type AuditLogger struct {
	mu       sync.Mutex
	entries  []AuditEntry
	logger   *log.Logger
	file     *os.File
	closeOnce sync.Once
}

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                `json:"level"` // info, warning, error
	Event     string                `json:"event"`
	User      string                `json:"user,omitempty"`
	Resource  string                `json:"resource,omitempty"`
	Action    string                `json:"action,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logFile string) (*AuditLogger, error) {
	var logger *log.Logger
	var file *os.File
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to open audit log file: %w", err)
		}
		file = f
		logger = log.New(f, "", log.LstdFlags)
	} else {
		logger = log.New(os.Stdout, "[AUDIT]", log.LstdFlags)
	}

	return &AuditLogger{
		entries: make([]AuditEntry, 0, 1000),
		logger:  logger,
		file:    file,
	}, nil
}

// LogEvent logs an audit event
func (a *AuditLogger) LogEvent(level, event, action string, details map[string]interface{}) {
	entry := AuditEntry{
		Timestamp: time.Now(),
		Level:     level,
		Event:     event,
		Action:    action,
		Details:   details,
	}

	a.mu.Lock()
	a.entries = append(a.entries, entry)
	a.mu.Unlock()

	// Write to log
	data, err := json.Marshal(entry)
	if err != nil {
		a.logger.Printf("ERROR: failed to marshal audit entry: %v | entry: %+v\n", err, entry)
	} else {
		a.logger.Printf("%s\n", string(data))
	}
}

// LogAuthEvent logs authentication/authorization events
func (a *AuditLogger) LogAuthEvent(event, user, resource string, success bool) {
	a.LogEvent(
		func() string {
			if success {
				return "info"
			}
			return "warning"
		}(),
		event,
		fmt.Sprintf("auth_%s", event),
		map[string]interface{}{
			"user":     user,
			"resource": resource,
			"success":  success,
		},
	)
}

// LogSkillExecution logs skill execution for audit
func (a *AuditLogger) LogSkillExecution(skillName, user string, prID int, success bool, duration time.Duration) {
	a.LogEvent(
		"info",
		"skill_execution",
		fmt.Sprintf("skill_%s", skillName),
		map[string]interface{}{
			"user":     user,
			"skill":    skillName,
			"pr_id":    prID,
			"success":  success,
			"duration": duration.String(),
		},
	)
}

// LogConfigChange logs configuration changes
func (a *AuditLogger) LogConfigChange(user, changedFile string) {
	a.LogEvent(
		"info",
		"config_change",
		"config_updated",
		map[string]interface{}{
			"user":        user,
			"changed_file": changedFile,
		},
	)
}

// LogSecurityEvent logs security-related events
func (a *AuditLogger) LogSecurityEvent(event, severity, user string, details map[string]interface{}) {
	a.LogEvent(
		severity,
		event,
		"security_event",
		map[string]interface{}{
			"user":    user,
			"details": details,
		},
	)
}

// GetRecentEntries returns recent audit entries
func (a *AuditLogger) GetRecentEntries(count int) []AuditEntry {
	a.mu.Lock()
	defer a.mu.Unlock()

	if count > len(a.entries) {
		count = len(a.entries)
	}

	start := len(a.entries) - count
	return a.entries[start:]
}

// Clear clears all audit entries (with care!)
func (a *AuditLogger) Clear() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.entries = make([]AuditEntry, 0, 1000)
	return nil
}

// Close closes the audit logger and releases resources
// Safe to call multiple times - subsequent calls are no-ops
func (a *AuditLogger) Close() error {
	a.closeOnce.Do(func() {
		if a.file != nil {
			a.file.Close()
		}
	})
	return nil
}

// Tracer provides distributed tracing capabilities
type Tracer struct {
	serviceName string
	enabled     bool
	spans       []*Span
	current    *Span
	mu         sync.Mutex
}

// Span represents a trace span
type Span struct {
	ID        string    `json:"id"`
	ParentID  string    `json:"parent_id,omitempty"`
	Name      string    `json:"name"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Duration  float64  `json:"duration_ms,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
	Events    []SpanEvent `json:"events,omitempty"`
}

// SpanEvent represents an event within a span
type SpanEvent struct {
	Time      time.Time `json:"time"`
	Name      string    `json:"name"`
	Payload   string    `json:"payload,omitempty"`
}

// NewTracer creates a new tracer
func NewTracer(serviceName string, enabled bool) *Tracer {
	return &Tracer{
		serviceName: serviceName,
		enabled:     enabled,
		spans:       make([]*Span, 0),
	}
}

// StartSpan starts a new trace span
func (t *Tracer) StartSpan(name string, parentID string, tags map[string]string) *Span {
	span := &Span{
		ID:        fmt.Sprintf("%s-%d", name, time.Now().UnixNano()),
		ParentID:  parentID,
		Name:      name,
		StartTime: time.Now(),
		Tags:      tags,
		Events:    make([]SpanEvent, 0),
	}

	t.mu.Lock()
	if parentID == "" {
		t.current = span
	}
	t.spans = append(t.spans, span)
	t.mu.Unlock()

	return span
}

// EndSpan ends a trace span
func (t *Tracer) EndSpan(span *Span) {
	if span == nil {
		return
	}

	span.EndTime = time.Now()
	span.Duration = float64(span.EndTime.Sub(span.StartTime).Milliseconds())

	t.mu.Lock()
	if t.current == span {
		t.current = nil
	}
	t.mu.Unlock()
}

// AddEvent adds an event to the current span
func (t *Tracer) AddEvent(name, payload string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.current != nil {
		t.current.Events = append(t.current.Events, SpanEvent{
			Time:    time.Now(),
			Name:    name,
			Payload: payload,
		})
	}
}

// GetCurrentSpan returns the active span
func (t *Tracer) GetCurrentSpan() *Span {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.current
}

// GetSpans returns all completed spans
func (t *Tracer) GetSpans() []*Span {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.spans
}
