# SPEC-OPS-01: Observability & Audit

**Version**: 1.1
**Status**: Draft
**Date**: 2026-01-24
**Changelog**:
- v1.1: Added 3.4 Resource Monitoring section for memory/CPU tracking

## 1. 概述 (Overview)
为了满足企业级合规要求（Audit）和运维需求（MTTR），Runner 必须提供全方位的可观测性支持，包括结构化日志、审计追踪和性能指标。

## 2. 核心职责 (Core Responsibilities)
- **Structured Logging**: 提供机器可读的 JSON 日志。
- **Audit Trails**: 记录所有对代码库的修改和敏感操作。
- **Metrics**: 暴露运行时指标 (Prometheus 格式)。
- **Resource Monitoring**: 监控内存、CPU、GCO 等资源使用情况。

## 3. 详细设计 (Detailed Design)

### 3.1 日志标准 (Logging Standard)
所有输出到 stdout/stderr 的日志必须符合以下结构（除非是 CLI 交互模式）：

```json
{
  "level": "info",
  "ts": "2026-01-24T10:00:00Z",
  "caller": "runner/lifecycle.go:42",
  "msg": "Claude process started",
  "trace_id": "req-12345",
  "span_id": "span-67890",
  "component": "process_manager"
}
```

### 3.2 审计日志 (Audit Log)
对于所有**写操作**（修改文件、发表评论、合并 PR），必须记录 Audit Log。
Audit Log 应当单独存储或打上特殊 Tag (`audit=true`)。

*   **Key Fields**:
    *   `actor`: 触发操作的用户 (Actor ID from Platform Context).
    *   `action`: `tool_use/edit_file`, `platform/post_comment`.
    *   `resource`: 被操作的对象（文件路径、PR ID）。
    *   `policy_id`: 授权此操作的 OPA 策略 ID ([SPEC-GOV-01](./SPEC-GOV-01-Policy_As_Code.md)).

### 3.3 指标 (Metrics)
Runner 在短生命周期的 CI 环境中，通常通过 Push Gateway 或输出 Summary 文件的方式暴露指标。

*   **Key Metrics**:
    *   `cicd_ai_duration_seconds`: E2E 耗时。
    *   `cicd_ai_token_usage_total`: Token 消耗量（按 Model 维度）。
    *   `cicd_ai_cache_hits_total`: 缓存命中次数。
    *   `cicd_ai_risk_score`: 计算出的风险分值。

### 3.4 资源监控 (Resource Monitoring)

为满足 PRD 5.1 的 **内存占用 <512MB** 要求，Runner 必须实时监控资源使用并在超限时触发告警或降级策略。

#### 3.4.1 内存监控 (Memory Monitoring)

```go
// ResourceMonitor tracks resource usage
type ResourceMonitor struct {
    memLimitBytes   uint64  // 512 * 1024 * 1024 = 536,870,912
    warnThreshold   float64 // 0.8 = 80%
    sampleInterval  time.Duration
    stopChan        chan struct{}
}

// MemoryStats captures current memory usage
type MemoryStats struct {
    Alloc        uint64  // Current allocated bytes
    TotalAlloc   uint64  // Total allocated (cumulative)
    Sys          uint64  // System memory obtained
    NumGC        uint32  // Number of GC cycles
    HeapAlloc    uint64  // Heap allocation
    HeapSys      uint64  // Heap system memory
    HeapObjects  uint64  // Number of heap objects
    StackInuse   uint64  // Stack in-use bytes
    StackSys     uint64  // Stack system bytes
}

// GetMemoryStats reads runtime memory stats
func GetMemoryStats() MemoryStats {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    return MemoryStats{
        Alloc:       m.Alloc,
        TotalAlloc:  m.TotalAlloc,
        Sys:         m.Sys,
        NumGC:       m.NumGC,
        HeapAlloc:   m.HeapAlloc,
        HeapSys:     m.HeapSys,
        HeapObjects: m.HeapObjects,
        StackInuse:  m.StackInuse,
        StackSys:    m.StackSys,
    }
}

// CheckMemoryUsage evaluates if memory is within limits
func (rm *ResourceMonitor) CheckMemoryUsage() ResourceStatus {
    stats := GetMemoryStats()
    usageRatio := float64(stats.Alloc) / float64(rm.memLimitBytes)

    status := ResourceStatus{
        WithinLimits: true,
        Level:        "normal",
        UsageBytes:   stats.Alloc,
        UsagePercent: usageRatio * 100,
    }

    if usageRatio >= 1.0 {
        status.WithinLimits = false
        status.Level = "critical"
        status.Message = fmt.Sprintf("Memory limit exceeded: %d / %d bytes", stats.Alloc, rm.memLimitBytes)
    } else if usageRatio >= rm.warnThreshold {
        status.Level = "warning"
        status.Message = fmt.Sprintf("Memory usage high: %.1f%% (%d / %d bytes)", usageRatio*100, stats.Alloc, rm.memLimitBytes)
    }

    return status
}
```

