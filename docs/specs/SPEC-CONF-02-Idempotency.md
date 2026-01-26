# SPEC-CONF-02: Idempotency & Reproducibility

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24
**Covers**: PRD 5.2 (可靠性要求 - 幂等性)

## 1. 概述 (Overview)

幂等性是分布式系统的关键属性，确保同一操作多次执行产生相同结果。对于 `cicd-ai-toolkit`，幂等性确保：
1. **重复运行**：同一 PR 重新触发 CI，产生相同的分析结果
2. **缓存一致性**：缓存命中时返回与原始分析完全一致的结果
3. **可追溯性**：每次运行都能追溯到唯一的执行记录

本 Spec 定义幂等性策略、去重机制和一致性保证。

## 2. 核心职责 (Core Responsibilities)

- **执行指纹**: 为每次分析生成唯一标识符
- **结果去重**: 检测并复用相同输入的历史结果
- **一致性验证**: 确保缓存结果与实际执行结果一致
- **冲突解决**: 处理参数变化导致的缓存失效

## 3. 详细设计 (Detailed Design)

### 3.1 幂等性定义

| 操作类型 | 幂等性保证 | 实现方式 |
|----------|-----------|----------|
| **代码审查** | 弱幂等性 | 相同 Diff → 相同 Issues |
| **变更分析** | 强幂等性 | 相同 Diff → 完全相同的报告 |
| **测试生成** | 非幂等 | 每次生成可能不同（时间戳、随机数据） |
| **评论发布** | 幂等 | 相同内容不重复发布 |

### 3.2 执行指纹 (Execution Fingerprint)

```go
// ExecutionFingerprint 唯一标识一次分析任务
type ExecutionFingerprint struct {
    // 核心标识
    ProjectID     string            // 仓库标识 (owner/repo)
    CommitSHA    string            // 提交 SHA
    DiffHash     string            // Git Diff 的 SHA256
    SkillHash    string            // SKILL.md 内容的 SHA256
    ConfigHash   string            // 配置文件的 SHA256
    ModelVersion string            // Claude 模型版本

    // 环境标识
    Platform      string            // github/gitlab/gitee
    RunnerVersion string            // cicd-runner 版本

    // 时间戳
    Timestamp     time.Time         // 指纹生成时间
}

// FingerprintOptions 生成指纹的选项
type FingerprintOptions struct {
    IncludeConfig    bool  // 是否包含配置 (默认 true)
    IncludePlatform   bool  // 是否包含平台信息 (默认 false)
    NormalizeDiff     bool  // 是否标准化 Diff (默认 true)
}

// Generate 生成执行指纹
func (ef *ExecutionFingerprint) Generate(ctx *AnalysisContext, opts *FingerprintOptions) error {
    // 1. 项目标识
    ef.ProjectID = fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo)

    // 2. Commit SHA
    ef.CommitSHA = ctx.CommitSHA

    // 3. Diff Hash (标准化后)
    if opts.NormalizeDiff {
        normalizedDiff := ef.normalizeDiff(ctx.RawDiff)
        ef.DiffHash = sha256.Sum256(normalizedDiff)
    } else {
        ef.DiffHash = sha256.Sum256([]byte(ctx.RawDiff))
    }

    // 4. Skill Hash
    if skillContent, err := os.ReadFile(ctx.SkillPath); err == nil {
        ef.SkillHash = sha256.Sum256(skillContent)
    }

    // 5. Config Hash
    if opts.IncludeConfig && ctx.ConfigPath != "" {
        if configContent, err := os.ReadFile(ctx.ConfigPath); err == nil {
            // 移除敏感信息后再计算 hash
            sanitized := ef.sanitizeConfig(configContent)
            ef.ConfigHash = sha256.Sum256(sanitized)
        }
    }

    // 6. 模型版本
    ef.ModelVersion = ctx.ModelVersion

    // 7. 平台信息
    if opts.IncludePlatform {
        ef.Platform = ctx.Platform
        ef.RunnerVersion = ctx.RunnerVersion
    }

    ef.Timestamp = time.Now()
    return nil
}

// String 生成指纹字符串
func (ef *ExecutionFingerprint) String() string {
    // 组合所有关键信息
    parts := []string{
        ef.ProjectID,
        ef.CommitSHA,
        ef.DiffHash,
        ef.SkillHash,
        ef.ConfigHash,
        ef.ModelVersion,
    }

    // 如果包含平台信息
    if ef.Platform != "" {
        parts = append(parts, ef.Platform)
    }

    combined := strings.Join(parts, ":")
    return fmt.Sprintf("fp:%s", sha256.Sum256([]byte(combined)))
}
```

