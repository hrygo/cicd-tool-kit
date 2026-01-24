# SPEC-GOV-02: Quality Gates & Risk Scoring

**Version**: 1.1
**Status**: Draft
**Date**: 2026-01-24
**Changelog**:
- v1.1: Added Section 4 - Metrics Collection & Visualization

## 1. 概述 (Overview)

传统的 CI 门禁通常是静态的（Pass/Fail）。AI 时代的门禁应当是动态的，基于代码变更的"风险分值"来决定是否放行或触发更严格的审查。本 Spec 定义了完整的质量门禁系统，包括风险评分、动态决策、指标采集与可视化展示。

## 2. 核心职责 (Core Responsibilities)

- **Risk Calculation**: 依据修改的文件类型、行数、复杂度计算 Risk Score
- **Gate Decision**: 基于 Risk Score 选择 Pass / Warning / Block / Request Human Review
- **Metrics Collection**: 采集所有关键指标用于分析和展示
- **Visualization**: 通过 Dashboard 和报告展示质量趋势

## 3. 风险评分模型 (Risk Model)

### 3.1 评分算法

Runner 在 Context Chunking 阶段计算 Risk Score (0-100)：

```go
type RiskScorer struct {
    rules []RiskRule
}

type RiskRule struct {
    Name        string
    Condition   func(*ChangeContext) bool
    Score       int
    Category    string // security, performance, stability, complexity
    Reason      string
}

type ChangeContext struct {
    Files       []ChangedFile
    DiffSize    int
    Author      string
    Branch      string
    Language    string
}

type ChangedFile struct {
    Path        string
    Additions   int
    Deletions   int
    IsNew       bool
}

type RiskScore struct {
    Total       int      // 0-100
    Factors     []RiskFactor
    Level       RiskLevel
    Recommendation string
}

type RiskFactor struct {
    Category    string
    Score       int
    Reason      string
    File        string
}

type RiskLevel string

const (
    RiskLevelLow      RiskLevel = "low"      // 0-20
    RiskLevelMedium   RiskLevel = "medium"   // 21-40
    RiskLevelHigh     RiskLevel = "high"     // 41-70
    RiskLevelCritical RiskLevel = "critical" // 71-100
)

func (rs *RiskScorer) Calculate(ctx *ChangeContext) *RiskScore {
    score := 0
    var factors []RiskFactor

    for _, rule := range rs.rules {
        if rule.Condition(ctx) {
            score += rule.Score
            factors = append(factors, RiskFactor{
                Category: rule.Category,
                Score:    rule.Score,
                Reason:   rule.Reason,
            })
        }
    }

    // Cap at 100
    if score > 100 {
        score = 100
    }

    return &RiskScore{
        Total: score,
        Factors: factors,
        Level:   calculateLevel(score),
    }
}
```

### 3.2 内置风险规则

| 规则 | 条件 | 分数 | 类别 |
|------|------|------|------|
| **Critical Path** | 修改 `auth/`, `payment/`, `security/` | +50 | stability |
| **Config Change** | 修改 `*.yaml`, `*.env`, `config/` | +30 | stability |
| **Large Diff** | Diff > 500 行 | +20 | complexity |
| **Database Schema** | 修改 `*schema*.sql`, `migrations/` | +40 | stability |
| **Infrastructure** | 修改 Dockerfile, k8s/, terraform/ | +35 | stability |
| **High Impact Language** | Go/Rust 修改 | +15 | complexity |
| **Test Changes** | 修改测试文件 | -10 | (reduction) |
| **Doc Changes** | 修改 `*.md`, `docs/` | -5 | (reduction) |
| **First-time Contributor** | 新贡献者 | +10 | stability |
| **Hotfix Branch** | 分支名包含 `hotfix/` | +20 | stability |

### 3.3 动态门禁配置 (Configuration)

```yaml
# .cicd-ai-toolkit.yaml
quality_gates:
  # 门禁规则列表 (按优先级排序)
  - name: "Low Risk Auto-Merge"
    condition: "score < 10"
    action: "approve"
    description: "Low risk changes can auto-merge"

  - name: "Medium Risk Warning"
    condition: "score >= 10 && score < 40"
    action: "warning"
    require_checks: ["ci-test"]
    description: "Medium risk: requires CI checks to pass"

  - name: "High Risk Review"
    condition: "score >= 40 && score < 70"
    action: "require_review"
    required_reviewers: 2
    required_skills: ["code-reviewer"]
    description: "High risk: requires 2 reviewer approval"

  - name: "Critical Security Review"
    condition: "score >= 70 || has_security_changes"
    action: "require_security_review"
    required_skills: ["security-scanner", "compliance-check"]
    require_human_override: true
    description: "Critical: requires security team approval"

  # 风险类别权重
  category_weights:
    security: 2.0      # 安全风险权重翻倍
    stability: 1.5     # 稳定性风险权重 1.5x
    performance: 1.0
    complexity: 0.8    # 复杂度风险权重降低
```

