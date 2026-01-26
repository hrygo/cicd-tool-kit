# SPEC-CORE-01: Runner Architecture & Lifecycle

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24

## 1. 概述 (Overview)
Runner 是 `cicd-ai-toolkit` 的核心执行引擎，负责编排 CI/CD 流程、管理 Claude 子进程、处理上下文注入以及与外部平台交互。它是一个不包含 AI 逻辑的 Go 二进制程序。

## 2. 核心职责 (Core Responsibilities)
- **进程管理**: 启动、监控、终止 `claude` CLI 子进程。
- **IO 重定向**: 接管 Stdin/Stdout/Stderr 以实现上下文注入和结果捕获。
- **生命周期**: 处理 Init, Execute, Cleanup 阶段。
- **信号处理**: 优雅退出 (Graceful Shutdown)。

## 3. 详细设计 (Detailed Design)

### 3.1 启动流程 (Bootstrap)
1.  **Config Load**:
    - 读取环境变量 (API Keys, Git Token)。
    - 读取 `.cicd-ai-toolkit.yaml`。
2.  **Platform Init**:
    - 根据 Env (`GITHUB_ACTIONS`, `GITEE_GO`) 初始化对应的 Platform Adapter (见 [SPEC-PLAT-01](./SPEC-PLAT-01-Platform_Adapter.md))。
3.  **Workspace Prep**:
    - 校验当前目录是否为 Git 仓库。
    - 检查 `CLAUDE.md` 是否存在（若无则给出 Warning）。

### 3.2 进程管理 (Process Management)
Runner 通过 `os/exec` 启动 Claude，类似 Supervisor。

*   **Command**: `claude`
*   **Args**:
    *   `-p` (Print/Headless mode)
    *   `--dangerously-skip-permissions` (Required for automation, guarded by [SPEC-SEC-02](./SPEC-SEC-02-Prompt_Injection.md))
    *   `--json-schema` (Optional, 配合 [SPEC-CORE-03](./SPEC-CORE-03-Output_Parsing.md))

```go
type ClaudeProcess struct {
    Cmd       *exec.Cmd
    Stdin     io.WriteCloser
    Stdout    io.ReadCloser
    WaitGroup sync.WaitGroup
}

func (cp *ClaudeProcess) Start(ctx context.Context, args []string) error {
    cp.Cmd = exec.CommandContext(ctx, "claude", args...)
    // Pipe setup...
    return cp.Cmd.Start()
}
```

### 3.3 故障恢复 (Watchdog)
*   **MaxRetries**: 3
*   **Backoff**: Exponential (1s, 2s, 4s)
*   **Condition**: 如果 ExitCode != 0 且不是明确的 "Task Failed"（如网络抖动），则重试。

### 3.4 退出代码 (Exit Codes)
*   `0`: Success (Issues found or not, but analysis completed)
*   `1`: Infrastructure Error (Network, Config)
*   `2`: Claude Error (API quota, Overloaded)
*   `101`: Timeout
*   `102`: Resource Limit Exceeded (see [SPEC-OPS-01](./SPEC-OPS-01-Observability.md))

### 3.5 冷启动优化 (Cold Start Optimization)

根据 PRD 5.1，Runner 冷启动时间必须 < 5s。本节定义启动时间优化策略。

**冷启动定义**: 从 Runner 进程启动到准备好执行第一个 Skill 的时间。

**时间分解**:

| 阶段 | 目标时间 | 优化策略 |
|------|----------|----------|
| **配置加载** | < 500ms | 延迟加载、缓存解析 |
| **平台初始化** | < 500ms | 懒加载适配器 |
| **技能发现** | < 1000ms | 索引缓存、并行扫描 |
| **Claude 进程启动** | < 2000ms | 进程池、预热 |
| **准备完成** | < 1000ms | 并行初始化 |
| **总计** | **< 5000ms** | - |

#### 3.5.1 配置加载优化

```go
type ConfigLoader struct {
    cache      *sync.Map
    parsers    map[string]Parser
}

func (cl *ConfigLoader) Load(path string) (*Config, error) {
    // Check cache first
    if cached, ok := cl.cache.Load(path); ok {
        stat, _ := os.Stat(path)
        if cached.(*cachedConfig).ModTime.Equal(stat.ModTime()) {
            return cached.(*cachedConfig).Config, nil
        }
    }

    // Lazy load sections
    cfg := &Config{}

    // Only parse required sections first
    if err := cl.parseSection(path, "claude", &cfg.Claude); err != nil {
        return nil, err
    }

    // Defer parsing optional sections
    go cl.parseOptionalSections(path, cfg)

    return cfg, nil
}
```

