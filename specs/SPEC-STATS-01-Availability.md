# SPEC-STATS-01: Availability & SLA Calculation

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24
**Covers**: PRD 5.2 (可用性 99.5%)

## 1. 概述 (Overview)

本 Spec 定义 `cicd-ai-toolkit` 的可用性计算方法、SLA (Service Level Agreement) 策略以及 downtime 处理机制，确保系统能够满足 PRD 5.2 中定义的 99.5% 可用性目标。

## 2. 核心职责 (Core Responsibilities)

- **可用性计算**: 准确计算系统可用性百分比
- **SLA 监控**: 实时追踪 SLA 合规状态
- **Downtime 记录**: 记录和分类所有系统停机事件
- **告警机制**: 在 SLA 违规时触发告警
- **报告生成**: 生成月度/季度 SLA 报告

## 3. 可用性定义 (Availability Definition)

### 3.1 可用性公式

```
可用性 = (总时间 - Downtime) / 总时间 × 100%

其中:
- 总时间 = 统计周期时长 (月/季/年)
- Downtime = 计划外停机时间 + 部分计划内停机时间
```

### 3.2 Downtime 分类

| 类型 | 说明 | 计入可用性 | 示例 |
|------|------|-----------|------|
| **Unplanned** | 系统故障、意外停机 | ✅ 是 | API 服务器宕机、网络中断 |
| **Planned** | 计划内维护 | ❌ 否 (定义的维护窗口) | 系统升级、基础设施维护 |
| **Partial Outage** | 部分功能不可用 | ⚠️ 部分 (按影响比例计算) | 某个平台适配器失败 |
| **Degraded** | 性能下降但可用 | ⚠️ 部分 (响应时间 > 阈值时计入) | 响应时间 > 60s |

### 3.3 可用性目标

根据 PRD 5.2：

| 指标 | 目标值 | 测量周期 |
|------|--------|----------|
| **整体可用性** | 99.5% | 月度滚动 |
| **CI 系统可用性** | 99.9% | 月度滚动 |
| **API 可用性** | 99.95% | 月度滚动 |

**对应允许的年度 Downtime**:

| 可用性 | 年度允许停机 | 月度允许停机 |
|--------|-------------|-------------|
| 99.5% | 43.8 小时 | ~3.65 小时 |
| 99.9% | 8.76 小时 | ~43 分钟 |
| 99.95% | 4.38 小时 | ~22 分钟 |

## 4. SLA 计算实现

### 4.1 数据结构

```go
type SLAMonitor struct {
    calculator  *AvailabilityCalculator
    recorder    *IncidentRecorder
    alertor     *Alerter
    reporter    *SLAReporter
}

type AvailabilityCalculator struct {
    period      time.Time // Measurement period (month, quarter, year)
    totalTime  time.Duration
    downtime    time.Duration
    incidents   []Incident
}

type Incident struct {
    ID          string
    StartTime   time.Time
    EndTime     time.Time
    Duration    time.Duration
    Type        IncidentType
    Severity    IncidentSeverity
    AffectedServices []string
    Impact      Impact
    Resolution  string
}

type IncidentType string

const (
    IncidentTypeUnplanned  IncidentType = "unplanned"
    IncidentTypePlanned    IncidentType = "planned"
    IncidentTypePartial    IncidentType = "partial"
    IncidentTypeDegraded   IncidentType = "degraded"
)

type IncidentSeverity string

const (
    SeverityCritical  IncidentSeverity = "critical" // 完全不可用
    SeverityMajor     IncidentSeverity = "major"     // 核心功能不可用
    SeverityMinor     IncidentSeverity = "minor"     // 部分功能受影响
    SeverityLow       IncidentSeverity = "low"       // 性能下降
)

type Impact struct {
    UserCount     int
    RequestCount  int
    AffectedPct   float64 // 0-1
}
```

### 4.2 可用性计算器