### 3.4 结果反馈 (Feedback)

| 风险等级 | Check 状态 | 评论行为 | 自动操作 |
|----------|-----------|----------|----------|
| **Low (< 10)** | ✅ success | 无 | 可自动合并 |
| **Medium (10-40)** | ✅ success | 发布摘要 | 无 |
| **High (40-70)** | ⚠️ pending | 详细风险列表 | 要求审查 |
| **Critical (> 70)** | ❌ failure | @security-team | 阻止合并 |

```go
type GateResult struct {
    Status      GateStatus
    Message     string
    Actions     []RequiredAction
    Metrics     GateMetrics
}

type GateStatus string

const (
    GateStatusApprove     GateStatus = "approve"
    GateStatusWarning     GateStatus = "warning"
    GateStatusRequireReview GateStatus = "require_review"
    GateStatusBlock       GateStatus = "block"
)

type RequiredAction struct {
    Type        string // "review", "check", "approval"
    Description string
    Resource    string
    Completed   bool
}

func (gr *GateResult) FormatComment() string {
    var sb strings.Builder

    sb.WriteString(fmt.Sprintf("## Quality Gate Result: %s\n\n", gr.Status))
    sb.WriteString(fmt.Sprintf("**Risk Score**: %d/100 (%s)\n\n", gr.Metrics.Score, gr.Metrics.Level))

    if len(gr.Metrics.Factors) > 0 {
        sb.WriteString("### Risk Factors\n")
        for _, factor := range gr.Metrics.Factors {
            sb.WriteString(fmt.Sprintf("- **%s**: %s (+%d)\n", factor.Category, factor.Reason, factor.Score))
        }
        sb.WriteString("\n")
    }

    if len(gr.Actions) > 0 {
        sb.WriteString("### Required Actions\n")
        for i, action := range gr.Actions {
            status := "☐"
            if action.Completed {
                status = "☑"
            }
            sb.WriteString(fmt.Sprintf("%s %d. %s\n", status, i+1, action.Description))
        }
    }

    return sb.String()
}
```

## 4. 指标采集与展示 (Metrics Collection & Visualization)

### 4.1 指标定义

基于 PRD 9.3 关键指标，定义以下采集指标：

| 指标 | 类型 | 目标 | 来源 |
|------|------|------|------|
| **Pipeline Success Rate** | Gauge | > 95% | CI/CD 执行结果 |
| **User Acceptance Rate** | Gauge | > 20% | AI 建议采纳率 |
| **False Positive Rate** | Gauge | < 10% | 用户反馈 |
| **Execution Time** | Histogram | < 90s | 运行时统计 |
| **Risk Score Distribution** | Histogram | - | 风险评分统计 |
| **Issue Category Breakdown** | Histogram | - | 问题分类 |
| **Coverage Improvement** | Gauge | +5% | 测试覆盖率变化 |

### 4.2 Prometheus 集成