#### 3.5.2 平台适配器懒加载

```go
type PlatformRegistry struct {
    platforms map[string]PlatformFactory
    instances map[string]Platform
    mu        sync.RWMutex
}

func (pr *PlatformRegistry) Get(env map[string]string) (Platform, error) {
    pr.mu.RLock()

    // Detect platform from env
    platformName := pr.detectPlatform(env)

    // Check if already initialized
    if p, ok := pr.instances[platformName]; ok {
        pr.mu.RUnlock()
        return p, nil
    }
    pr.mu.RUnlock()

    // Initialize platform (write lock)
    pr.mu.Lock()
    defer pr.mu.Unlock()

    // Double-check
    if p, ok := pr.instances[platformName]; ok {
        return p, nil
    }

    // Create new instance
    factory := pr.platforms[platformName]
    p, err := factory(env)
    if err != nil {
        return nil, err
    }

    pr.instances[platformName] = p
    return p, nil
}
```

#### 3.5.3 技能发现与缓存

```go
type SkillRegistry struct {
    skills      map[string]*Skill
    index       *SkillIndex
    lastScan    time.Time
    scanDirs    []string
}

type SkillIndex struct {
    byName     map[string]string // name -> path
    byCategory map[string][]string
    modTime    map[string]time.Time
}

func (sr *SkillRegistry) Scan() error {
    start := time.Now()

    // Only scan if index is stale
    if sr.index != nil && time.Since(sr.lastScan) < 5*time.Minute {
        return nil
    }

    // Parallel scan directories
    var mu sync.Mutex
    var wg sync.WaitGroup
    errChan := make(chan error, len(sr.scanDirs))

    for _, dir := range sr.scanDirs {
        wg.Add(1)
        go func(scanDir string) {
            defer wg.Done()

            files, err := filepath.Glob(filepath.Join(scanDir, "*/SKILL.md"))
            if err != nil {
                errChan <- err
                return
            }

            mu.Lock()
            for _, file := range files {
                skill, err := sr.loadSkill(file)
                if err == nil {
                    sr.skills[skill.Name] = skill
                    sr.index.byName[skill.Name] = file
                }
            }
            mu.Unlock()
        }(dir)
    }

    wg.Wait()
    close(errChan)

    sr.lastScan = time.Now()

    // Log scan time
    duration := time.Since(start)
    if duration > 500*time.Millisecond {
        log.Warn("Skill scan took too long", zap.Duration("duration", duration))
    }

    return nil
}
```

#### 3.5.4 Claude 进程预热 (Process Warmup)

```go
type ProcessPool struct {
    warm     bool
    preload  []string // Skills to preload
    timeout  time.Duration
}

func (pp *ProcessPool) Warmup(ctx context.Context) error {
    // Start Claude with minimal session
    cmd := exec.CommandContext(ctx, "claude", "-p", "ping")

    start := time.Now()
    output, err := cmd.CombinedOutput()
    duration := time.Since(start)

    if err != nil {
        return fmt.Errorf("Claude warmup failed: %w", err)
    }

    log.Info("Claude process warmed up", zap.Duration("duration", duration))
    pp.warm = true

    return nil
}
```

#### 3.5.5 启动流程优化

```go
func (r *Runner) Bootstrap(ctx context.Context) error {
    start := time.Now()

    // Phase 1: Quick init (parallel)
    var wg sync.WaitGroup
    errChan := make(chan error, 3)

    // 1.1 Load config (fast path)
    wg.Add(1)
    go func() {
        defer wg.Done()
        cfg, err := r.configLoader.Load(r.configPath)
        if err != nil {
            errChan <- err
            return
        }
        r.config = cfg
    }()

    // 1.2 Detect platform (no init yet)
    wg.Add(1)
    go func() {
        defer wg.Done()
        r.platformName = r.detectPlatform()
    }()

    // 1.3 Scan skills (use cache if available)
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := r.skillRegistry.Scan(); err != nil {
            errChan <- err // Non-fatal
        }
    }()

    wg.Wait()
    close(errChan)

    // Check for critical errors
    for err := range errChan {
        if err != nil {
            return err
        }
    }

    // Phase 2: Platform init (lazy, only when needed)
    // Defer until first use

    // Phase 3: Pre-warm Claude (optional, in background)
    if r.config.PreWarmClaude {
        go r.processPool.Warmup(ctx)
    }

    bootstrapTime := time.Since(start)
    log.Info("Bootstrap complete", zap.Duration("duration", bootstrapTime))

    if bootstrapTime > 3*time.Second {
        log.Warn("Bootstrap took longer than expected",
            zap.Duration("duration", bootstrapTime),
            zap.Duration("target", 3*time.Second))
    }

    return nil
}
```