### 3.3 Diff 标准化 (Diff Normalization)

为了提高缓存命中率，需要对 Diff 进行标准化处理：

```go
// DiffNormalizer 标准化 Git Diff
type DiffNormalizer struct {
    rules []NormalizationRule
}

type NormalizationRule struct {
    Name     string
    Apply    func(string) string
}

func NewDiffNormalizer() *DiffNormalizer {
    return &DiffNormalizer{
        rules: []NormalizationRule{
            {
                Name: "remove-timestamps",
                Apply: removeTimestamps,
            },
            {
                Name: "normalize-line-endings",
                Apply: normalizeLineEndings,
            },
            {
                Name: "remove-whitespace",
                Apply: func(diff string) string {
                    // 移除行尾空白，但保留代码缩进
                    lines := strings.Split(diff, "\n")
                    for i, line := range lines {
                        lines[i] = strings.TrimRight(line, " \t")
                    }
                    return strings.Join(lines, "\n")
                },
            },
            {
                Name: "sort-file-lists",
                Apply: func(diff string) string {
                    // 对文件列表排序，确保文件顺序不影响结果
                    // (仅适用于某些分析类型)
                    return diff
                },
            },
        },
    }
}

func (dn *DiffNormalizer) Normalize(diff string) string {
    result := diff
    for _, rule := range dn.rules {
        result = rule.Apply(result)
    }
    return result
}
```

### 3.4 幂等性缓存 (Idempotency Cache)

