# SPEC-PLAT-07: Project Directory Structure

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24
**Covers**: PRD 3.5 - 目录结构设计

## 1. 概述 (Overview)

本 Spec 定义了 `cicd-ai-toolkit` 项目的完整目录结构，包括 Go Runner 代码、Skills 定义、配置文件、文档和 CI/CD 集成。

## 2. 完整目录结构

```
cicd-ai-toolkit/
├── cmd/                                  # Go 可执行文件入口
│   └── cicd-runner/                     # 主命令
│       ├── main.go                      # 入口点
│       ├── root.go                      # Root 命令实现
│       ├── run.go                       # run 命令
│       ├── skill.go                     # skill 子命令
│       └── version.go                   # version 命令
│
├── pkg/                                 # Go 核心库 (可被外部引用)
│   ├── runner/                          # 运行器核心逻辑
│   │   ├── lifecycle.go                 # 启动、停止、生命周期管理
│   │   ├── process.go                   # Claude 子进程管理
│   │   ├── executor.go                  # 任务执行器
│   │   └── watcher.go                   # Watchdog 监控
│   │
│   ├── platform/                        # 平台适配器
│   │   ├── platform.go                  # Platform 接口定义
│   │   ├── registry.go                  # 平台注册表
│   │   ├── github.go                    # GitHub 实现
│   │   ├── gitee.go                     # Gitee 实现
│   │   ├── gitlab.go                    # GitLab 实现
│   │   └── jenkins.go                   # Jenkins 实现
│   │
│   ├── buildctx/                        # 上下文构建
│   │   ├── context.go                   # CI 上下文
│   │   ├── diff.go                      # Git Diff 获取
│   │   ├── chunker.go                   # 智能分片
│   │   └── pruner.go                    # 上下文剪枝
│   │
│   ├── claude/                          # Claude 集成
│   │   ├── client.go                    # Claude API 客户端
│   │   ├── process.go                   # 进程管理
│   │   ├── stream.go                    # 流式输出处理
│   │   └── parser.go                    # 输出解析器
│   │
│   ├── skill/                           # Skill 管理
│   │   ├── skill.go                     # Skill 定义和加载
│   │   ├── registry.go                  # Skill 注册表
│   │   ├── loader.go                    # Skill 加载器
│   │   └── executor.go                  # Skill 执行器
│   │
│   ├── config/                          # 配置管理
│   │   ├── config.go                    # 配置结构定义
│   │   ├── loader.go                    # 配置加载器
│   │   ├── validator.go                  # 配置验证
│   │   └── defaults.go                  # 默认值
│   │
│   ├── cache/                           # 缓存系统
│   │   ├── cache.go                     # 缓存接口
│   │   ├── memory.go                    # 内存缓存
│   │   ├── disk.go                      # 磁盘缓存
│   │   └── key.go                       # 缓存键生成
│   │
│   ├── output/                          # 输出处理
│   │   ├── formatter.go                 # 结果格式化
│   │   ├── reporter.go                  # 平台报告
│   │   └── comment.go                   # 评论生成器
│   │
│   ├── security/                        # 安全模块
│   │   ├── rbac.go                      # RBAC 权限控制
│   │   ├── sandbox.go                   # 沙箱隔离
│   │   └── secrets.go                   # 密钥管理
│   │
│   ├── observability/                   # 可观测性
│   │   ├── logger.go                    # 结构化日志
│   │   ├── metrics.go                   # Prometheus 指标
│   │   ├── trace.go                     # 分布式追踪
│   │   └── audit.go                     # 审计日志
│   │
│   └── version/                         # 版本信息
│       └── version.go                   # 版本常量
│
├── skills/                              # 内置 Skills
│   ├── README.md                        # Skills 索引
│   ├── code-reviewer/                   # 代码审查
│   │   ├── SKILL.md                     # Skill 定义
│   │   └── scripts/                     # 辅助脚本
│   │       └── helpers.sh               # Shell 辅助脚本
│   ├── test-generator/                  # 测试生成
│   │   ├── SKILL.md
│   │   └── scripts/
│   ├── change-analyzer/                 # 变更分析
│   │   ├── SKILL.md
│   │   └── scripts/
│   ├── log-analyzer/                    # 日志分析
│   │   ├── SKILL.md
│   │   └── scripts/
│   └── issue-triage/                    # Issue 分类
│       ├── SKILL.md
│       └── scripts/
│
├── configs/                             # 配置示例
│   ├── cicd-ai-toolkit.yaml            # 完整配置示例
│   ├── minimal.yaml                     # 最小化配置
│   └── github-actions.yaml              # GitHub Actions 配置
│
├── actions/                             # GitHub Actions
│   ├── setup/action.yml                 # 环境设置 Action
│   ├── review/action.yml                # 代码审查 Action
│   ├── test-gen/action.yml              # 测试生成 Action
│   ├── analyze/action.yml               # 变更分析 Action
│   └── all/action.yml                   # 全功能组合 Action
│
├── .github/                             # GitHub 资源
│   ├── workflows/                       # GitHub Actions 工作流
│   │   ├── ci.yml                       # 持续集成
│   │   ├── release.yml                  # 发布流程
│   │   └── test.yml                     # 测试流程
│   ├── ISSUE_TEMPLATE/                  # Issue 模板
│   ├── PULL_REQUEST_TEMPLATE.md        # PR 模板
│   └── dependabot.yml                  # Dependabot 配置
│
├── .gitee/                              # Gitee 资源
│   └── workflows/                       # Gitee Go 工作流
│       └── ai-review.yml                # AI 审查工作流
│
├── docs/                                # 文档
│   ├── architecture/                    # 架构文档
│   │   └── overview.md
│   ├── api/                             # API 文档
│   │   └── openapi.yaml
│   ├── guides/                          # 使用指南
│   │   ├── getting-started.md
│   │   ├── custom-skills.md
│   │   └── platform-integration.md
│   └── development/                     # 开发文档
│       ├── contributing.md
│       └── testing.md
│
├── test/                                # 测试
│   ├── unit/                            # 单元测试
│   │   ├── runner_test.go
│   │   ├── platform_test.go
│   │   └── skill_test.go
│   ├── integration/                     # 集成测试
│   │   ├── github_test.go
│   │   └── gitee_test.go
│   ├── e2e/                             # 端到端测试
│   │   └── scenarios/
│   └── fixtures/                        # 测试固件
│       ├── diffs/                       # 测试用 Diff
│       └── expected/                    # 预期结果
│
├── build/                               # 构建相关
│   ├── docker/                          # Docker 相关
│   │   ├── Dockerfile                   # 主 Dockerfile
│   │   ├── Dockerfile.slim              # 精简版 Dockerfile
│   │   └── docker-compose.yml           # 本地开发环境
│   ├── packaging/                       # 打包脚本
│   │   ├── build.sh                     # 构建脚本
│   │   ├── release.sh                   # 发布脚本
│   │   └── checksum.sh                  # 校验和生成
│   └── ci/                              # CI 脚本
│       ├── build.sh
│       └── test.sh
│
├── scripts/                             # 开发脚本
│   ├── generate.go.sh                   # Go 代码生成
│   ├── lint.sh                          # 代码检查
│   ├── test.sh                          # 运行测试
│   └── clean.sh                         # 清理临时文件
│
├── examples/                            # 示例
│   ├── basic-review/                    # 基础审查示例
│   │   ├── .cicd-ai-toolkit.yaml
│   │   └── README.md
│   ├── multi-skill/                     # 多 Skill 示例
│   │   ├── .cicd-ai-toolkit.yaml
│   │   └── README.md
│   └── custom-skill/                    # 自定义 Skill 示例
│       └── my-skill/
│           └── SKILL.md
│
├── go.mod                               # Go 模块定义
├── go.sum                               # Go 依赖锁定
├── Makefile                             # Make 构建配置
├── Dockerfile                           # Docker 镜像
├── Dockerfile.slim                      # 精简版 Docker 镜像
├── .dockerignore                        # Docker 忽略文件
├── .gitignore                           # Git 忽略文件
├── LICENSE                               # 许可证 (Apache 2.0)
├── README.md                             # 项目说明
├── CHANGELOG.md                          # 变更日志
└── CLAUDE.md                             # Claude 项目上下文
```