#### 3.5.6 启动时间监控

```go
type BootstrapMetrics struct {
    StartTime     time.Time
    ConfigLoad    time.Duration
    PlatformInit  time.Duration
    SkillScan     time.Duration
    ClaudeWarmup  time.Duration
    TotalTime     time.Duration
}

func (r *Runner) RecordBootstrapMetrics() {
    metrics := &BootstrapMetrics{
        StartTime:   r.bootstrapStart,
        ConfigLoad:  r.configLoadEnd.Sub(r.bootstrapStart),
        SkillScan:   r.skillScanEnd.Sub(r.configLoadEnd),
        TotalTime:   time.Since(r.bootstrapStart),
    }

    // Report to Prometheus
    bootstrapDuration.WithLabelValues("config_load").Observe(metrics.ConfigLoad.Seconds())
    bootstrapDuration.WithLabelValues("skill_scan").Observe(metrics.SkillScan.Seconds())
    bootstrapDuration.WithLabelValues("total").Observe(metrics.TotalTime.Seconds())

    // Log warning if too slow
    if metrics.TotalTime > 5*time.Second {
        log.Warn("Bootstrap exceeded 5s target",
            zap.Duration("total", metrics.TotalTime),
            zap.Duration("config_load", metrics.ConfigLoad),
            zap.Duration("skill_scan", metrics.SkillScan))
    }
}
```

#### 3.5.7 冷启动验收标准

1. **基本场景**: 在有缓存的场景下，冷启动 < 2s
2. **首次启动**: 无缓存场景，冷启动 < 5s
3. **配置加载**: 配置解析 < 500ms
4. **技能发现**: 技能扫描 < 1s
5. **Claude 启动**: Claude 进程首次响应 < 2s
6. **监控**: 每次启动记录时间指标

## 4. Claude 降级策略 (Fallback Strategy)

**Covers**: PRD 5.2 - 降级策略: Claude 不可用时跳过，不阻塞 CI

### 4.1 降级原则

当 Claude API 不可用时，Runner 应根据错误类型选择适当的降级策略，确保 CI 流程不被阻塞。

### 4.2 错误分类 (Error Classification)

```go
// ClaudeError Claude API 错误类型
type ClaudeError struct {
    Code       string  // 错误代码
    Message    string  // 错误消息
    Retryable  bool    // 是否可重试
    Fallback   FallbackAction // 降级行为
}

// FallbackAction 降级行为
type FallbackAction string

const (
    FallbackRetry      FallbackAction = "retry"       // 重试
    FallbackSkip       FallbackAction = "skip"        // 跳过，不阻塞
    FallbackCache      FallbackAction = "cache"       // 仅使用缓存
    FallbackPartial    FallbackAction = "partial"     // 返回部分结果
    FallbackFail       FallbackAction = "fail"        // 阻塞 CI
)

// ErrorClassifier 错误分类器
type ErrorClassifier struct {
    rules map[string]FallbackAction
}

// Classify 分类错误
func (ec *ErrorClassifier) Classify(err error) *ClaudeError {
    msg := err.Error()

    // 网络超时 - 可重试
    if strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded") {
        return &ClaudeError{
            Code:      "TIMEOUT",
            Message:   msg,
            Retryable: true,
            Fallback:  FallbackRetry,
        }
    }

    // API 限流 - 可重试，使用退避
    if strings.Contains(msg, "rate limit") || strings.Contains(msg, "429") {
        return &ClaudeError{
            Code:      "RATE_LIMITED",
            Message:   msg,
            Retryable: true,
            Fallback:  FallbackRetry,
        }
    }

    // API 密钥无效 - 不重试，跳过
    if strings.Contains(msg, "401") || strings.Contains(msg, "unauthorized") {
        return &ClaudeError{
            Code:      "UNAUTHORIZED",
            Message:   msg,
            Retryable: false,
            Fallback:  FallbackSkip,
        }
    }

    // API 服务器错误 - 可重试
    if strings.Contains(msg, "500") || strings.Contains(msg, "502") || strings.Contains(msg, "503") {
        return &ClaudeError{
            Code:      "SERVER_ERROR",
            Message:   msg,
            Retryable: true,
            Fallback:  FallbackRetry,
        }
    }

    // 内容长度超限 - 不重试，使用降级
    if strings.Contains(msg, "too large") || strings.Contains(msg, "exceeds limit") {
        return &ClaudeError{
            Code:      "CONTENT_TOO_LARGE",
            Message:   msg,
            Retryable: false,
            Fallback:  FallbackPartial,
        }
    }

    // 默认：可重试
    return &ClaudeError{
        Code:      "UNKNOWN",
        Message:   msg,
        Retryable: true,
        Fallback:  FallbackRetry,
    }
}
```