```go
func (ac *AvailabilityCalculator) CalculateAvailability() *AvailabilityResult {
    var effectiveDowntime time.Duration

    for _, incident := range ac.incidents {
        // Exclude planned maintenance from calculation
        if incident.Type == IncidentTypePlanned {
            continue
        }

        // Apply impact percentage for partial outages
        downtime := incident.Duration
        if incident.Impact.AffectedPct > 0 && incident.Impact.AffectedPct < 1 {
            downtime = time.Duration(float64(downtime) * incident.Impact.AffectedPct)
        }

        effectiveDowntime += downtime
    }

    availability := float64(ac.totalTime-effectiveDowntime) / float64(ac.totalTime) * 100

    return &AvailabilityResult{
        Period:         ac.period,
        TotalTime:      ac.totalTime,
        TotalDowntime:  effectiveDowntime,
        Availability:   availability,
        SLACompliant:   availability >= 99.5,
        IncidentCount:  len(ac.incidents),
        Incidents:      ac.incidents,
    }
}

type AvailabilityResult struct {
    Period         time.Time
    TotalTime      time.Duration
    TotalDowntime  time.Duration
    Availability   float64
    SLACompliant   bool
    IncidentCount  int
    Incidents      []Incident
}

func (ar *AvailabilityResult) String() string {
    return fmt.Sprintf(
        "可用性: %.2f%% | 停机: %s | 事件: %d",
        ar.Availability,
        ar.TotalDowntime.Round(time.Second),
        ar.IncidentCount,
    )
}
```

### 4.3 降级性能计算

当系统性能下降但未完全不可用时：

```go
func (ac *AvailabilityCalculator) CalculateDegradedPerformance() (downtime time.Duration) {
    // 获取性能指标
    metrics := ac.getPerformanceMetrics()

    // P50, P95, P99 延迟
    p50 := metrics.Percentile(50)
    p95 := metrics.Percentile(95)
    p99 := metrics.Percentile(99)

    var degradedTime time.Duration

    // P99 > 60s = 完全不可用
    if p99 > 60*time.Second {
        degradedTime = ac.getDurationAboveThreshold(60 * time.Second)
        return degradedTime
    }

    // P95 > 30s = 50% 不可用
    if p95 > 30*time.Second {
        degradedTime = ac.getDurationAboveThreshold(30 * time.Second)
        return time.Duration(float64(degradedTime) * 0.5)
    }

    // P50 > 10s = 20% 不可用
    if p50 > 10*time.Second {
        degradedTime = ac.getDurationAboveThreshold(10 * time.Second)
        return time.Duration(float64(degradedTime) * 0.2)
    }

    return 0
}

func (ac *AvailabilityCalculator) getDurationAboveThreshold(threshold time.Duration) time.Duration {
    // 计算响应时间超过阈值的总时长
    var total time.Duration
    samples := ac.metricsSamples

    for _, sample := range samples {
        if sample.Latency > threshold {
            // 估算到上一次采样的时间
            total += sample.Interval
        }
    }

    return total
}
```

## 5. 事故记录 (Incident Recording)

### 5.1 事故检测

```go
type IncidentDetector struct {
    healthChecks []HealthCheck
    threshold  time.Duration
}

type HealthCheck interface {
    Check(ctx context.Context) error
    ServiceName() string
}

func (id *IncidentDetector) Monitor(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    var currentIncident *Incident

    for {
        select {
        case <-ctx.Done():
            id.closeIncident(currentIncident)
            return

        case <-ticker.C:
            allHealthy := true
            var failedServices []string

            for _, hc := range id.healthChecks {
                if err := hc.Check(ctx); err != nil {
                    allHealthy = false
                    failedServices = append(failedServices, hc.ServiceName())
                }
            }

            if !allHealthy && currentIncident == nil {
                // Start new incident
                currentIncident = &Incident{
                    ID:              generateIncidentID(),
                    StartTime:       time.Now(),
                    Type:            IncidentTypeUnplanned,
                    Severity:        id.determineSeverity(failedServices),
                    AffectedServices: failedServices,
                }
                id.recorder.Record(currentIncident)
                id.alertor.SendAlert(currentIncident)
            }

            if allHealthy && currentIncident != nil {
                // End incident
                currentIncident.EndTime = time.Now()
                currentIncident.Duration = time.Since(currentIncident.StartTime)
                id.recorder.Update(currentIncident)
                currentIncident = nil
            }
        }
    }
}
```

### 5.2 事故存储