## 3. 核心目录说明

### 3.1 `cmd/` - 可执行文件入口

遵循标准 Go 项目布局，每个子目录是一个可执行文件。

```go
// cmd/cicd-runner/main.go
package main

func main() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### 3.2 `pkg/` - 核心库

可被外部项目引用的公共库。

#### `pkg/runner/` - 运行器

```go
// pkg/runner/lifecycle.go
package runner

type Runner struct {
    config    *config.Config
    platform  platform.Platform
    skills    *skill.Registry
    cache     *cache.Cache
    metrics   *observability.Metrics
}

func (r *Runner) Bootstrap(ctx context.Context) error
func (r *Runner) Run(ctx context.Context, req *RunRequest) (*RunResult, error)
func (r *Runner) Shutdown(ctx context.Context) error
```

#### `pkg/platform/` - 平台适配

```go
// pkg/platform/platform.go
package platform

type Platform interface {
    Name() string
    GetPullRequest(ctx context.Context, number int) (*PullRequest, error)
    PostComment(ctx context.Context, number int, body string) error
    // ...
}
```

### 3.3 `skills/` - Skills

每个 Skill 是一个独立的目录，包含 `SKILL.md` 和可选的辅助脚本。

```markdown
<!-- skills/code-reviewer/SKILL.md -->
---
name: "code-reviewer"
version: "1.0.0"
description: "AI-powered code review"
author: "cicd-ai-toolkit"
options:
  thinking:
    budget_tokens: 4096
tools:
  allow: ["read", "grep", "ls"]
---

# Code Reviewer

