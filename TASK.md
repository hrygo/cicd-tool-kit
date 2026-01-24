# dev-a 任务卡片

**开发者**: dev-a (Core Platform Engineer)
**技术栈**: Go 1.21
**命名空间**: `pkg/runner/`, `pkg/platform/`, `pkg/config/`

---

## 当前任务

### 任务: CORE-01 - Runner Architecture & Lifecycle

- **状态**: 🚧 In Progress
- **优先级**: P0
- **Phase**: Phase 2
- **依赖**: CONF-01 ✅, SKILL-01 ✅
- **预估**: 2 人周

### 任务描述

Runner 是 `cicd-ai-toolkit` 的核心执行引擎，负责编排 CI/CD 流程、管理 Claude 子进程、处理上下文注入以及与外部平台交互。它是一个不包含 AI 逻辑的 Go 二进制程序。

### 核心职责

1. **进程管理**: 启动、监控、终止 `claude` CLI 子进程
2. **IO 重定向**: 接管 Stdin/Stdout/Stderr 以实现上下文注入和结果捕获
3. **生命周期**: 处理 Init, Execute, Cleanup 阶段
4. **信号处理**: 优雅退出 (Graceful Shutdown)

### 交付物

| 交付物 | 描述 | 状态 |
|--------|------|------|
| **启动流程** | Config Load → Platform Init → Workspace Prep | ⏳ |
| **进程管理** | os/exec 启动 claude，管道 IO | ⏳ |
| **故障恢复** | MaxRetries=3，指数退避 (1s, 2s, 4s) | ⏳ |
| **退出代码** | 0/1/2/101/102 语义定义 | ⏳ |
| **冷启动优化** | < 5s 启动时间 | ⏳ |

### 关键设计要点

#### 启动流程 (Bootstrap)
```go
// 1. Config Load: 读取环境变量和 .cicd-ai-toolkit.yaml
// 2. Platform Init: 根据 GITHUB_ACTIONS/GITEE_GO 初始化适配器
// 3. Workspace Prep: 校验 Git 仓库，检查 CLAUDE.md
```

#### 进程管理
```go
type ClaudeProcess struct {
    Cmd       *exec.Cmd
    Stdin     io.WriteCloser
    Stdout    io.ReadCloser
    WaitGroup sync.WaitGroup
}

// Command: claude -p --dangerously-skip-permissions [--json-schema]
```

#### 退出代码
- `0`: Success (分析完成)
- `1`: Infrastructure Error (网络、配置)
- `2`: Claude Error (API 配额、超载)
- `101`: Timeout
- `102`: Resource Limit Exceeded

#### 冷启动时间预算 (< 5s)
| 阶段 | 目标 | 策略 |
|------|------|------|
| 配置加载 | < 500ms | 延迟加载、缓存 |
| 平台初始化 | < 500ms | 懒加载适配器 |
| 技能发现 | < 1000ms | 索引缓存、并行扫描 |
| Claude 启动 | < 2000ms | 进程池、预热 |
| 准备完成 | < 1000ms | 并行初始化 |

### 目录结构

```
pkg/runner/
├── lifecycle.go       # 启动、停止、清理
├── process.go         # Claude 进程管理
├── io.go              # Stdin/Stdout 重定向
└── watchdog.go        # 故障恢复
```

### 验收标准

- [ ] 能启动 claude 子进程并建立 IO 管道
- [ ] 正确处理 SIGTERM/SIGINT 信号，优雅退出
- [ ] 故障重试机制正常工作 (3次，指数退避)
- [ ] 冷启动时间 < 5s
- [ ] 退出代码符合规范
- [ ] 单元测试覆盖率 > 80%

### 相关文件

- Spec 文档: `../../specs/SPEC-CORE-01-Runner_Lifecycle.md`
- 依赖 Spec: `../../specs/SPEC-CONF-01-Configuration.md`
- 依赖 Spec: `../../specs/SPEC-SKILL-01-Skill_Definition.md`

---

## 已完成任务

| Spec ID | 名称 | 完成日期 | PR |
|---------|------|----------|-----|
| PLAT-07 | Project Structure | 2026-01-25 | - |
| CONF-01 | Configuration | 2026-01-25 | - |

---

## 工作区信息

- **当前 Worktree**: `/Users/huangzhonghui/.worktree/pr-a-CORE-01`
- **当前分支**: `pr-a-CORE-01`
- **锁定文件**: `runner`

---

## 开发命令

```bash
# 运行测试
make test

# 运行特定包测试
go test ./pkg/runner/... -v -race

# 构建
make build

# Lint
make lint
```

---

## 进度日志

| 日期 | 操作 | 状态 |
|------|------|------|
| 2026-01-25 | 分配 CORE-01 任务 | ✅ |
