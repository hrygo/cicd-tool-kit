# SPEC-PERF-01: Two-Level Caching Strategy

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24

## 1. 概述 (Overview)
Token 是昂贵的，且 LLM 响应较慢。对于代码未变更、Skill 未变更的场景，重复运行 AI 分析是极大的浪费。本 Spec 定义了基于内容的去重缓存策略。

## 2. 核心职责 (Core Responsibilities)
- **Determinism**: 确保同样的输入产生同样的 Cache Key。
- **Store Interface**: 支持内存和文件系统存储。
- **Hit/Miss Logic**: 缓存决策树。

## 3. 详细设计 (Detailed Design)

### 3.1 缓存键生成 (Key Generation)
Cache Key 必须包含所有影响输出的变量：

`Key = SHA256(FileContent + SkillPrompt + Config + ModelVersion)`

*   **FileContent**: 目标代码块的内容。
*   **SkillPrompt**: `SKILL.md` 的完整内容（如果修改 Prompt，缓存应失效）。
*   **Config**: 如 Token Limit, Temperature。
*   **ModelVersion**: e.g., "claude-3-5-sonnet-20241022"。

### 3.2 存储层 (Storage Layer)

#### Level 1: Memory (In-Process)
*   `map[string]AnalysisResult`
*   用途: 类似于 "Memoization"，防止单次运行中对同一段代码（如果被多次引用）重复分析。

#### Level 2: Filesystem (Persistent)
*   Path: `.cicd-ai-cache/sha256_prefix/full_hash.json`
*   Integration: 依赖 CI 系统（如 GitHub Actions Cache）保存和恢复此目录。

### 3.3 流程 (Workflow)
1.  **CalcHash**: 计算当前 Task 的 Key。
2.  **LookUp**: 检查 `.cicd-ai-cache/{Key}` 是否存在。
3.  **Hit**:
    *   读取 JSON。
    *   Log: "Cache Hit for {File}".
    *   Return Result.
4.  **Miss**:
    *   Call Claude.
    *   Write JSON to `.cicd-ai-cache/{Key}`.
    *   Return Result.

## 4. 依赖关系 (Dependencies)
- **Deps**: 无外部依赖，纯 Go 逻辑。
- **Related**: [SPEC-CONF-02](./SPEC-CONF-02-Idempotency.md) - 幂等性与可复现性缓存。

### 3.4 缓存决策树 (Cache Decision Tree)

```
┌─────────────────────────────────────────────────────────────────┐
│                    Cache Decision Flow                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐     │
│  │  Calc Hash   │ -> │  Check L1    │ -> │  Check L2    │     │
│  │  (SHA256)    │    │  (Memory)    │    │  (File)      │     │
│  └──────────────┘    └──────────────┘    └──────────────┘     │
│                                │                    │          │
│                          Hit /  \ Hit          Hit /  \ Hit    │
│                           /       \                /       \    │
│                      Return     Check        Return    Call    │
│                      Result     L2           Result    Claude   │
│                                                              │
└─────────────────────────────────────────────────────────────────┘
```

### 3.5 缓存配置 (Cache Configuration)

```yaml
# .cicd-ai-toolkit.yaml
cache:
  enabled: true

  # Level 1: 内存缓存
  memory:
    enabled: true
    max_entries: 1000     # 最多缓存 1000 个结果
    ttl: "1h"             # 内存缓存过期时间

  # Level 2: 文件系统缓存
  filesystem:
    enabled: true
    path: ".cicd-ai-cache"
    max_size: "1GB"       # 最大缓存大小
    compression: true     # 启用 gzip 压缩

  # 缓存策略
  strategy:
    # 缓存模式: aggressive | standard | conservative
    mode: "standard"

    # aggressive: 缓存更多，可能使用过期结果
    # standard: 平衡模式（默认）
    # conservative: 更严格的缓存验证

  # 缓存失效规则
  invalidation:
    # 文件变更后失效
    on_file_change: true

    # Skill 变更后失效
    on_skill_change: true

    # 配置变更后失效
    on_config_change: true

    # 模型版本变更后失效
    on_model_change: true
```

### 3.6 缓存指标 (Cache Metrics)

```go
// CacheMetrics 缓存指标
type CacheMetrics struct {
    // 命中统计
    L1Hits      int64  // 内存缓存命中次数
    L2Hits      int64  // 文件缓存命中次数
    Misses      int64  // 缓存未命中次数

    // 时间统计
    AvgHitTime  time.Duration  // 平均命中时间
    AvgMissTime time.Duration  // 平均未命中时间

    // 大小统计
    L1Size      int64  // 内存缓存占用
    L2Size      int64  // 文件缓存占用

    // 命中率
    HitRate     float64 // 总命中率
}

// GetHitRate 计算命中率
func (m *CacheMetrics) GetHitRate() float64 {
    total := m.L1Hits + m.L2Hits + m.Misses
    if total == 0 {
        return 0
    }
    return float64(m.L1Hits+m.L2Hits) / float64(total)
}
```