```go
type IncidentRecorder struct {
    storage    IncidentStorage
    compressor *Compressor // 压缩旧数据
}

type IncidentStorage interface {
    Record(incident *Incident) error
    Update(incident *Incident) error
    Query(query IncidentQuery) ([]*Incident, error)
}

type IncidentQuery struct {
    StartTime   time.Time
    EndTime     time.Time
    Types       []IncidentType
    Severities  []IncidentSeverity
    MinSeverity string
}

func (ir *IncidentRecorder) Record(incident *Incident) error {
    incident.ID = generateIncidentID()

    // Validate
    if incident.StartTime.IsZero() {
        return fmt.Errorf("incident start time is required")
    }

    // Store
    return ir.storage.Record(incident)
}
```

## 6. SLA 监控与告警

### 6.1 SLA 追踪

```go
type SLATracker struct {
    slaTarget   float64 // 99.5
    windowSize  time.Duration
    recorder    *IncidentRecorder
}

type SLAStatus struct {
    CurrentAvailability float64
    SLATarget          float64
    IsCompliant         bool
    RemainingBudget     time.Duration // 允许的剩余停机时间
    RiskLevel           RiskLevel
}

func (st *SLATracker) GetStatus(ctx context.Context) (*SLAStatus, error) {
    // 计算滚动窗口的可用性
    windowStart := time.Now().Add(-st.windowSize)

    incidents, err := st.recorder.Query(IncidentQuery{
        StartTime: windowStart,
        EndTime:   time.Now(),
    })
    if err != nil {
        return nil, err
    }

    result := st.calculateAvailability(windowStart, time.Now(), incidents)

    status := &SLAStatus{
        CurrentAvailability: result.Availability,
        SLATarget:          st.slaTarget,
        IsCompliant:         result.Availability >= st.slaTarget,
    }

    // 计算剩余预算
    totalAllowed := time.Duration(float64(st.windowSize) * (1 - st.slaTarget/100))
    used := result.TotalDowntime
    status.RemainingBudget = totalAllowed - used

    // 评估风险
    status.RiskLevel = st.assessRisk(status)

    return status, nil
}

func (st *SLATracker) assessRisk(status *SLAStatus) RiskLevel {
    usedBudget := 1 - (status.RemainingBudget.Seconds() / time.Duration(float64(st.windowSize)*(1-status.SLATarget/100).Seconds())

    switch {
    case usedBudget < 0.5:
        return RiskLevelLow
    case usedBudget < 0.8:
        return RiskLevelMedium
    case usedBudget < 1.0:
        return RiskLevelHigh
    default:
        return RiskLevelCritical
}
```

### 6.2 告警规则

```go
type Alerter struct {
    channels    []AlertChannel
    escalations *EscalationPolicy
}

type AlertLevel string

const (
    AlertLevelInfo     AlertLevel = "info"
    AlertLevelWarning  AlertLevel = "warning"
    AlertLevelCritical AlertLevel = "critical"
)

type Alert struct {
    Level      AlertLevel
    Title      string
    Message    string
    Metrics    map[string]interface{}
    Incident   *Incident
}

func (a *Alerter) EvaluateAndAlert(status *SLAStatus) error {
    var level AlertLevel
    var message string

    switch status.RiskLevel {
    case RiskLevelLow:
        level = AlertLevelInfo
        message = fmt.Sprintf("SLA status good. Availability: %.2f%%, Budget: %s remaining",
            status.CurrentAvailability, status.RemainingBudget)

    case RiskLevelMedium:
        level = AlertLevelWarning
        message = fmt.Sprintf("SLA warning. Availability: %.2f%%, Budget: %s remaining",
            status.CurrentAvailability, status.RemainingBudget)

    case RiskLevelHigh:
        level = AlertLevelWarning
        message = fmt.Sprintf("SLA at risk. Availability: %.2f%%, Budget: %s remaining",
            status.CurrentAvailability, status.RemainingBudget)

    case RiskLevelCritical:
        level = AlertLevelCritical
        message = fmt.Sprintf("SLA BREACH IMMINENT. Availability: %.2f%%, Budget: %s remaining",
            status.CurrentAvailability, status.RemainingBudget)
    }

    alert := &Alert{
        Level:   level,
        Title:   fmt.Sprintf("SLA %s", strings.ToUpper(string(level))),
        Message: message,
        Metrics: map[string]interface{}{
            "availability":    status.CurrentAvailability,
            "sla_target":     status.SLATarget,
            "is_compliant":    status.IsCompliant,
            "remaining_budget": status.RemainingBudget.String(),
            "risk_level":      status.RiskLevel,
        },
    }

    // Send to all channels
    for _, ch := range a.channels {
        ch.Send(alert)
    }

    return nil
}
```

