# 贡献指南 / Contributing Guide

感谢您对 CICD AI Toolkit 的关注！我们欢迎各种形式的贡献。

---

## 目录 / Table of Contents

- [快速开始](#快速开始--quick-start)
- [开发环境设置](#开发环境设置--development-environment)
- [项目结构](#项目结构--project-structure)
- [开发工作流](#开发工作流--development-workflow)
- [代码规范](#代码规范--coding-standards)
- [测试指南](#测试指南--testing-guidelines)
- [提交 PR](#提交-pr--submitting-a-pr)
- [社区准则](#社区准则--community-guidelines)

---

## 快速开始 / Quick Start

### 1. Fork 并克隆仓库

```bash
# Fork 项目后，克隆你的 fork
git clone https://github.com/YOUR_USERNAME/cicd-tool-kit.git
cd cicd-tool-kit

# 添加上游远程仓库
git remote add upstream https://github.com/cicd-ai-toolkit/cicd-tool-kit.git
```

### 2. 安装开发依赖

```bash
# 安装 Go 依赖
go mod download

# 安装开发工具
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
```

### 3. 安装 Git Hooks

```bash
# 安装 pre-commit 和 pre-push hooks
./scripts/install-hooks.sh
```

### 4. 验证环境

```bash
# 运行测试
go test ./...

# 运行 linter
golangci-lint run --config=.golangci.yaml

# 构建项目
go build -o bin/cicd-runner ./cmd/cicd-runner
```

---

## 开发环境设置 / Development Environment

### 必需工具

| 工具 | 版本 | 用途 |
|------|------|------|
| Go | 1.23+ | 核心开发语言 |
| Git | 2.30+ | 版本控制 |
| golangci-lint | latest | 代码检查 |
| staticcheck | latest | 静态分析 |

### 推荐工具

```bash
# 代码格式化
go install golang.org/x/tools/cmd/goimports@latest

# 测试覆盖率
go install github.com/golangci/gocover-cobertura/cmd/gocover-cobertura@latest

# 本地 HTTP 测试
go install github.com/cespare/xxhash/v2@latest
```

### IDE 配置

**VS Code (推荐)**

安装以下扩展：
- Go (golang.go)
- GitLens (eamodio.gitlens)
- YAML (redhat.vscode-yaml)

**.vscode/settings.json** 推荐配置：

```json
{
  "go.formatTool": "goimports",
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--fast", "--config=.golangci.yaml"],
  "go.testFlags": ["-v", "-race"],
  "go.testTimeout": "60s",
  "[go]": {
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
      "source.organizeImports": "explicit"
    }
  }
}
```

---

## 项目结构 / Project Structure

```
cicd-tool-kit/
├── cmd/                      # CLI 入口点
│   └── cicd-runner/
│       ├── main.go           # 程序入口
│       └── root.go           # Cobra 根命令
│
├── pkg/                      # 公共库
│   ├── ai/                   # AI 执行引擎
│   │   ├── brain.go          # Claude 交互
│   │   ├── factory.go        # 后端创建
│   │   └── utils.go          # 工具函数
│   ├── buildcontext/         # Git 上下文
│   ├── claude/               # Claude Code 集成
│   ├── config/               # 配置加载
│   ├── errors/               # 错误定义
│   ├── observability/        # 可观测性
│   ├── perf/                 # 性能工具
│   ├── platform/             # 平台 API
│   ├── runner/               # 核心编排
│   ├── security/             # 安全检查
│   ├── skill/                # Skill 加载
│   └── webhook/              # Webhook 处理
│
├── skills/                   # 可插拔 Skills
│   ├── code-reviewer/
│   │   └── SKILL.md
│   ├── test-generator/
│   └── change-analyzer/
│
├── configs/                  # 配置示例
├── docs/                     # 文档
├── scripts/                  # 构建脚本
├── .github/                  # GitHub 配置
│   ├── workflows/
│   └── ISSUE_TEMPLATE/
└── .claude/                  # Claude 规则
    └── rules/
```

---

## 开发工作流 / Development Workflow

### 完整流程

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  创建 Issue  │ → │  创建分支    │ → │  开发提交    │ → │  发起 PR     │
│  (gh issue) │    │  (git checkout -b)│ │  (git commit)│  │  (gh pr create)│
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                                            │                    │
                                            ▼                    ▼
                                    ┌─────────────┐    ┌─────────────┐
                                    │  自动检查    │ → │  审核合并    │
                                    │  (hooks+CI) │    │  (merge)     │
                                    └─────────────┘    └─────────────┘
```

### 1. 创建 Issue

每个功能/修复都应该有对应的 Issue：

```bash
# 使用模板创建 Issue
gh issue create --template "feature_request.yml"

# 或直接创建
gh issue create --title "[feat] 添加新功能" --body "描述内容"
```

### 2. 创建分支

**禁止直接在 main 分支修改**

```bash
# 同步上游最新代码
git fetch upstream
git checkout main
git merge upstream/main

# 创建功能分支 (引用 Issue 编号)
git checkout -b feat/123-add-feature
```

**分支命名规范**：

| 类型 | 格式 | 示例 |
|------|------|------|
| 功能 | `feat/<id>-desc` | `feat/123-async-mode` |
| 修复 | `fix/<id>-desc` | `fix/456-memory-leak` |
| 重构 | `refactor/<id>-desc` | `refactor/789-cleanup` |
| 文档 | `docs/<id>-desc` | `docs/200-readme` |
| 测试 | `test/<id>-desc` | `test/300-coverage` |

### 3. 开发与提交

**Pre-commit Hook (~2秒)**：
- 自动运行 `go fmt`
- 自动运行 `go vet`
- 检查 `go.mod` 是否 tidy

```bash
# 提交代码 (hook 自动运行)
git add .
git commit -m "feat(ai): add async session mode

- Implement SessionManager
- Add UUID v5 mapping
- Add unit tests

Refs #123

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

**提交信息格式**：

```
<type>(<scope>): <subject>

<body>

Refs #<issue>

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

| Type | 说明 | 示例 |
|------|------|------|
| `feat` | 新功能 | `feat(ai): add async mode` |
| `fix` | Bug 修复 | `fix(runner): handle nil config` |
| `refactor` | 重构 | `refactor(cli): extract command` |
| `perf` | 性能优化 | `perf(cache): reduce allocations` |
| `test` | 测试 | `test(ai): add mock tests` |
| `docs` | 文档 | `docs(readme): update examples` |
| `chore` | 杂项 | `chore(deps): upgrade go 1.23` |

### 4. 推送代码

**Pre-push Hook (~1分钟)**：
- 检查 `go.mod` 是否 tidy
- 运行 `golangci-lint`
- 运行 `go test -short`

```bash
git push -u origin feat/123-add-feature
```

### 5. 定期同步上游

```bash
# 在功能分支上，每天或上游有更新时执行
git fetch upstream
git rebase upstream/main
```

---

## 代码规范 / Coding Standards

### Go 代码风格

遵循 [Effective Go](https://go.dev/doc/effective_go) 和 [Uber Go Style Guide](https://github.com/uber-go/guide)。

**基本原则**：

1. **包名使用小写单词**
   ```go
   // Good
   package runner

   // Bad
   package Runner
   package runner_pkg
   ```

2. **导出函数添加注释**
   ```go
   // Review executes AI code review on the given diff.
   // It returns structured issues found in the code.
   func Review(ctx context.Context, diff string) ([]Issue, error) {
       // ...
   }
   ```

3. **错误处理**
   ```go
   // Good - 包装上下文
   if err := run(); err != nil {
       return fmt.Errorf("failed to run review: %w", err)
   }

   // Bad - 丢弃错误
   _ := run()
   ```

4. **接口定义**
   ```go
   // Good - 接口在使用方定义
   type Reviewer interface {
       Review(ctx context.Context, diff string) ([]Issue, error)
   }

   // Bad - 提前定义"以防万一"
   type Runner interface {
       Run() error
       Stop() error
   }
   ```

### 错误处理

```go
// 使用 pkg/errors 包装
import "github.com/cicd-ai-toolkit/cicd-runner/pkg/errors"

func Process(diff string) error {
    if err := validate(diff); err != nil {
        return errors.Wrap(err, "validation failed")
    }
    // ...
}

// 检查错误类型
if errors.Is(err, context.Canceled) {
    // 处理取消
}
```

### 并发安全

```go
// 使用 sync.RWMutex 保护共享状态
type SessionPool struct {
    mu   sync.RWMutex
    m    map[string]*Session
}

func (p *SessionPool) Get(id string) (*Session, bool) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    s, ok := p.m[id]
    return s, ok
}
```

### 配置结构

```go
// 配置结构使用 JSON/YAML 标签
type Config struct {
    // Claude API 配置
    Model    string  `yaml:"model" json:"model" env:"CLAUDE_MODEL"`
    MaxTurns int     `yaml:"max_turns" json:"maxTurns" env:"CLAUDE_MAX_TURNS"`
    Timeout  string  `yaml:"timeout" json:"timeout" env:"CLAUDE_TIMEOUT"`
}
```

---

## 测试指南 / Testing Guidelines

### 测试原则

1. **单元测试** - 每个包都应该有测试
2. **表驱动测试** - 使用 table-driven 测试多个场景
3. **Mock 外部依赖** - 使用接口和 mock

### 单元测试示例

```go
func TestReview(t *testing.T) {
    tests := []struct {
        name    string
        diff    string
        wantErr bool
        wantLen int
    }{
        {
            name:    "empty diff",
            diff:    "",
            wantErr: true,
        },
        {
            name:    "valid go code",
            diff:    "package main\n\nfunc main() {}",
            wantErr: false,
            wantLen: 0,
        },
        {
            name:    "contains security issue",
            diff:    "package main\n\nfunc main() { exec.Command(\"rm\", \"-rf\", \"/\") }",
            wantErr: false,
            wantLen: 1,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            issues, err := Review(context.Background(), tt.diff)
            if (err != nil) != tt.wantErr {
                t.Errorf("Review() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !tt.wantErr && len(issues) != tt.wantLen {
                t.Errorf("Review() got %d issues, want %d", len(issues), tt.wantLen)
            }
        })
    }
}
```

### Mock 示例

```go
// mock platform for testing
type mockPlatform struct {
    diffFunc func(prID int) (string, error)
}

func (m *mockPlatform) GetDiff(ctx context.Context, prID int) (string, error) {
    return m.diffFunc(prID)
}

func TestRunnerProcess(t *testing.T) {
    mock := &mockPlatform{
        diffFunc: func(prID int) (string, error) {
            return "sample diff", nil
        },
    }

    runner := NewRunner(mock)
    err := runner.Process(context.Background(), 123)
    if err != nil {
        t.Fatalf("Process() error = %v", err)
    }
}
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./pkg/ai

# 运行匹配的测试
go test -run TestReview ./pkg/ai

# 运行测试并显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 运行基准测试
go test -bench=. -benchmem
```

### 测试覆盖率目标

| 包类型 | 目标覆盖率 |
|--------|-----------|
| 核心逻辑 (pkg/ai, pkg/runner) | > 70% |
| 平台集成 (pkg/platform) | > 60% |
| 工具函数 (pkg/*) | > 80% |
| CLI 入口 (cmd/*) | > 40% |

---

## 提交 PR / Submitting a PR

### 1. 创建 PR

```bash
# 确保分支是最新的
git fetch upstream
git rebase upstream/main

# 推送到你的 fork
git push -u origin feat/123-add-feature

# 创建 PR
gh pr create --title "feat(ai): add async session mode" \
             --body "Resolves #123"
```

### 2. PR 描述模板

PR 创建时会自动使用 `.github/pull_request_template.md`：

```markdown
## 概述
简短描述此 PR 的目的

## 变更内容
- [ ] 变更点 1
- [ ] 变更点 2

## 关联 Issue
Resolves #123

## 测试计划
- [ ] 本地测试通过
- [ ] 单元测试新增/更新
- [ ] 手动测试场景

## 检查清单
- [ ] 代码遵循项目规范
- [ ] 自我审查代码
- [ ] 文档已更新（如需要）
- [ ] 无合并冲突
```

### 3. PR 检查

PR 创建后会自动检查：
- **分支命名**: `feat/123-desc` 格式
- **Issue 关联**: 包含 `Resolves #123`
- **CI 状态**: 所有检查通过

### 4. 审核反馈

- 及时响应审核意见
- 按反馈修改后推送新的 commit
- 讨论达成一致后再合并

### 5. 合并方式

- **Squash Merge**: 多个 commit 压缩为一个 (推荐)
- 合并后自动删除分支

---

## 社区准则 / Community Guidelines

### 行为准则

我们致力于提供友好、安全的社区环境：

1. **尊重他人** - 建设性讨论，尊重不同观点
2. **欢迎新手** - 帮助新贡献者成长
3. **关注问题** - 讨论技术而非个人
4. **接受反馈** - 开放接受建设性批评

### 沟通渠道

- **GitHub Issues**: Bug 报告、功能请求
- **GitHub Discussions**: 技术讨论、问题求助
- **PR Review**: 代码审查

### 获得帮助

1. 查阅文档 (`docs/`)
2. 搜索已有 Issues
3. 在 Discussion 提问
4. 参加 weekly sync (如有)

---

## 常见问题 / FAQ

### Q: 我该如何选择要贡献的 Issue？

**A**: 查看 Issue 标签：
- `good first issue`: 适合新手
- `help wanted`: 需要帮助
- `enhancement`: 功能增强

### Q: Pre-commit hook 失败怎么办？

**A**:
```bash
# 查看具体错误
# 运行修复命令
go fmt ./...
go vet ./...
go mod tidy

# 或临时跳过
git commit --no-verify -m "msg"
```

### Q: 如何处理合并冲突？

**A**:
```bash
git fetch upstream
git rebase upstream/main
# 解决冲突
git add .
git rebase --continue
git push --force-with-lease
```

### Q: CI 检查失败怎么办？

**A**:
1. 查看 Actions 日志
2. 本地复现问题
3. 修复后推送新 commit

---

## 许可证 / License

贡献的代码将使用 [Apache License 2.0](LICENSE) 许可。

提交 PR 即表示您同意：
- 您的代码将按项目许可证发布
- 您拥有贡献代码的版权
- 您的代码是原创的

---

**再次感谢您的贡献！** 🎉

如有问题，请通过 Issue 或 Discussion 与我们联系。