You are an expert code reviewer...
```

### 3.4 `actions/` - GitHub Actions

复合 Actions 用于模块化 CI/CD 步骤。

```yaml
# actions/review/action.yml
name: 'AI Code Review'
description: 'Perform AI-powered code review'
inputs:
  skills:
    description: 'Skills to run'
    required: false
    default: 'code-reviewer'
runs:
  using: 'composite'
  steps:
    - run: cicd-runner run --skills ${{ inputs.skills }}
      shell: bash
```

## 4. 构建产物

```
build/
├── bin/
│   ├── cicd-runner-linux-amd64        # Linux 二进制
│   ├── cicd-runner-linux-arm64
│   ├── cicd-runner-darwin-amd64        # macOS 二进制
│   ├── cicd-runner-darwin-arm64
│   └── cicd-runner-windows-amd64.exe   # Windows 二进制
├── dist/
│   ├── cicd-ai-toolkit-v1.0.0.tar.gz  # 源码打包
│   └── checksums.txt                   # SHA256 校验和
└── docker/
    ├── cicd-ai-toolkit:latest         # Docker 镜像
    └── cicd-ai-toolkit:v1.0.0
```

## 5. 配置文件位置

| 配置类型 | 位置 | 用途 |
|---------|------|------|
| **用户配置** | `~/.cicd-ai-toolkit/config.yaml` | 全局默认配置 |
| **项目配置** | `.cicd-ai-toolkit.yaml` | 项目特定配置 |
| **Skill 定义** | `skills/<name>/SKILL.md` | Skill 定义 |
| **CI 配置** | `.github/workflows/*.yml` | GitHub Actions |
| **CI 配置** | `.gitee/workflows/*.yml` | Gitee Go |
| **项目上下文** | `CLAUDE.md` | Claude 项目说明 |

## 6. Go 模块依赖

```
cicd-ai-toolkit
├── github.com/anthropics/claude-code       # Claude Code CLI
├── github.com/google/go-github              # GitHub API
├── github.com/xanzy/go-gitlab               # GitLab API
├── github.com/uber-go/zap                   # 结构化日志
├── github.com/prometheus/client_golang     # Prometheus 指标
├── github.com/spf13/viper                   # 配置管理
├── github.com/spf13/cobra                   # CLI 框架
├── gopkg.in/yaml.v3                          # YAML 解析
└── github.com/open-policy-agent/opa         # OPA 策略引擎
```

## 7. 目录约定

### 7.1 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| **文件名** | `snake_case` | `config_loader.go` |
| **包名** | `lowercase` | `package runner` |
| **接口名** | `PascalCase` + `er` 后缀 | `Platform`, `Cache` |
| **常量** | `PascalCase` | `DefaultTimeout` |
| **私有变量** | `camelCase` | `internalState` |

### 7.2 文件组织

- 每个包一个目录
- 包名与目录名一致
- 测试文件与源文件同目录：`xxx_test.go`
- 接口和实现在同一文件或分开

### 7.3 依赖方向

```
cmd/
  ↓ (depends on)
pkg/
  ↓ (imported by)
  skills/
    (runtime discovery)

外部项目/
  ↓ (can import)
pkg/
  ↓ (no import)
cmd/
```

## 8. Makefile 目标

```makefile
# 构建目标
.PHONY: build build-all clean test lint docker-release

# 构建当前平台
build:
	@echo "Building cicd-runner..."
	@go build -o bin/cicd-runner ./cmd/cicd-runner

# 构建所有平台
build-all:
	@echo "Building for all platforms..."
	@./build/packaging/build-all.sh

# 运行测试
test:
	@echo "Running tests..."
	@go test -race -coverprofile=coverage.out ./...

# 运行代码检查
lint:
	@echo "Running linters..."
	@golangci-lint run ./...

# 构建 Docker 镜像
docker:
	@docker build -t cicd-ai-toolkit:latest .

# 发布
release:
	@./build/packaging/release.sh $(VERSION)
```

## 9. 依赖关系 (Dependencies)

- **Related**: [SPEC-CORE-01](./SPEC-CORE-01-Runner_Lifecycle.md) - Runner 入口
- **Related**: [SPEC-SKILL-01](./SPEC-SKILL-01-Skill_Definition.md) - Skill 定义
- **Related**: [SPEC-DIST-01](./SPEC-DIST-01-Distribution.md) - 分发机制

## 10. 验收标准 (Acceptance Criteria)

1. **目录存在**: 所有必需目录存在，结构清晰。
2. **可编译**: `go build ./cmd/cicd-runner` 成功。
3. **测试运行**: `go test ./...` 通过所有单元测试。
4. **Skill 加载**: `skills/code-reviewer/SKILL.md` 能被正确解析。
5. **Docker 构建**: `docker build` 成功构建镜像。
6. **文档完整**: README.md 和必要的文档存在。
7. **配置示例**: `configs/cicd-ai-toolkit.yaml` 是有效配置。
8. **CI 通过**: GitHub Actions 工作流通过。