#### 3.4.2 监控策略 (Monitoring Strategy)

| 阈值 | 动作 |
|------|------|
| **< 80% (410MB)** | 正常运行，记录日志 |
| **80% - 95%** | 警告日志，考虑触发 GC |
| **> 95%** | 触发 `runtime.GC()`，拒绝新任务 |
| **> 100% (512MB)** | 强制退出，返回错误码 102 |

#### 3.4.3 自动 GC 触发 (Automatic GC)

```go
// StartMonitoring begins periodic resource monitoring
func (rm *ResourceMonitor) StartMonitoring(ctx context.Context) {
    ticker := time.NewTicker(rm.sampleInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            status := rm.CheckMemoryUsage()

            // Log status
            log.WithFields(log.Fields{
                "memory_bytes":      status.UsageBytes,
                "memory_percent":    status.UsagePercent,
                "heap_alloc":        status.HeapAlloc,
                "heap_sys":          status.HeapSys,
                "num_gc":            status.NumGC,
                "goroutines":        runtime.NumGoroutine(),
            }).Info("resource_monitor")

            // Trigger GC if warning threshold exceeded
            if status.UsagePercent >= 80 && status.NumGC > 0 {
                log.WithField("before_alloc", status.UsageBytes).Warn("Triggering GC due to high memory")
                runtime.GC()
            }

            // Check critical
            if !status.WithinLimits {
                log.WithField("error", status.Message).Error("Memory limit exceeded")
                return ctx.Err()
            }

        case <-rm.stopChan:
            return
        case <-ctx.Done():
            return
        }
    }
}
```

#### 3.4.4 Prometheus 指标扩展

新增资源相关指标：

```go
var (
    memoryAllocBytes = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "cicd_ai_memory_alloc_bytes",
        Help: "Current memory allocation in bytes",
    })
    memorySysBytes = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "cicd_ai_memory_sys_bytes",
        Help: "System memory obtained in bytes",
    })
    memoryLimitBytes = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "cicd_ai_memory_limit_bytes",
        Help: "Configured memory limit in bytes",
    })
    goroutinesCount = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "cicd_ai_goroutines_count",
        Help: "Current number of goroutines",
    })
    gcCountTotal = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "cicd_ai_gc_count_total",
        Help: "Total number of GC cycles",
    })
    cpuPercent = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "cicd_ai_cpu_percent",
        Help: "CPU usage as percentage",
    })
)
```

#### 3.4.5 配置 (Configuration)

在 `.cicd-ai-toolkit.yaml` 中添加资源限制配置：

```yaml
# Resource Limits
resources:
  memory:
    # Hard limit in bytes (default: 512MB)
    limit_bytes: 536870912
    # Warning threshold (0.0-1.0, default: 0.8)
    warn_threshold: 0.8
    # Enable automatic GC
    auto_gc: true
    # Sampling interval (default: 10s)
    sample_interval: "10s"

  cpu:
    # Max CPU cores to use (0 = unlimited)
    max_cores: 4

  goroutines:
    # Maximum goroutines warning threshold
    max_count: 1000
```

#### 3.4.6 退出码扩展 (Exit Codes)

在 [SPEC-CORE-01](./SPEC-CORE-01-Runner_Lifecycle.md) 的退出码基础上新增：

| Exit Code | 含义 |
|-----------|------|
| `102` | Resource Limit Exceeded (内存超限) |

## 4. 依赖关系 (Dependencies)
- **Lib**: `uber-go/zap` (Logging), `prometheus/client_golang` (Metrics).
- **Related**: [SPEC-CONF-02](./SPEC-CONF-02-Idempotency.md) - 幂等性指标监控。

## 5. 验收标准 (Acceptance Criteria)
1.  **JSON Output**: 运行 `cicd-runner --log-format=json`，输出必须为合法 JSON 序列。
2.  **Trace Correlation**: 日志中必须包含 `trace_id`，且能与 Platform 的 Run ID 关联。
3.  **Audit Check**: 模拟一个 `edit_file` 操作，验证日志中是否包含 "Audit: Actor X modified Resource Y"。
4.  **Memory Monitoring**: 正常运行后，日志中应包含 `memory_bytes` 和 `memory_percent` 字段。
5.  **Memory Limit**: 模拟内存接近 512MB 限制，应触发 Warning 日志并自动调用 GC。
6.  **Memory Exceeded**: 当内存超过 512MB 时，Runner 应返回 exit code 102。
7.  **Goroutine Leak**: 间隔 10s 两次采样 `goroutines` 指标，数量不应持续增长。
8.  **Prometheus Format**: 运行 `cicd-runner --metrics`，应输出 Prometheus 格式的指标。