### 4.3 重试策略 (Retry Strategy)

```go
// RetryPolicy 重试策略
type RetryPolicy struct {
    MaxRetries    int           // 最大重试次数
    InitialDelay  time.Duration // 初始延迟
    MaxDelay      time.Duration // 最大延迟
    Multiplier    float64       // 延迟倍数
}

// DefaultRetryPolicy 默认重试策略
var DefaultRetryPolicy = &RetryPolicy{
    MaxRetries:   3,
    InitialDelay: 1 * time.Second,
    MaxDelay:     10 * time.Second,
    Multiplier:   2.0, // 指数退避: 1s, 2s, 4s, 8s
}

// RetryExecutor 重试执行器
type RetryExecutor struct {
    policy *RetryPolicy
}

// Execute 带重试的执行
func (re *RetryExecutor) Execute(ctx context.Context, fn func() error) error {
    var lastErr error
    delay := re.policy.InitialDelay

    for attempt := 0; attempt <= re.policy.MaxRetries; attempt++ {
        if attempt > 0 {
            log.WithFields(log.Fields{
                "attempt": attempt,
                "delay":   delay,
                "error":   lastErr,
            }).Warn("Retrying Claude request")

            select {
            case <-time.After(delay):
            case <-ctx.Done():
                return ctx.Err()
            }

            // 计算下次延迟 (指数退避)
            delay = time.Duration(float64(delay) * re.policy.Multiplier)
            if delay > re.policy.MaxDelay {
                delay = re.policy.MaxDelay
            }
        }

        err := fn()
        if err == nil {
            return nil
        }

        // 分类错误
        classified := &ErrorClassifier{}.Classify(err)

        // 如果不可重试，直接返回
        if !classified.Retryable {
            return err
        }

        lastErr = err
    }

    return fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

### 4.4 降级行为执行

```go
// FallbackHandler 降级处理器
type FallbackHandler struct {
    cache          Cache
    config         *Config
    metrics        *MetricsCollector
}

// Handle 执行降级行为
func (fh *FallbackHandler) Handle(ctx context.Context, err *ClaudeError, req *AnalysisRequest) (*AnalysisResult, error) {
    log.WithFields(log.Fields{
        "error_code": err.Code,
        "fallback":   err.Fallback,
    }).Warn("Executing fallback action")

    switch err.Fallback {
    case FallbackRetry:
        // 由 RetryExecutor 处理
        return nil, err

    case FallbackSkip:
        // 跳过分析，不阻塞 CI
        fh.metrics.RecordFallback("skip")
        return &AnalysisResult{
            Skipped:   true,
            SkipReason: fmt.Sprintf("Claude API unavailable: %s", err.Code),
            Message:   "Analysis skipped due to API unavailability",
        }, nil

    case FallbackCache:
        // 仅使用缓存结果
        fh.metrics.RecordFallback("cache")
        if cached := fh.cache.Get(req.CacheKey()); cached != nil {
            return cached, nil
        }
        // 缓存未命中，跳过
        return &AnalysisResult{
            Skipped:    true,
            SkipReason: "Cache miss during fallback",
            Message:    "No cached result available",
        }, nil

    case FallbackPartial:
        // 返回部分结果（如已分析的 Chunks）
        fh.metrics.RecordFallback("partial")
        return fh.partialResult(ctx, req)

    case FallbackFail:
        // 阻塞 CI
        fh.metrics.RecordFallback("fail")
        return nil, fmt.Errorf("Claude API error (blocking): %w", err)
    }

    return nil, err
}
```

### 4.5 配置驱动的降级策略

```yaml
# .cicd-ai-toolkit.yaml
fallback:
  # 全局降级策略
  mode: "graceful"  # graceful | strict | blocking

  # graceful: Claude 不可用时跳过，不阻塞 CI (默认)
  # strict: Claude 不可用时阻塞 CI
  # blocking: 任何错误都阻塞 CI

  # 重试配置
  retry:
    max_attempts: 3
    initial_delay: "1s"
    max_delay: "10s"
    multiplier: 2.0

  # 错误特定策略
  error_policies:
    - error_code: "TIMEOUT"
      action: "retry"
      max_retries: 3

    - error_code: "RATE_LIMITED"
      action: "retry"
      max_retries: 5
      backoff: "exponential"

    - error_code: "UNAUTHORIZED"
      action: "skip"
      message: "Skipping analysis due to authentication error"

    - error_code: "SERVER_ERROR"
      action: "retry"
      max_retries: 2

    - error_code: "CONTENT_TOO_LARGE"
      action: "partial"
      message: "Content too large, returning partial results"

  # 降级时的缓存策略
  cache_fallback:
    enabled: true
    require_fresh: false  # false: 允许使用过期缓存
    max_age: "168h"        # 缓存最大有效期 7 天

  # 部分结果策略
  partial_result:
    enabled: true
    min_chunks: 1          # 至少返回 1 个 Chunk 的结果
    include_summary: true  # 包含摘要信息
