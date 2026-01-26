// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package observability

import (
	"context"
	"sync"
	"time"
)

// AvailabilityTracker tracks system availability metrics.
// Implements SPEC-STATS-01: Availability Metrics
type AvailabilityTracker struct {
	mu              sync.RWMutex
	uptimeStart     time.Time
	downtimePeriods []*DowntimePeriod
	totalChecks     int
	successfulChecks int
	failedChecks    int
	lastCheck       time.Time
	lastStatus      bool
	window          time.Duration
	checkInterval   time.Duration
}

// DowntimePeriod represents a period of system unavailability.
type DowntimePeriod struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end,omitempty"`
	Duration time.Duration `json:"duration"`
	Reason   string    `json:"reason,omitempty"`
}

// AvailabilityReport represents an availability report.
type AvailabilityReport struct {
	UptimePercent   float64           `json:"uptime_percent"`
	DowntimePercent float64           `json:"downtime_percent"`
	TotalUptime     time.Duration     `json:"total_uptime"`
	TotalDowntime   time.Duration     `json:"total_downtime"`
	DowntimePeriods []*DowntimePeriod `json:"downtime_periods"`
	TotalChecks     int               `json:"total_checks"`
	FailedChecks    int               `json:"failed_checks"`
	PeriodStart     time.Time         `json:"period_start"`
	PeriodEnd       time.Time         `json:"period_end"`
}

// NewAvailabilityTracker creates a new availability tracker.
func NewAvailabilityTracker() *AvailabilityTracker {
	return &AvailabilityTracker{
		uptimeStart:     time.Now(),
		downtimePeriods: make([]*DowntimePeriod, 0),
		window:          24 * time.Hour,
		checkInterval:   time.Minute,
		lastCheck:       time.Now(),
		lastStatus:      true,
	}
}

// RecordCheck records an availability check result.
func (t *AvailabilityTracker) RecordCheck(success bool, reason string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.totalChecks++
	t.lastCheck = now

	if success {
		t.successfulChecks++
		// End any ongoing downtime
		if len(t.downtimePeriods) > 0 {
			last := t.downtimePeriods[len(t.downtimePeriods)-1]
			if last.End.IsZero() {
				last.End = now
				last.Duration = now.Sub(last.Start)
			}
		}
	} else {
		t.failedChecks++
		// Start a new downtime period if not already in one
		if t.lastStatus || len(t.downtimePeriods) == 0 ||
			!t.downtimePeriods[len(t.downtimePeriods)-1].End.IsZero() {
			t.downtimePeriods = append(t.downtimePeriods, &DowntimePeriod{
				Start:  now,
				Reason: reason,
			})
		}
	}

	t.lastStatus = success
}

// GetReport generates an availability report for the time window.
func (t *AvailabilityTracker) GetReport() *AvailabilityReport {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	windowStart := now.Add(-t.window)
	totalDowntime := time.Duration(0)

	// Calculate total downtime within window
	periods := make([]*DowntimePeriod, 0)
	for _, period := range t.downtimePeriods {
		// Check if period overlaps with window
		if period.End.IsZero() {
			period.End = now
			period.Duration = now.Sub(period.Start)
		}

		if period.End.After(windowStart) && period.Start.Before(now) {
			periods = append(periods, period)
			totalDowntime += period.Duration
		}
	}

	totalTime := t.window
	if totalTime == 0 {
		totalTime = now.Sub(t.uptimeStart)
	}

	totalUptime := totalTime - totalDowntime
	uptimePercent := float64(0)
	if totalTime > 0 {
		uptimePercent = (float64(totalUptime) / float64(totalTime)) * 100
	}

	return &AvailabilityReport{
		UptimePercent:   uptimePercent,
		DowntimePercent: 100 - uptimePercent,
		TotalUptime:     totalUptime,
		TotalDowntime:   totalDowntime,
		DowntimePeriods: periods,
		TotalChecks:     t.totalChecks,
		FailedChecks:    t.failedChecks,
		PeriodStart:     windowStart,
		PeriodEnd:       now,
	}
}