```go
// IdempotencyCache 幂等性缓存
type IdempotencyCache struct {
    store     ResultStore
    ttl       time.Duration
    validator *ConsistencyValidator
}

// ResultStore 结果存储接口
type ResultStore interface {
    Get(fp string) (*CachedResult, error)
    Put(fp string, result *AnalysisResult) error
    Delete(fp string) error
    List(filter ResultFilter) ([]*CachedResult, error)
}

// CachedResult 缓存的结果
type CachedResult struct {
    Fingerprint    string            `json:"fingerprint"`
    Status        string            `json:"status"`        // success, failure, timeout
    Result        *AnalysisResult   `json:"result"`
    ExecutionTime time.Duration    `json:"execution_time"`
    CachedAt      time.Time         `json:"cached_at"`
    ExpiresAt     time.Time         `json:"expires_at"`
    Metadata      ResultMetadata    `json:"metadata"`
}

// AnalysisResult 分析结果 (核心数据)
type AnalysisResult struct {
    Issues        []Issue           `json:"issues"`
    Summary       string            `json:"summary"`
    RiskScore     int               `json:"risk_score"`
    Labels        []string          `json:"labels"`
    TokenUsage    int               `json:"token_usage"`
}

// ResultMetadata 元数据
type ResultMetadata struct {
    RunnerVersion string            `json:"runner_version"`
    Platform      string            `json:"platform"`
    TriggeredBy   string            `json:"triggered_by"`
    TriggerURL   string            `json:"trigger_url"`
}

// Check 检查是否可以使用缓存结果
func (ic *IdempotencyCache) Check(ctx context.Context, fp *ExecutionFingerprint) (*CachedResult, IdempotencyDecision) {
    decision := IdempotencyDecision{
        Fingerprint: fp.String(),
        Action:      "execute",
        Reason:      "",
    }

    // 1. 查找缓存
    cached, err := ic.store.Get(fp.String())
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            decision.Reason = "no_cached_result"
            return nil, decision
        }
        decision.Reason = fmt.Sprintf("cache_error: %v", err)
        decision.Action = "execute" // 缓存错误时执行
        return nil, decision
    }

    // 2. 检查过期
    if time.Now().After(cached.ExpiresAt) {
        decision.Action = "execute"
        decision.Reason = "cache_expired"
        return nil, decision
    }

    // 3. 验证一致性 (可选)
    if ic.validator != nil {
        if valid, err := ic.validator.Validate(ctx, cached); !valid {
            decision.Action = "execute"
            decision.Reason = fmt.Sprintf("cache_invalid: %v", err)
            return nil, decision
        }
    }

    // 4. 缓存命中
    decision.Action = "use_cache"
    decision.Reason = fmt.Sprintf("cache_hit (age: %s)", time.Since(cached.CachedAt))

    log.Info("Idempotency cache hit",
        zap.String("fingerprint", fp.String()),
        zap.Duration("age", time.Since(cached.CachedAt)),
    )

    return cached, decision
}

// IdempotencyDecision 幂等性决策
type IdempotencyDecision struct {
    Fingerprint string            // 执行指纹
    Action      string            // execute, use_cache, skip
    Reason      string            // 决策原因
    CacheResult *CachedResult    // 缓存结果 (如果命中)
}

// Store 存储执行结果
func (ic *IdempotencyCache) Store(fp *ExecutionFingerprint, result *AnalysisResult, duration time.Duration) error {
    cached := &CachedResult{
        Fingerprint:    fp.String(),
        Status:        "success",
        Result:        result,
        ExecutionTime: duration,
        CachedAt:      time.Now(),
        ExpiresAt:     time.Now().Add(ic.ttl),
        Metadata: ResultMetadata{
            RunnerVersion: version.Version,
        },
    }

    return ic.store.Put(fp.String(), cached)
}
```

### 3.5 一致性验证 (Consistency Validation)

```go
// ConsistencyValidator 一致性验证器
type ConsistencyValidator struct {
    strictMode bool  // 严格模式：缓存结果必须完全匹配
}

// Validate 验证缓存结果是否仍然有效
func (cv *ConsistencyValidator) Validate(ctx context.Context, cached *CachedResult) (bool, error) {
    // 1. 检查外部依赖变化
    if cv.hasExternalDependencyChanges(ctx, cached) {
        return false, nil
    }

    // 2. 检查 Skill 定义变化
    if cv.hasSkillDefinitionChanged(ctx, cached) {
        return false, nil
    }

    // 3. 检查策略变化
    if cv.hasPolicyChanged(ctx, cached) {
        return false, nil
    }

    return true, nil
}

func (cv *ConsistencyValidator) hasExternalDependencyChanges(ctx context.Context, cached *CachedResult) bool {
    // 检查依赖的外部资源是否变化
    // 例如：Jira ticket 状态、CI 配置等

    // 如果 Skill 依赖外部 MCP，需要验证这些资源状态
    // 这里可以扩展具体检查逻辑

    return false
}

func (cv *ConsistencyValidator) hasSkillDefinitionChanged(ctx context.Context, cached *CachedResult) bool {
    // 比较 Skill Hash
    // 如果 SKILL.md 内容变化，缓存失效

    // 这里需要比较当前 Skill Hash 和缓存时的 Skill Hash
    // 可以从 cached.Metadata 中提取原始 Skill Hash

    return false
}

func (cv *ConsistencyValidator) hasPolicyChanged(ctx context.Context, cached *CachedResult) bool {
    // 检查 OPA 策略是否变化
    // 如果影响分析结果的策略文件被修改，缓存失效

    // 这里可以扩展具体的策略检查逻辑

    return false
}
```