### 6.3 告警通道

```go
type AlertChannel interface {
    Send(alert *Alert) error
    Name() string
}

// Slack Channel
type SlackAlertChannel struct {
    webhookURL string
}

func (sac *SlackAlertChannel) Send(alert *Alert) error {
    color := map[AlertLevel]string{
        AlertLevelInfo:     "36a64f", // blue
        AlertLevelWarning:  "ffeb3b", // yellow
        AlertLevelCritical: "ff0000", // red
    }[alert.Level]

    payload := map[string]interface{}{
        "attachments": []map[string]interface{}{
            {
                "color":  color,
                "title":  alert.Title,
                "text":   alert.Message,
                "fields": []map[string]interface{}{
                    {"title": "Availability", "value": fmt.Sprintf("%.2f%%", alert.Metrics["availability"]), "short": true},
                    {"title": "SLA Target", "value": fmt.Sprintf("%.2f%%", alert.Metrics["sla_target"]), "short": true},
                    {"title": "Budget Remaining", "value": alert.Metrics["remaining_budget"].(string), "short": true},
                },
            },
        },
    }

    return sendSlackWebhook(sac.webhookURL, payload)
}

// PagerDuty Channel (for critical alerts)
type PagerDutyAlertChannel struct {
    integrationKey string
}

func (pdac *PagerDutyAlertChannel) Send(alert *Alert) error {
    if alert.Level != AlertLevelCritical {
        return nil // Only send critical alerts
    }

    payload := map[string]interface{}{
        "routing_key": pdac.integrationKey,
        "event_action": "trigger",
        "payload": map[string]interface{}{
            "summary":   alert.Title,
            "severity":  string(alert.Level),
            "source":    "cicd-ai-toolkit",
            "timestamp": time.Now().Unix(),
            "custom_details": alert.Message,
        },
    }

    return sendPagerDutyEvent(pdac.integrationKey, payload)
}
```

## 7. 报告生成

### 7.1 月度 SLA 报告

```go
type SLAReportGenerator struct {
    recorder *IncidentRecorder
    template *ReportTemplate
}

type MonthlySLAReport struct {
    Period          Month
    Year           int
    Availability    float64
    SLATarget      float64
    Compliance    bool
    TotalDowntime  time.Duration
    IncidentCount  int
    Incidents      []IncidentSummary
    MTTR           time.Duration // Mean Time To Recovery
    MTBF           time.Duration // Mean Time Between Failures
    TopCauses      []CauseBreakdown
}

type IncidentSummary struct {
    ID            string
    Type          string
    Severity      string
    Duration      time.Duration
    Services      []string
    RootCause     string
}

func (srg *SLAReportGenerator) GenerateMonthly(year int, month time.Month) (*MonthlySLAReport, error) {
    start := time.Date(year, month, 1, 0, 0, 0, time.UTC)
    end := start.AddDate(0, 1, 0).Add(-time.Second)

    // Query incidents
    incidents, err := srg.recorder.Query(IncidentQuery{
        StartTime: start,
        EndTime:   end,
    })
    if err != nil {
        return nil, err
    }

    report := &MonthlySLAReport{
        Period:     month,
        Year:      year,
        Incidents:  srg.summarizeIncidents(incidents),
    }

    // Calculate availability
    report.TotalDowntime = srg.calculateTotalDowntime(incidents)
    totalTime := end.Sub(start)
    report.Availability = float64(totalTime-report.TotalDowntime) / float64(totalTime) * 100

    // Set SLA target and check compliance
    report.SLATarget = 99.5
    report.Compliance = report.Availability >= report.SLATarget
    report.IncidentCount = len(incidents)

    // Calculate MTTR and MTBF
    if len(incidents) > 0 {
        report.MTTR = srg.calculateMTTR(incidents)
        report.MTBF = srg.calculateMTBF(incidents, totalTime)
    }

    // Analyze top causes
    report.TopCauses = srg.analyzeCauses(incidents)

    return report, nil
}

func (srg *SLAReportGenerator) calculateMTTR(incidents []Incident) time.Duration {
    var total time.Duration
    count := 0

    for _, inc := range incidents {
        if !inc.EndTime.IsZero() {
            total += inc.Duration
            count++
        }
    }

    if count == 0 {
        return 0
    }
    return total / time.Duration(count)
}

func (srg *SLAReportGenerator) calculateMTBF(incidents []Incident, period time.Duration) time.Duration {
    if len(incidents) == 0 {
        return period
    }

    // MTBF = 总时间 / 事故次数
    return period / time.Duration(len(incidents))
}
```

