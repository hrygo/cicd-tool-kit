# dev-a 任务卡片

**开发者**: dev-a (Core Platform Engineer)
**技术栈**: Go
**命名空间**: `pkg/runner/`, `pkg/platform/`, `pkg/config/`

---

## 当前任务

### 任务: CORE-01 - Runner Lifecycle

- **状态**: ✅ Ready for Review
- **优先级**: P0
- **Phase**: Phase 2
- **依赖**: CONF-01 ✅, SKILL-01 ✅
- **预估**: 1-2 人周

### 任务描述

Runner 是 `cicd-ai-toolkit` 的核心执行引擎，负责编排 CI/CD 流程、管理 Claude 子进程、处理上下文注入以及与外部平台交互。

### 核心职责

1. **进程管理**: 启动、监控、终止 `claude` CLI 子进程
2. **IO 重定向**: 接管 Stdin/Stdout/Stderr 以实现上下文注入和结果捕获
3. **生命周期**: 处理 Init, Execute, Cleanup 阶段
4. **信号处理**: 优雅退出 (Graceful Shutdown)
5. **故障恢复**: Watchdog 机制，支持重试
6. **冷启动优化**: 启动时间 < 5s

### 交付物

| 组件 | 描述 | 状态 |
|------|------|------|
| **Bootstrap** | 配置加载、平台初始化、工作区准备 | ✅ |
| **ProcessManager** | Claude 进程启动、监控、终止 | ✅ |
| **IOHandler** | Stdin/Stdout/Stderr 重定向 | ✅ |
| **Watchdog** | 重试机制、退避策略 | ✅ |
| **SignalHandler** | SIGINT/SIGTERM 优雅处理 | ✅ |
| **Fallback** | Claude API 降级策略 | ✅ |

### 验收标准

- [x] 运行 `cicd-runner --skill review` 能成功拉起 claude 进程并捕获输出
- [x] 超时场景：Claude 挂起超过 timeout，Runner 发送 SIGKILL 并输出 "Execution timed out"
- [x] 信号处理：SIGINT (Ctrl+C) 后，Runner 等待 Claude 清理（< 5s）后退出
- [x] 冷启动：有缓存场景 < 2s，首次启动 < 5s
- [x] 降级策略：Claude API 不可用时跳过，不阻塞 CI
- [x] 重试机制：指数退避，最多 3 次

### 相关文件

- Spec 文档: `../../specs/SPEC-CORE-01-Runner_Lifecycle.md`
- 依赖 Spec: `../../specs/SPEC-CONF-01-Configuration.md`
- 依赖 Spec: `../../specs/SPEC-SKILL-01-Skill_Definition.md`

---

## 工作区信息

- **当前 Worktree**: `~/.worktree/pr-a-CORE-01`
- **当前分支**: `pr-a-CORE-01`
- **锁定文件**: `runner`

---

## 进度日志

| 日期 | 操作 | 状态 |
|------|------|------|
| 2026-01-25 | 分配 CORE-01 任务 | ✅ |
| 2026-01-26 | 实现 Runner Lifecycle 全部组件 | ✅ |

---

## 实现摘要

### 新增文件

| 文件 | 描述 |
|------|------|
| `pkg/runner/errors.go` | Exit codes、错误定义、ClaudeError 类型 |
| `pkg/runner/lifecycle.go` | Runner 核心生命周期管理、Bootstrap、Run、Shutdown |
| `pkg/runner/process.go` | ClaudeProcess、ProcessManager、ProcessPool |
| `pkg/runner/watcher.go` | Watchdog、RetryExecutor、指数退避策略 |
| `pkg/runner/fallback.go` | FallbackHandler、错误分类、降级策略 |
| `pkg/runner/executor.go` | Skill 执行器 |
| `pkg/runner/*_test.go` | 单元测试 (30+ test cases, 39.5% coverage) |

### 关键特性

1. **并行 Bootstrap**: 配置加载、平台检测、技能扫描并行执行
2. **Claude 进程管理**: 完整的进程生命周期 (Start/Stop/Kill/Wait)
3. **IO 重定向**: 通过管道捕获 stdin/stdout/stderr
4. **信号处理**: SIGINT/SIGTERM 触发优雅关闭
5. **指数退避重试**: 默认 1s/2s/4s，最多 3 次
6. **错误分类**: 自动识别 TIMEOUT/RATE_LIMITED/UNAUTHORIZED/SERVER_ERROR 等
7. **降级策略**: Skip/Cache/Partial/Fail 多种降级模式