### 3.6 去重机制 (Deduplication)

```go
// Deduplicator 去重器
type Deduplicator struct {
    cache    *IdempotencyCache
    lock     sync.Mutex
}

// Deduplicate 检查是否可以复用结果
func (d *Deduplicator) Deduplicate(ctx context.Context, req *AnalysisRequest) (*DeduplicationResult, error) {
    d.lock.Lock()
    defer d.lock.Unlock()

    // 1. 生成执行指纹
    fp := &ExecutionFingerprint{}
    if err := fp.Generate(ctx, &FingerprintOptions{
        IncludeConfig:  true,
        IncludePlatform: false,
        NormalizeDiff:   true,
    }); err != nil {
        return nil, fmt.Errorf("failed to generate fingerprint: %w", err)
    }

    // 2. 检查缓存
    cached, decision := d.cache.Check(ctx, fp)

    result := &DeduplicationResult{
        Fingerprint: fp.String(),
        Decision:    decision,
    }

    if decision.Action == "use_cache" && cached != nil {
        result.FromCache = true
        result.Result = cached.Result
        result.CacheAge = time.Since(cached.CachedAt)

        log.Info("Using cached result",
            zap.String("fingerprint", fp.String()),
            zap.Duration("cache_age", result.CacheAge),
        )
    } else {
        result.FromCache = false
        result.Reason = decision.Reason

        log.Info("Cache miss, executing",
            zap.String("fingerprint", fp.String()),
            zap.String("reason", decision.Reason),
        )
    }

    return result, nil
}

// DeduplicationResult 去重结果
type DeduplicationResult struct {
    Fingerprint string            // 执行指纹
    Decision    IdempotencyDecision
    FromCache   bool              // 是否来自缓存
    Result      *AnalysisResult   // 缓存的结果
    CacheAge   time.Duration     // 缓存年龄
    Reason      string            // 未使用缓存的原因
}
```

### 3.7 CLI 集成

```bash
# 强制重新执行，忽略缓存
cicd-runner analyze --force --no-cache

# 仅检查缓存状态，不执行分析
cicd-runner analyze --check-cache

# 清除特定缓存
cicd-runner cache-clear --fingerprint <fp>

# 清除所有缓存
cicd-runner cache-clear --all

# 查看缓存统计
cicd-runner cache-stats
```

### 3.8 配置选项

```yaml
# .cicd-ai-toolkit.yaml
idempotency:
  enabled: true

  # 缓存策略
  cache:
    enabled: true
    ttl: "168h"              # 缓存有效期 (默认 7 天)
    max_size: "1GB"          # 最大缓存大小
    backend: "file"          # file, redis, s3
    path: ".cicd-ai-cache"  # 本地缓存路径

  # 指纹生成选项
  fingerprint:
    include_config: true      # 配置变更时失效缓存
    include_platform: false   # 平台信息不影响缓存
    normalize_diff: true      # 标准化 Diff 提高命中率

  # 一致性验证
  validation:
    enabled: true
    strict_mode: false       # 严格模式：完全匹配才使用缓存
    check_dependencies: true  # 检查外部依赖变化

  # 幂等性行为
  behavior:
    # 代码审查：相同 diff 必须返回相同结果
    code_review:
      idempotent: true
      ttl: "24h"

    # 变更分析：相同 diff 返回相同报告
    change_analysis:
      idempotent: true
      ttl: "24h"

    # 测试生成：不保证幂等（每次可能不同）
    test_generation:
      idempotent: false

    # 评论发布：相同内容不重复发布
    comment_post:
      idempotent: true
      content_hash: true    # 基于内容哈希去重
```

### 3.9 评论发布幂等性