```

### 4.6 降级决策流程图

```
┌─────────────────────────────────────────────────────────────────┐
│                    Fallback Decision Flow                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐                                               │
│  │ Claude Call  │                                               │
│  └──────┬───────┘                                               │
│         │                                                       │
│         v                                                       │
│  ┌──────────────┐    Success                                   │
│  │    Error?    │ ────────────> Return Result                   │
│  └──────┬───────┘                                               │
│         │ Yes                                                   │
│         v                                                       │
│  ┌──────────────┐                                              │
│  │ Classify     │                                              │
│  │ Error Type   │                                              │
│  └──────┬───────┘                                              │
│         │                                                       │
│         v                                                       │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   Error Classification                    │   │
│  ├─────────────────────────────────────────────────────────┤   │
│  │                                                         │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │   │
│  │  │ Retryable?  │  │ Unauthorized│  │   Too Large │     │   │
│  │  │  (Timeout,  │  │   (401)     │  │   (Limit)   │     │   │
│  │  │   5xx)      │  │             │  │             │     │   │
│  │  └─────┬───────┘  └─────┬───────┘  └─────┬───────┘     │   │
│  │        │                │                │             │   │
│  │        v                v                v             │   │
│  │    ┌─────────┐      ┌─────────┐      ┌─────────┐     │   │
│  │    │  Retry  │      │  Skip   │      │ Partial │     │   │
│  │    │<3 times │      │(No block)│      │         │     │   │
│  │    └────┬────┘      └─────────┘      └────┬────┘     │   │
│  │         │                                │             │   │
│  └─────────┼────────────────────────────────┼─────────────┘   │
│            │                                │                     │
│            v                                v                     │
│      ┌──────────┐                    ┌──────────┐              │
│      │ Still    │                    │ Return   │              │
│      │ Failing? │ ─────────Yes───────> Partial  │              │
│      └────┬─────┘                    │ Result   │              │
│           │ No                        └──────────┘              │
│           v                                                     │
│      ┌──────────┐                                              │
│      │ Return   │                                              │
│      │ Success  │                                              │
│      └──────────┘                                              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 4.7 降级监控指标

```go
// FallbackMetrics 降级指标
type FallbackMetrics struct {
    // 总降级次数
    TotalFallbacks int64

    // 按类型统计
    ByAction map[FallbackAction]int64

    // 按错误码统计
    ByErrorCode map[string]int64

    // 降级率
    FallbackRate float64
}

// RecordFallback 记录降级
func (fm *FallbackMetrics) RecordFallback(action FallbackAction) {
    fm.TotalFallbacks++
    if fm.ByAction == nil {
        fm.ByAction = make(map[FallbackAction]int64)
    }
    fm.ByAction[action]++

    // 计算降级率
    fm.FallbackRate = float64(fm.TotalFallbacks) / float64(fm.TotalRequests)
}
```

## 5. 依赖关系 (Dependencies)

- **Upstream**: 被 CI 系统 (GitHub Actions/Gitee Go) 调用。
- **Downstream**:
    - 调用 `claude` CLI 必须在 PATH 中。
    - 依赖 [SPEC-SEC-01](./SPEC-SEC-01-Sandboxing.md) 提供的沙箱环境。
- **Related**: [SPEC-PERF-01](./SPEC-PERF-01-Caching.md) - 降级时使用缓存。

## 5. 验收标准 (Acceptance Criteria)
1.  **Happy Path**: 运行 `cicd-runner --skill review` 能成功拉起 claude 进程，并将其 stdout 捕获并打印。
2.  **Timeout**: 模拟 Claude 挂起超过 `timeout` 配置值，Runner 必须发送 `SIGKILL` 并在 stderr 输出 "Execution timed out"。
3.  **Signal Handling**: 发送 `SIGINT` (Ctrl+C) 给 Runner，Runner 应等待 Claude 子进程清理（Graceful Stop）后再退出，不超过 5s。