### 7.2 报告输出

```go
func (r *MonthlySLAReport) GenerateMarkdown() string {
    var sb strings.Builder

    sb.WriteString(fmt.Sprintf("# SLA Report - %s %d\n\n", r.Period.String(), r.Year))

    // Summary
    sb.WriteString("## Summary\n\n")
    sb.WriteString(renderMetric("Availability", fmt.Sprintf("%.2f%%", r.Availability), r.Compliance))
    sb.WriteString(renderMetric("SLA Target", fmt.Sprintf("%.2f%%", r.SLATarget), true))
    sb.WriteString(renderMetric("Total Downtime", r.TotalDowntime.String(), r.TotalDowntime < 2*time.Hour))
    sb.WriteString(fmt.Sprintf("**Incidents**: %d\n\n", r.IncidentCount))
    sb.WriteString(fmt.Sprintf("**MTTR**: %s\n\n", r.MTTR.Round(time.Minute)))
    sb.WriteString(fmt.Sprintf("**MTBF**: %s\n\n", r.MTBF.Round(time.Hour)))

    // Incident Breakdown
    sb.WriteString("## Incidents\n\n")
    sb.WriteString("| ID | Type | Severity | Duration | Services |\n")
    sb.WriteString("|----|------|----------|----------|----------|\n")
    for _, inc := range r.Incidents {
        sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
            inc.ID, inc.Type, inc.Severity, inc.Duration, strings.Join(inc.Services, ", ")))
    }
    sb.WriteString("\n")

    // Top Causes
    sb.WriteString("## Top Causes\n\n")
    for i, cause := range r.TopCauses {
        sb.WriteString(fmt.Sprintf("%d. **%s**: %d incidents (%.1f%%)\n",
            i+1, cause.Category, cause.Count, cause.Percentage))
    }

    return sb.String()
}

func renderMetric(name, value string, good bool) string {
    icon := "✅"
    if !good {
        icon = "❌"
    }
    return fmt.Sprintf("%s **%s**: %s", icon, name, value)
}
```

## 8. 配置示例

```yaml
# .cicd-ai-toolkit.yaml
sla:
  enabled: true

  # SLA 目标
  targets:
    overall: 99.5      # 整体可用性
    ci_system: 99.9   # CI 系统
    api: 99.95        # API 服务

  # 监控窗口
  window: "30d"        # 滚动窗口 (7d, 30d, 90d)

  # 性能阈值 (超过则计入降级)
  performance:
    degraded_p50: 10s    # P50 超过 10s = 降级
    degraded_p95: 30s    # P95 超过 30s = 部分故障
    unavailable_p99: 60s # P99 超过 60s = 完全不可用

  # 告警配置
  alerts:
    - level: warning
      threshold: 0.8    # 使用 80% 预算时告警
      channels: ["slack", "email"]

    - level: critical
      threshold: 0.95   # 使用 95% 预算时告警
      channels: ["slack", "pagerduty"]

  # 维护窗口 (不计入可用性)
  maintenance_windows:
    - start: "Sunday 02:00 UTC"
      duration: "2h"
      timezone: "UTC"

  # 事故存储
  storage:
    backend: "s3"        # local, s3, database
    retention: "2y"
```

## 9. 依赖关系 (Dependencies)

- **Related**: [SPEC-OPS-01](./SPEC-OPS-01-Observability.md) - 事故日志记录
- **Related**: [SPEC-GOV-02](./SPEC-GOV-02-Quality_Gates.md) - 指标采集

## 10. 验收标准 (Acceptance Criteria)

1. **可用性计算**: 能准确计算月度滚动可用性
2. **SLA 监控**: 能实时显示 SLA 状态和剩余预算
3. **事故检测**: 能自动检测和记录系统故障
4. **告警触发**: 超过阈值时能触发正确的告警级别
5. **报告生成**: 能生成月度 SLA 报告
6. **性能降级**: 能正确计算性能降级的等效停机时间