```go
// CommentPoster 评论发布器（带幂等性）
type CommentPoster struct {
    platform   Platform
    cache      CommentCache
    idempotencyKey string
}

type CommentCache struct {
    store KeyValueStore
    ttl   time.Duration
}

type PostedComment struct {
    PRNumber    int       `json:"pr_number"`
    BodyHash    string    `json:"body_hash"`    // 评论内容的 SHA256
    PostedAt    time.Time `json:"posted_at"`
    CommentID   string    `json:"comment_id"`  // 平台返回的评论 ID
}

// PostComment 发布评论（幂等）
func (cp *CommentPoster) PostComment(ctx context.Context, pr int, body string) error {
    // 1. 计算内容哈希
    bodyHash := sha256.Sum256([]byte(body))
    key := fmt.Sprintf("comment:%d:%s", pr, bodyHash)

    // 2. 检查是否已发布
    if cached, err := cp.cache.Get(key); err == nil {
        log.Info("Comment already posted, skipping",
            zap.Int("pr", pr),
            zap.String("comment_id", cached.CommentID),
        )
        return nil // 幂等：不重复发布
    }

    // 3. 发布评论
    commentID, err := cp.platform.PostComment(ctx, pr, body)
    if err != nil {
        return err
    }

    // 4. 记录到缓存
    cp.cache.Put(key, &PostedComment{
        PRNumber:  pr,
        BodyHash:   bodyHash,
        PostedAt:   time.Now(),
        CommentID:  commentID,
    })

    return nil
}
```

## 4. 幂等性矩阵

| 操作 | 输入 | 幂等性保证 | 实现方式 |
|------|------|-----------|----------|
| **代码审查** | Diff + Config | 强幂等 | DiffHash + SkillHash |
| **变更分析** | Diff + Config | 强幂等 | DiffHash + SkillHash |
| **测试生成** | Diff + Config | 无幂等 | 每次执行不同 |
| **日志分析** | Log Content + Config | 强幂等 | LogHash + ConfigHash |
| **安全扫描** | Diff + Tools | 强幂等 | DiffHash + ToolVersions |
| **评论发布** | PR + Content | 强幂等 | PR + ContentHash |
| **状态更新** | PR + Status | 弱幂等 | 多次更新最终状态一致 |
| **标签应用** | PR + Labels | 强幂等 | 去重已有标签 |

## 5. 依赖关系 (Dependencies)

- **Related**: [SPEC-PERF-01](./SPEC-PERF-01-Caching.md) - 两级缓存策略
- **Related**: [SPEC-CONF-01](./SPEC-CONF-01-Configuration.md) - 配置系统
- **Related**: [SPEC-OPS-01](./SPEC-OPS-01-Observability.md) - 幂等性指标监控

## 6. 验收标准 (Acceptance Criteria)

1. **基本幂等**: 对相同 PR 运行两次代码审查，第二次应返回与第一次完全相同的结果
2. **缓存命中**: 相同 Diff 第二次运行应从缓存读取（< 100ms）
3. **配置失效**: 修改配置文件后，缓存应自动失效
4. **Diff 标准化**: 相同内容但格式不同的 Diff 应被视为相同输入
5. **评论去重**: 尝试发布相同内容的评论，第二次应跳过
6. **强制重新执行**: `--force` 标志应绕过缓存强制执行
7. **缓存统计**: `cache-stats` 命令应显示命中率、缓存大小等指标

## 7. 监控指标

```go
// IdempotencyMetrics 幂等性指标
type IdempotencyMetrics struct {
    CacheHitRate      float64  // 缓存命中率
    CacheMissRate     float64  // 缓存未命中率
    CacheErrorRate    float64  // 缓存错误率
    AvgCacheAge       float64  // 平均缓存年龄（小时）
    DuplicateRate      float64  // 重复请求率

    // 按操作类型分解
    CacheHitBySkill   map[string]float64
    ExecutionTimeBySource map[string]time.Duration // cached vs new
}

func RecordIdempotencyDecision(decision *IdempotencyDecision, skill string) {
    metrics.IdempotencyDecisions.WithLabelValues(
        "action", decision.Action,
        "skill", skill,
        "reason", decision.Reason,
    ).Inc()
}
```