## 4. 60s 分析耗时保证机制

**Covers**: PRD 5.1 - 单次分析耗时 < 60s (中等 PR，P90)

### 4.1 目标定义

| 指标 | 目标值 | 测量方法 |
|------|--------|----------|
| **P90 分析耗时** | < 60s | 90% 的 PR 分析在 60s 内完成 |
| **P50 分析耗时** | < 30s | 中等 PR (100-500 行变更) |
| **缓存命中耗时** | < 1s | 完全缓存命中场景 |

### 4.2 超时分级策略

```go
// TimeoutStrategy 超时策略
type TimeoutStrategy struct {
    // 总超时时间 (包含缓存、API、处理)
    TotalTimeout time.Duration // 60s

    // Claude API 超时
    APITimeout   time.Duration // 45s (留 15s 余量给其他操作)

    // 单个 Chunk 超时
    ChunkTimeout time.Duration // 30s

    // 降级阈值
    DegradationThreshold time.Duration // 50s 触发降级
}

// DefaultTimeoutStrategy 默认超时策略
var DefaultTimeoutStrategy = &TimeoutStrategy{
    TotalTimeout:          60 * time.Second,
    APITimeout:            45 * time.Second,
    ChunkTimeout:          30 * time.Second,
    DegradationThreshold:  50 * time.Second,
}
```

### 4.3 超时降级策略

当分析时间接近或超过阈值时，系统自动降级：

```go
// DegradationLevel 降级级别
type DegradationLevel int

const (
    NoDegradation   DegradationLevel = 0 // 完整分析
    FastAnalysis    DegradationLevel = 1 // 快速分析
    CacheOnly       DegradationLevel = 2 // 仅使用缓存
    PartialResult   DegradationLevel = 3 // 返回部分结果
)

// TimeoutHandler 超时处理器
type TimeoutHandler struct {
    strategy *TimeoutStrategy
}

// HandleTimeout 处理超时
func (h *TimeoutHandler) HandleTimeout(ctx context.Context, elapsed time.Duration) DegradationLevel {
    remaining := h.strategy.TotalTimeout - elapsed

    switch {
    case remaining > 30 * time.Second:
        // 充裕时间，完整分析
        return NoDegradation

    case remaining > 15 * time.Second:
        // 时间紧张，快速分析
        log.Warn("Time budget tight, enabling fast analysis")
        return FastAnalysis

    case remaining > 5 * time.Second:
        // 仅使用缓存
        log.Warn("Time critical, cache-only mode")
        return CacheOnly

    default:
        // 返回部分结果
        log.Error("Timeout exceeded, returning partial results")
        return PartialResult
    }
}
```

### 4.4 快速分析模式

```go
// FastAnalysisConfig 快速分析配置
type FastAnalysisConfig struct {
    // 减少 Chunk 大小
    MaxChunkTokens int    // 默认 32000 -> 16000

    // 跳过可选步骤
    SkipLint     bool   // 跳过 Lint 集成
    SkipExamples  bool   // 跳过示例生成

    // 使用更快的模型
    UseFastModel  bool   // 切换到 haiku 模型

    // 降低分析深度
    AnalysisDepth string // "deep" -> "standard"
}

// ApplyFastAnalysis 应用快速分析
func (h *TimeoutHandler) ApplyFastAnalysis(req *AnalysisRequest) *AnalysisRequest {
    fastReq := req.Clone()
    fastReq.MaxTokens = 16000
    fastReq.SkipLint = true
    fastReq.Model = "haiku"
    fastReq.AnalysisDepth = "standard"
    return fastReq
}
```

### 4.5 Chunk Token 限制

为满足 60s 目标，需限制单次 Chunk 的大小：

```go
// ChunkSizeCalculator Chunk 大小计算器
type ChunkSizeCalculator struct {
    // 目标时间
    TargetTime      time.Duration // 30s per chunk

    // Token 处理速度 (保守估计)
    TokensPerSecond int           // 1000 tokens/s

    // 安全系数
    SafetyFactor    float64       // 0.8 (留 20% 余量)
}

// CalculateMaxTokens 计算最大 Token 数
func (c *ChunkSizeCalculator) CalculateMaxTokens() int {
    rawTokens := int(c.TargetTime.Seconds()) * c.TokensPerSecond
    safeTokens := int(float64(rawTokens) * c.SafetyFactor)
    return safeTokens
}

// 默认配置：30s * 1000 * 0.8 = 24000 tokens
var DefaultChunkCalculator = &ChunkSizeCalculator{
    TargetTime:      30 * time.Second,
    TokensPerSecond: 1000,
    SafetyFactor:    0.8,
}
// 结果: MAX_CHUNK_TOKENS = 24000
```

### 4.6 并行处理策略

对于多个 Chunk，使用并行处理加速：