// GetUptimePercent returns the current uptime percentage.
func (t *AvailabilityTracker) GetUptimePercent() float64 {
	return t.GetReport().UptimePercent
}

// GetDowntimePercent returns the current downtime percentage.
func (t *AvailabilityTracker) GetDowntimePercent() float64 {
	return t.GetReport().DowntimePercent
}

// IsAvailable returns true if the system is currently available.
func (t *AvailabilityTracker) IsAvailable() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastStatus
}

// GetCurrentDowntime returns the current downtime period if any.
func (t *AvailabilityTracker) GetCurrentDowntime() *DowntimePeriod {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.downtimePeriods) == 0 {
		return nil
	}

	last := t.downtimePeriods[len(t.downtimePeriods)-1]
	if last.End.IsZero() {
		return last
	}

	return nil
}

// SetWindow sets the time window for availability calculation.
func (t *AvailabilityTracker) SetWindow(window time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.window = window
}

// Reset resets the tracker.
func (t *AvailabilityTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.uptimeStart = time.Now()
	t.downtimePeriods = make([]*DowntimePeriod, 0)
	t.totalChecks = 0
	t.successfulChecks = 0
	t.failedChecks = 0
	t.lastCheck = time.Now()
	t.lastStatus = true
}

// StartMonitoring starts a monitoring goroutine.
func (t *AvailabilityTracker) StartMonitoring(ctx context.Context, checkFunc func() (bool, string)) {
	ticker := time.NewTicker(t.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			success, reason := checkFunc()
			t.RecordCheck(success, reason)
		}
	}
}

// SLACompliance checks if the tracker meets SLA requirements.
func (t *AvailabilityTracker) SLACompliance(targetUptime float64) bool {
	return t.GetUptimePercent() >= targetUptime
}

// HealthChecker defines a health check function.
type HealthChecker func(ctx context.Context) error

// HealthTracker combines availability tracking with health checks.
type HealthTracker struct {
	availability *AvailabilityTracker
	checkers     map[string]HealthChecker
	timeout      time.Duration
}

// NewHealthTracker creates a new health tracker.
func NewHealthTracker() *HealthTracker {
	return &HealthTracker{
		availability: NewAvailabilityTracker(),
		checkers:     make(map[string]HealthChecker),
		timeout:      30 * time.Second,
	}
}

// AddChecker adds a health checker.
func (h *HealthTracker) AddChecker(name string, checker HealthChecker) {
	h.checkers[name] = checker
}

// CheckHealth runs all health checks.
func (h *HealthTracker) CheckHealth(ctx context.Context) map[string]error {
	results := make(map[string]error)
	allSuccess := true

	for name, checker := range h.checkers {
		checkCtx, cancel := context.WithTimeout(ctx, h.timeout)
		defer cancel()

		err := checker(checkCtx)
		results[name] = err

		if err != nil {
			allSuccess = false
		}
	}

	reason := ""
	if !allSuccess {
		reason = "health check failed"
	}
	h.availability.RecordCheck(allSuccess, reason)

	return results
}

// GetReport returns the availability report.
func (h *HealthTracker) GetReport() *AvailabilityReport {
	return h.availability.GetReport()
}

// IsHealthy returns true if all health checks pass.
func (h *HealthTracker) IsHealthy() bool {
	return h.availability.IsAvailable()
}

// Status represents the system status.
type Status struct {
	Healthy      bool              `json:"healthy"`
	UptimePercent float64          `json:"uptime_percent"`
	Checks       map[string]string `json:"checks"`
	Timestamp    time.Time         `json:"timestamp"`
}

// GetStatus returns the current system status.
func (h *HealthTracker) GetStatus(ctx context.Context) *Status {
	checkResults := h.CheckHealth(ctx)
	checks := make(map[string]string)

	allHealthy := true
	for name, err := range checkResults {
		if err != nil {
			checks[name] = err.Error()
			allHealthy = false
		} else {
			checks[name] = "ok"
		}
	}

	return &Status{
		Healthy:      allHealthy,
		UptimePercent: h.availability.GetUptimePercent(),
		Checks:       checks,
		Timestamp:    time.Now(),
	}
}
