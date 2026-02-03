# 异步架构升级 - 实施指南摘要

**版本**: 1.1.1
**日期**: 2025-02-03
**状态**: 详细规格已完成

---

## 快速概览

本文档是 `docs/specs/cicd_runner_async_arch.md` 的实施摘要，供快速参考。

### 核心改进

| 指标 | 当前 (One-shot) | 目标 (Async) | 改进 |
|------|----------------|--------------|------|
| 首次执行延迟 | ~2s | ~2s | - |
| 后续执行延迟 | ~2s | ~50ms | **40x** |
| 并发能力 | 1 | 10 | **10x** |
| 内存占用 (空闲) | 0 | ~50MB/session | - |

---

## 架构概览

```
                    ┌─────────────────────────────────────┐
                    │        Session Manager              │
                    │     (1:N 会话池管理)                  │
                    └─────────────────────────────────────┘
                                      │
            ┌─────────────────────────┼─────────────────────────┐
            ▼                         ▼                         ▼
     ┌──────────┐            ┌──────────┐            ┌──────────┐
     │Session 1 │            │Session 2 │            │Session N │
     │(PR #123) │            │(PR #456) │            │(...)     │
     └─────┬────┘            └─────┬────┘            └─────┬────┘
           │                      │                      │
           ▼                      ▼                      ▼
     ┌─────────────────────────────────────────────────────────┐
     │              Claude Code CLI Processes                  │
     │  --print --verbose --output-format stream-json           │
     └─────────────────────────────────────────────────────────┘
```

---

## 实施路线图

### Phase 1: 基础会话管理 (1-2周)

**新增文件**:
```
pkg/async/session/
├── manager.go      # SessionManager 接口
├── session.go      # Session 结构体
└── pool.go         # 会话池

pkg/async/registry/
└── uuid.go         # UUID v5 映射
```

**关键代码模式**:

```go
// 会话状态机
type SessionStatus string
const (
    StatusStarting SessionStatus = "starting"
    StatusReady    SessionStatus = "ready"
    StatusBusy     SessionStatus = "busy"
    StatusDead     SessionStatus = "dead"
)

// SessionManager 接口
type Manager interface {
    GetOrCreateSession(ctx context.Context, sessionID string, config SessionConfig) (*Session, error)
    TerminateSession(sessionID string) error
    ListActiveSessions() []SessionMeta
    CleanupIdleSessions(timeout time.Duration) int
}
```

### Phase 2: 双向流式通信 (1-2周)

**新增文件**:
```
pkg/async/stream/
├── protocol.go     # 消息协议定义
├── bidi.go         # 双向流处理器
└── streamer.go     # 流管理器
```

**复用现有组件**:
- `pkg/claude/stream_parser.go` - 扩展为双向
- `pkg/perf/workerpool.go` - 任务队列处理

### Phase 3: Runner 层集成 (1周)

**修改文件**:
```
pkg/runner/
├── async_runner.go     # 新增异步Runner
└── impl.go              # 扩展现有Runner

pkg/ai/
└── brain.go             # 添加 ExecuteAsync() 方法
```

### Phase 4: Webhook 集成 (1周)

**新增文件**:
```
pkg/webhook/
└── async.go             # 异步webhook处理

cmd/cicd-webhook/
├── status.go            # 状态API
└── stream.go            # SSE流式端点 (可选)
```

---

## 可复用组件清单

| 组件 | 路径 | 复用方式 |
|------|------|---------|
| WorkerPool | `pkg/perf/workerpool.go` | 直接复用 |
| SessionPool | `pkg/claude/pool.go` | 扩展生命周期 |
| StreamParser | `pkg/claude/stream_parser.go` | 扩展双向 |
| Event Normalization | `pkg/webhook/` | 直接复用 |

**总计可复用**: ~1700 LOC

---

## 配置变更

### 最小配置 (启用异步)

```yaml
# config.yaml
async:
  enabled: true
  max_sessions: 10
  idle_timeout: 30m

claude:
  output_format: stream-json  # 必需
  verbose: true               # 必需
```

### 环境变量

```bash
ASYNC_ENABLED=true
ASYNC_MAX_SESSIONS=10
ASYNC_IDLE_TIMEOUT=30m
```

---

## 错误处理策略

### 重试策略

```go
type RetryPolicy struct {
    MaxAttempts     int           // 默认: 3
    InitialDelay    time.Duration // 默认: 1s
    MaxDelay        time.Duration // 默认: 30s
    BackoffFactor   float64       // 默认: 2.0
}
```

### 降级层级

1. **Level 0 (Async)**: 完全异步模式
2. **Level 1 (Sync+Session)**: 同步模式但复用会话
3. **Level 2 (One-shot)**: 传统模式

降级触发条件:
- 内存使用 > 1GB
- 队列深度 > 80%
- 错误率 > 10%

---

## 可观测性

### Prometheus 指标

```go
// 会话指标
cicd_async_sessions_active        // Gauge
cicd_async_sessions_total         // Counter
cicd_async_session_duration       // Histogram

// 任务指标
cicd_async_tasks_executed         // Counter
cicd_async_tasks_succeeded        // Counter
cicd_async_tasks_failed           // Counter
cicd_async_task_duration_seconds  // Histogram

// 错误指标
cicd_async_errors_total           // CounterVec
```

### 健康检查端点

```
GET /health
→ {"status": "healthy", "sessions": 5, "queue_depth": 2}

GET /health/ready
→ HTTP 200 if ready, HTTP 503 if not

GET /metrics
→ Prometheus metrics
```

---

## 部署方式

### Systemd (推荐)

```ini
# /etc/systemd/system/cicd-runner@.service
[Unit]
Description=CICD Runner Async Agent (%i)
After=network-online.target

[Service]
Type=notify
ExecStart=/usr/local/bin/cicd-runner async --session-id %%i
Restart=always
RestartSec=5

MemoryMax=1G
CPUQuota=50%

[Install]
WantedBy=multi-user.target
```

启动服务:
```bash
# 启动多个实例
systemctl start cicd-runner@{1..5}

# 启用开机自启
systemctl enable cicd-runner@1
```

---

## 测试检查清单

### Phase 1 验收

- [ ] 可以创建和终止会话
- [ ] 会话在超时后自动清理
- [ ] 单元测试覆盖率 > 80%

### Phase 2 验收

- [ ] 可以向 CLI 发送用户输入
- [ ] 可以实时接收 CLI 输出
- [ ] 支持 cancel 操作

### Phase 3 验收

- [ ] 现有 Review 功能向后兼容
- [ ] 可以切换到异步模式
- [ ] 缓存仍然有效

### Phase 4 验收

- [ ] Webhook 立即返回 202
- [ ] 后台异步执行
- [ ] 可查询执行状态

---

## 开放问题与决策

| 问题 | 影响 | 决策 |
|------|------|------|
| WebSocket 支持? | 低 | Phase 4 可选 |
| 会话磁盘持久化? | 中 | 依赖 CLI 自身 |
| 进程崩溃恢复? | 高 | 自动重启 + 降级 |

---

## 下一步行动

1. **Phase 0**: 评审规格文档，确认架构方向
2. **Phase 1**: 开始实现 SessionManager
3. **持续集成**: 每个 Phase 完成后合并主分支

---

## 参考资料

- **完整规格**: `docs/specs/cicd_runner_async_arch.md`
- **DivineSense 参考**: `/Users/huangzhonghui/divinesense/docs/specs/cc_runner_async_arch.md`
- **最佳实践**: `docs/BEST_PRACTICE_CLI_AGENT.md`