```go
// Metrics Collector
type MetricsCollector struct {
    registry *prometheus.Registry

    // Pipeline metrics
    pipelineSuccessRate    prometheus.Gauge
    pipelineExecutionTime  prometheus.Histogram
    pipelineRunsTotal      prometheus.Counter

    // Quality metrics
    userAcceptanceRate     prometheus.Gauge
    falsePositiveRate      prometheus.Gauge
    riskScoreDistribution  prometheus.Histogram

    // Issue metrics
    issueCategoryBreakdown *prometheus.HistogramVec
    issueSeverityBreakdown *prometheus.HistogramVec

    // Coverage metrics
    coverageImprovement     prometheus.Gauge

    // Token usage
    tokensConsumed          prometheus.Counter
    tokensCached            prometheus.Counter
}

func NewMetricsCollector() *MetricsCollector {
    mc := &MetricsCollector{
        registry: prometheus.NewRegistry(),
    }

    // Pipeline metrics
    mc.pipelineSuccessRate = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "cicd_ai_pipeline_success_rate",
        Help: "Success rate of AI-powered pipelines (rolling 7-day average)",
    })

    mc.pipelineExecutionTime = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name: "cicd_ai_pipeline_duration_seconds",
        Help: "Pipeline execution duration in seconds",
        Buckets: prometheus.DefBuckets,
    })

    mc.pipelineRunsTotal = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "cicd_ai_pipeline_runs_total",
        Help: "Total number of pipeline runs",
    })

    // Quality metrics
    mc.userAcceptanceRate = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "cicd_ai_user_acceptance_rate",
        Help: "Rate of AI suggestions accepted by users (rolling 30-day)",
    })

    mc.falsePositiveRate = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "cicd_ai_false_positive_rate",
        Help: "Rate of false positive AI findings (rolling 30-day)",
    })

    mc.riskScoreDistribution = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name: "cicd_ai_risk_score",
        Help: "Distribution of calculated risk scores",
        Buckets: []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
    })

    // Issue metrics (labeled by category and severity)
    mc.issueCategoryBreakdown = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "cicd_ai_issues_by_category",
            Help: "Issues found by category",
            Buckets: []float64{1, 5, 10, 20, 50, 100},
        },
        []string{"category"}, // security, performance, logic, style
    )

    mc.issueSeverityBreakdown = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "cicd_ai_issues_by_severity",
            Help: "Issues found by severity",
            Buckets: []float64{1, 5, 10, 20, 50, 100},
        },
        []string{"severity"}, // critical, high, medium, low
    )

    // Coverage metrics
    mc.coverageImprovement = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "cicd_ai_coverage_improvement",
        Help: "Code coverage improvement percentage",
    })

    // Token usage
    mc.tokensConsumed = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "cicd_ai_tokens_consumed_total",
        Help: "Total tokens consumed",
    })

    mc.tokensCached = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "cicd_ai_tokens_cached_total",
        Help: "Total tokens served from cache",
    })

    // Register all metrics
    mc.registry.MustRegister(
        mc.pipelineSuccessRate,
        mc.pipelineExecutionTime,
        mc.pipelineRunsTotal,
        mc.userAcceptanceRate,
        mc.falsePositiveRate,
        mc.riskScoreDistribution,
        mc.issueCategoryBreakdown,
        mc.issueSeverityBreakdown,
        mc.coverageImprovement,
        mc.tokensConsumed,
        mc.tokensCached,
    )

    return mc
}

// Record methods
func (mc *MetricsCollector) RecordPipeline(success bool, duration time.Duration) {
    mc.pipelineRunsTotal.Inc()
    mc.pipelineExecutionTime.Observe(duration.Seconds())

    // Update success rate (simplified - in production use exponential moving average)
    // TODO: Implement proper rolling window calculation
}

func (mc *MetricsCollector) RecordRiskScore(score float64) {
    mc.riskScoreDistribution.Observe(score)
}

func (mc *MetricsCollector) RecordIssues(category string, severity string, count int) {
    mc.issueCategoryBreakdown.WithLabelValues(category).Observe(float64(count))
    mc.issueSeverityBreakdown.WithLabelValues(severity).Observe(float64(count))
}

func (mc *MetricsCollector) RecordTokenUsage(consumed, cached int) {
    mc.tokensConsumed.Add(float64(consumed))
    mc.tokensCached.Add(float64(cached))
}
```

### 4.3 Metrics Endpoint

Runner 提供 `/metrics` 端点供 Prometheus 抓取：

```go
func (mc *MetricsCollector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    promhttp.HandlerFor(mc.registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}

// Start metrics server
func (mc *MetricsCollector) Start(addr string) error {
    http.Handle("/metrics", mc)
    return http.ListenAndServe(addr, nil)
}
```

### 4.4 Push Gateway 支持

对于短生命周期的 CI 环境，支持 Push Gateway：

```go
func (mc *MetricsCollector) PushToGateway(gatewayURL, job string) error {
    return push.New(gatewayURL, job).
        Collector(mc.registry).
        Grouping("instance", os.Getenv("HOSTNAME")).
        Grouping("repo", os.Getenv("GITHUB_REPOSITORY")).
        Push()
}
```

### 4.5 Dashboard 配置 (Grafana)

提供预配置的 Grafana Dashboard JSON：