```go
// ParallelChunkProcessor 并行 Chunk 处理器
type ParallelChunkProcessor struct {
    maxConcurrency int           // 最大并发数
    timeout        time.Duration // 每个 Chunk 的超时
}

// Process 并行处理多个 Chunks
func (p *ParallelChunkProcessor) Process(ctx context.Context, chunks []*Chunk) []*AnalysisResult {
    results := make([]*AnalysisResult, len(chunks))

    // 创建信号量控制并发
    sem := make(chan struct{}, p.maxConcurrency)
    var wg sync.WaitGroup

    for i, chunk := range chunks {
        wg.Add(1)
        go func(idx int, c *Chunk) {
            defer wg.Done()
            sem <- struct{}{}        // 获取信号量
            defer func() { <-sem }() // 释放信号量

            // 为每个 Chunk 设置超时
            chunkCtx, cancel := context.WithTimeout(ctx, p.timeout)
            defer cancel()

            results[idx] = p.processChunk(chunkCtx, c)
        }(i, chunk)
    }

    wg.Wait()
    return results
}
```

### 4.7 P90 监控与告警

```go
// LatencyMonitor 延迟监控器
type LatencyMonitor struct {
    samples []time.Duration
    p90     time.Duration
    p95     time.Duration
    p99     time.Duration
}

// Record 记录一次分析耗时
func (m *LatencyMonitor) Record(duration time.Duration) {
    m.samples = append(m.samples, duration)

    // 保持最近 1000 个样本
    if len(m.samples) > 1000 {
        m.samples = m.samples[len(m.samples)-1000:]
    }

    m.calculatePercentiles()
}

// calculatePercentiles 计算百分位数
func (m *LatencyMonitor) calculatePercentiles() {
    sorted := make([]time.Duration, len(m.samples))
    copy(sorted, m.samples)
    sort.Slice(sorted, func(i, j int) bool {
        return sorted[i] < sorted[j]
    })

    n := len(sorted)
    m.p90 = sorted[n*90/100]
    m.p95 = sorted[n*95/100]
    m.p99 = sorted[n*99/100]
}

// CheckP90 检查 P90 是否满足目标
func (m *LatencyMonitor) CheckP90() bool {
    return m.p90 < 60*time.Second
}
```

### 4.8 超时配置示例

```yaml
# .cicd-ai-toolkit.yaml
performance:
  # 超时配置
  timeout:
    total: "60s"        # 总超时
    api: "45s"          # Claude API 超时
    chunk: "30s"        # 单个 Chunk 超时

  # 降级策略
  degradation:
    enabled: true
    threshold: "50s"    # 触发降级的时间阈值

    # 降级级别
    levels:
      fast_analysis:
        enabled: true
        trigger: "30s"  # 剩余 30s 时触发

      cache_only:
        enabled: true
        trigger: "15s"  # 剩余 15s 时触发

  # Chunk 配置
  chunking:
    max_tokens: 24000   # 单个 Chunk 最大 Token 数
    max_concurrency: 3  # 最大并发 Chunk 数

  # 性能监控
  monitoring:
    enabled: true
    track_percentiles: true  # 跟踪 P90, P95, P99
    alert_threshold: "55s"  # 告警阈值
```

## 5. 依赖关系 (Dependencies)

- **Deps**: 无外部依赖，纯 Go 逻辑。
- **Related**: [SPEC-CONF-02](./SPEC-CONF-02-Idempotency.md) - 幂等性与可复现性缓存。
- **Related**: [SPEC-CORE-02](./SPEC-CORE-02-Context_Chunking.md) - Chunk 分片策略。
- **Related**: [SPEC-OPS-01](./SPEC-OPS-01-Observability.md) - 性能监控。

## 6. 验收标准 (Acceptance Criteria)

### 基础缓存功能
1.  **Hit Consistency**:
    *   Run 1: Analyze `main.go` -> Cost 5s, 写入 Cache。
    *   Run 2 (No Change): Analyze `main.go` -> Cost < 50ms, 读取 Cache。
2.  **Invalidation**:
    *   Modify `main.go` 增加一行注释。
    *   Run 3: Cost 5s (Cache Miss)。
3.  **Skill Change**:
    *   Modify `SKILL.md` Prompt。
    *   Run 4: Cost 5s (Cache Miss, even if code is same)。

### 60s 保证
4.  **P90 目标**:
    *   运行 100 次，90% 次在 60s 内完成。
5.  **超时降级**:
    *   模拟 50s 已用时间，应触发快速分析模式。
    *   模拟 58s 已用时间，应触发 cache-only 模式。
6.  **部分结果**:
    *   超过 60s 应返回已分析的部分结果而非失败。
7.  **并行处理**:
    *   3 个 Chunk 应并行处理，总耗时 < 单个 Chunk 的 2 倍。
8.  **监控指标**:
    *   `/metrics` 端点应暴露 `cicd_ai_latency_p90_seconds` 指标。
