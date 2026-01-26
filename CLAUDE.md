# CICD AI Toolkit - 项目上下文

> **渐进式披露**：本文件为 AI 助手提供项目上下文。信息按优先级分层，避免一次性加载过多细节。

---

## 快速概览 (Quick Overview)

**项目**: CICD AI Toolkit
**语言**: Go 1.21+
**定位**: 基于 Claude Code 的可插拔 CI/CD 工具集
**状态**: 生产就绪 (32/32 Specs 完成)

### 核心组件

| 组件 | 位置 | 职责 |
|------|------|------|
| **Runner** | `pkg/runner/` | 核心执行引擎，生命周期管理 |
| **Platform** | `pkg/platform/` | 平台适配器 (GitHub/GitLab/Gitee/Jenkins) |
| **Skill** | `pkg/skill/` | 技能加载器、注册表、注入器 |
| **Config** | `pkg/config/` | 配置系统 (YAML + Env) |
| **BuildCtx** | `pkg/buildctx/` | Git Diff 分析、分片、剪枝 |

---

## 第一层：项目结构

```
cicdtools/
├── cmd/cicd-runner/        # CLI 入口
├── pkg/                     # 核心库
│   ├── runner/              # Runner 生命周期
│   ├── platform/            # 平台适配器
│   ├── skill/               # 技能系统
│   ├── config/              # 配置加载
│   ├── buildctx/            # 构建上下文
│   ├── cache/               # 两级缓存
│   ├── security/            # 沙箱、RBAC
│   ├── governance/          # 质量门禁
│   ├── observability/       # 日志、指标、追踪
│   └── ...
├── skills/                  # 内置技能定义
│   ├── code-reviewer/       # 代码审查
│   ├── test-generator/      # 测试生成
│   ├── change-analyzer/     # 变更分析
│   ├── log-analyzer/        # 日志分析
│   ├── issue-triage/        # Issue 分类
│   └── committer/           # 智能提交
├── docs/                    # 文档
├── configs/                 # 配置示例
├── actions/                 # GitHub Actions
└── examples/                # 使用示例
```

---

## 第二层：核心概念

### Runner 生命周期

```
Uninitialized → Initializing → Ready → Running → ShuttingDown → Stopped
```

**关键路径**:
1. `Bootstrap()` - 并行加载配置、扫描技能、验证工作区
2. `Run()` - 执行技能，支持重试和降级
3. `Shutdown()` - 优雅关闭，清理进程

### Platform 接口

所有平台适配器必须实现：

```go
type Platform interface {
    Name() string
    GetPullRequest(ctx, number) (*PullRequest, error)
    PostComment(ctx, number, body) error
    GetDiff(ctx, number) (string, error)
    GetEvent(ctx) (*Event, error)
    GetFileContent(ctx, path, ref) (string, error)
    ListFiles(ctx, path, ref) ([]string, error)
    CreateStatus(ctx, sha, state, description, context) error
}
```

### Skill 定义

技能由 `SKILL.md` 文件定义，采用 YAML frontmatter + Markdown 格式：

```yaml
---
name: "code-reviewer"
version: "1.0.0"
description: "Expert code review"
options:
  temperature: 0.2
tools:
  allow: ["read", "grep", "ls"]
inputs:
  - name: diff
    type: string
---
```

---

## 第三层：关键技术决策

### 幂等性 (CONF-02)

- 使用 `SHA256(diff + config + skill)` 作为指纹
- 相同输入必产生相同输出
- 结果缓存在 `.cicd-ai-cache/`

### 分片策略 (CORE-02)

- 目标：单次 Claude 调用 < 24000 tokens
- 策略：按文件/模块分片，并行分析后汇总
- 剪枝：自动排除 `vendor/`, `*.lock`, `dist/` 等

### 降级机制 (CORE-01)

- Claude 不可用时：跳过，不阻塞 CI
- 超时：可配置，默认 5 分钟
- 重试：指数退避，最多 3 次

### 安全模型 (SEC-01/02/03)

- 沙箱：只读根文件系统
- 注入防护：Prompt 消毒
- RBAC：基于 OPA 策略

---

## 第四层：开发规范

### 代码风格

- **格式**: `go fmt`
- **导入**: 分组标准库、第三方、项目内部
- **错误**: 使用 `%w` 包装，定义 `var ErrXxx = errors.New(...)`
- **日志**: 结构化 JSON 格式

### 提交规范

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**类型**: feat, fix, refactor, perf, docs, test, chore

### 测试要求

- 单元测试覆盖率 > 80%
- 关键路径必须有集成测试
- 使用 table-driven tests

---

## 第五层：故障排查

### 常见问题

| 症状 | 原因 | 解决 |
|------|------|------|
| Bootstrap 失败 | 非 Git 目录 | 在 Git 仓库中运行 |
| Skill 加载失败 | SKILL.md 格式错误 | 检查 YAML 语法 |
| Claude 超时 | Token 超限 | 启用分片 |
| 平台 API 错误 | Token 过期 | 更新环境变量 |

### 调试技巧

```bash
# 启用详细日志
cicd-runner run -v

# 检查配置
cicd-runner skill validate ./skills/code-reviewer

# 测试单个技能
cicd-runner run -s code-reviewer --dry-run
```

---

## 第六层：扩展指南

### 添加新技能

1. 创建 `skills/<name>/SKILL.md`
2. 定义输入/输出 schema
3. 编写 prompt 指令
4. 本地测试：`cicd-runner skill validate`

### 添加新平台

1. 在 `pkg/platform/` 创建 `<platform>.go`
2. 实现 `Platform` 接口
3. 在 `registry.go` 注册
4. 添加平台检测逻辑

### 自定义质量门禁

编辑 `.cicd-ai-toolkit/gates.yaml`:

```yaml
quality_gates:
  - name: "critical-security"
    conditions:
      - category: "security"
        severity: ["critical", "high"]
    action: "block_merge"
```

---

## 附录：相关文档

- [PRD](docs/PRD.md) - 产品需求文档
- [架构概览](docs/architecture/overview.md) - 详细架构设计
- [Specs](docs/specs/) - 32 个技术规范
- [贡献指南](docs/development/contributing.md) - 贡献流程