```json
{
  "dashboard": {
    "title": "cicd-ai-toolkit Quality Dashboard",
    "panels": [
      {
        "title": "Pipeline Success Rate (7d)",
        "targets": [
          {
            "expr": "cicd_ai_pipeline_success_rate"
          }
        ],
        "type": "gauge",
        "fieldConfig": {
          "defaults": {
            "unit": "percentunit",
            "min": 0,
            "max": 1,
            "thresholds": {
              "steps": [
                {"color": "red", "value": 0},
                {"color": "yellow", "value": 0.9},
                {"color": "green", "value": 0.95}
              ]
            }
          }
        }
      },
      {
        "title": "Risk Score Distribution",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, cicd_ai_risk_score_bucket)"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Issues by Category",
        "targets": [
          {
            "expr": "sum(rate(cicd_ai_issues_by_category_bucket[1h])) by (le, category)"
          }
        ],
        "type": "heatmap"
      },
      {
        "title": "User Acceptance Rate",
        "targets": [
          {
            "expr": "cicd_ai_user_acceptance_rate"
          }
        ],
        "type": "gauge"
      },
      {
        "title": "False Positive Rate",
        "targets": [
          {
            "expr": "cicd_ai_false_positive_rate"
          }
        ],
        "type": "gauge"
      },
      {
        "title": "Pipeline Duration (P50, P95, P99)",
        "targets": [
          {
            "expr": "histogram_quantile(0.50, cicd_ai_pipeline_duration_seconds_bucket)",
            "legendFormat": "P50"
          },
          {
            "expr": "histogram_quantile(0.95, cicd_ai_pipeline_duration_seconds_bucket)",
            "legendFormat": "P95"
          },
          {
            "expr": "histogram_quantile(0.99, cicd_ai_pipeline_duration_seconds_bucket)",
            "legendFormat": "P99"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Token Usage Efficiency",
        "targets": [
          {
            "expr": "cicd_ai_tokens_cached_total / (cicd_ai_tokens_consumed_total + cicd_ai_tokens_cached_total)"
          }
        ],
        "type": "gauge"
      }
    ]
  }
}
```

### 4.6 报告生成

生成 Markdown/HTML 格式的质量报告：

```go
type QualityReport struct {
    Period     time.Time     // Report period
    Project    string
    Metrics    ReportMetrics
    Trends     []Trend
    Findings   []Finding
    Actions    []ActionItem
}

type ReportMetrics struct {
    TotalRuns          int
    SuccessRate        float64
    AvgRiskScore       float64
    AvgDuration        time.Duration
    IssuesFound        int
    IssuesByCategory   map[string]int
    IssuesBySeverity   map[string]int
    UserAcceptance     float64
    FalsePositive      float64
    CoverageImprovement float64
}

type Trend struct {
    Metric      string
    Direction   string // "up", "down", "stable"
    Change      float64
    Status      string // "good", "bad", "neutral"
}

type Finding struct {
    Title       string
    Description string
    Severity    string
    Evidence    []string
}

type ActionItem struct {
    Priority    string
    Title       string
    Description string
    Owner       string
    DueDate     time.Time
}

func (qr *QualityReport) GenerateMarkdown() string {
    var sb strings.Builder

    sb.WriteString("# Quality Report\n\n")
    sb.WriteString(fmt.Sprintf("**Period**: %s to %s\n", qr.Period.Format("2006-01-02"), qr.Period.AddDate(0, 0, 7).Format("2006-01-02")))
    sb.WriteString(fmt.Sprintf("**Project**: %s\n\n", qr.Project))

    // Summary
    sb.WriteString("## Summary\n\n")
    sb.WriteString(renderSummaryCard("Pipeline Success Rate", fmt.Sprintf("%.1f%%", qr.Metrics.SuccessRate*100), qr.Metrics.SuccessRate >= 0.95))
    sb.WriteString(renderSummaryCard("Avg Risk Score", fmt.Sprintf("%.0f", qr.Metrics.AvgRiskScore), qr.Metrics.AvgRiskScore < 50))
    sb.WriteString(renderSummaryCard("User Acceptance", fmt.Sprintf("%.1f%%", qr.Metrics.UserAcceptance*100), qr.Metrics.UserAcceptance >= 0.20))
    sb.WriteString(renderSummaryCard("False Positive Rate", fmt.Sprintf("%.1f%%", qr.Metrics.FalsePositive*100), qr.Metrics.FalsePositive < 0.10))
    sb.WriteString("\n")

    // Trends
    sb.WriteString("## Trends\n\n")
    for _, trend := range qr.Trends {
        icon := "➡️"
        if trend.Direction == "up" {
            icon = trend.Status == "good" ? "📈" : "📉"
        } else if trend.Direction == "down" {
            icon = trend.Status == "good" ? "📉" : "📈"
        }
        sb.WriteString(fmt.Sprintf("- %s **%s**: %s %.1f%%\n", icon, trend.Metric, trend.Direction, trend.Change))
    }
    sb.WriteString("\n")

    // Issues breakdown
    sb.WriteString("## Issues Found\n\n")
    sb.WriteString("| Category | Count |\n|----------|-------|\n")
    for cat, count := range qr.Metrics.IssuesByCategory {
        sb.WriteString(fmt.Sprintf("| %s | %d |\n", cat, count))
    }
    sb.WriteString("\n")

    sb.WriteString("| Severity | Count |\n|----------|-------|\n")
    for sev, count := range qr.Metrics.IssuesBySeverity {
        sb.WriteString(fmt.Sprintf("| %s | %d |\n", sev, count))
    }
    sb.WriteString("\n")

    // Key findings
    sb.WriteString("## Key Findings\n\n")
    for _, finding := range qr.Findings {
        sb.WriteString(fmt.Sprintf("### %s\n\n", finding.Title))
        sb.WriteString(fmt.Sprintf("**Severity**: %s\n\n", finding.Severity))
        sb.WriteString(fmt.Sprintf("%s\n\n", finding.Description))
        for _, evidence := range finding.Evidence {
            sb.WriteString(fmt.Sprintf("- %s\n", evidence))
        }
        sb.WriteString("\n")
    }

    // Action items
    sb.WriteString("## Action Items\n\n")
    for i, action := range qr.Actions {
        sb.WriteString(fmt.Sprintf("%d. **[%s]** %s\n", i+1, action.Priority, action.Title))
        sb.WriteString(fmt.Sprintf("   - %s\n", action.Description))
        sb.WriteString(fmt.Sprintf("   - Owner: %s, Due: %s\n\n", action.Owner, action.DueDate.Format("2006-01-02")))
    }

    return sb.String()
}
```

### 4.7 CLI 可视化

提供 `cicd-runner metrics` 命令查看实时指标：

```bash
$ cicd-runner metrics

┌─────────────────────────────────────────────────────────────────┐
│                    cicd-ai-toolkit Metrics                       │
├─────────────────────────────────────────────────────────────────┤
│ Pipeline Success Rate  │ ████████████░░░░ 95.2%                 │
│ User Acceptance       │ ████████░░░░░░░░ 65.8%                 │
│ False Positive Rate   │ ████░░░░░░░░░░░░ 8.3%                   │
│ Cache Hit Rate        │ ████████████████ 85.7%                 │
├─────────────────────────────────────────────────────────────────┤
│ Average Duration      │ 45.3s                                   │
│ Average Risk Score    │ 32.5 / 100                              │
│ Issues Found (24h)    │ 127                                     │
├─────────────────────────────────────────────────────────────────┤
│ Issues by Severity                                               │
│   Critical ████████ 12                                            │
│   High      ██████████████████████████████████ 58                │
│   Medium    ████████████████████ 42                               │
│   Low       ████████████████████████████████████ 67              │
└─────────────────────────────────────────────────────────────────┘
```

### 4.8 配置示例

```yaml
# .cicd-ai-toolkit.yaml
metrics:
  enabled: true
  # Prometheus scrape endpoint
  listen_address: ":9090"
  # Or use Push Gateway
  push_gateway:
    url: "http://prometheus-pushgateway:9091"
    interval: "30s"
    job: "cicd-ai-toolkit"

  # Report generation
  reports:
    # Daily summary
    daily:
      enabled: true
      format: "markdown"
      output: "/var/log/cicd-reports/daily.md"

    # Weekly detailed report
    weekly:
      enabled: true
      format: "html"
      output: "/var/log/cicd-reports/weekly.html"
      send_to:
        - type: "slack"
          webhook: "${SLACK_WEBHOOK}"
        - type: "email"
          recipients: ["team@example.com"]

  # Dashboard integration
  dashboards:
    grafana:
      url: "http://grafana:3000"
      datasource: "prometheus"
      dashboard_id: "cicd-ai-toolkit"
```

## 5. 依赖关系 (Dependencies)

- **Deps**: 依赖 [SPEC-CORE-02](./SPEC-CORE-02-Context_Chunking.md) 分析文件列表
- **Related**: [SPEC-OPS-01](./SPEC-OPS-01-Observability.md) - 指标采集集成
- **Related**: [SPEC-SEC-03](./SPEC-SEC-03-RBAC.md) - 审批权限控制

## 6. 验收标准 (Acceptance Criteria)

1. **Score Accuracy**: 修改 `README.md`，Risk Score 应 < 10。修改 `main.go`，Score 应 > 20
2. **Gate Action**: 当 Score > Threshold 时，Runner 应将 GitHub Check 状态设为 `failure` 或 `neutral`，并明确提示需要人工介入
3. **Metrics Endpoint**: `/metrics` 端点返回正确的 Prometheus 格式
4. **Dashboard**: Grafana Dashboard 能正确显示所有指标
5. **Report Generation**: 能生成包含趋势和行动项的 Markdown 报告
6. **CLI Visualization**: `cicd-runner metrics` 命令显示正确的 ASCII 图表
