# cicd-ai-toolkit

> 基于 Claude Code Headerless 模式的可插拔 CI/CD 工具集

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![Claude](https://img.shields.io/badge/Claude-Code-purple?logo=anthropic)](https://code.anthropic.com/)

## 简介

**cicd-ai-toolkit** 是一个企业级 CI/CD 智能化工具集，基于 Anthropic Claude Code 构建。通过可插拔的 Skills 架构，实现代码审查、测试生成、变更分析等 AI 赋能的 DevOps 自动化。

### 核心特性

- **Runner + Skills 架构**: Go 高性能运行器 + Claude 智能决策
- **原生 Claude Code 集成**: 利用完整工具生态 (Bash, Edit, Read, MCP)
- **可插拔技能**: Markdown 定义的 Skills，无需编译即可扩展
- **多平台支持**: GitHub Actions, Gitee Enterprise, GitLab CI/CD
- **成本控制**: 内置预算限制和智能缓存

### 支持的 Skills

| Skill | 功能 | 状态 |
|-------|------|------|
| **code-reviewer** | 安全、性能、逻辑、架构分析 | ✅ MVP |
| **test-generator** | 基于代码变更生成测试用例 | ✅ MVP |
| **change-analyzer** | PR 总结、影响分析、风险评分 | ✅ MVP |
| **log-analyzer** | 日志分析、异常检测、根因定位 | ✅ MVP |

## 快速开始

### GitHub Actions 集成

1. **创建配置文件** `.cicd-ai-toolkit.yaml`:

```yaml
version: "1.0"

claude:
  model: "sonnet"
  max_budget_usd: 5.0

skills:
  - name: code-reviewer
    enabled: true

  - name: change-analyzer
    enabled: true

platform:
  github:
    post_comment: true
```

2. **添加工作流** `.github/workflows/ai-review.yml`:

```yaml
name: AI Code Review

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  ai-review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: AI Code Review
        uses: cicd-ai-toolkit/action@v1
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          anthropic_key: ${{ secrets.ANTHROPIC_API_KEY }}
```

3. **配置 Secrets**:
   - `ANTHROPIC_API_KEY`: 从 [Anthropic Console](https://console.anthropic.com/) 获取

### 本地运行

```bash
# 安装
go install github.com/cicd-ai-toolkit/cicd-runner@latest

# 审查当前分支变更
cicd-runner review --skills code-reviewer

# 生成测试
cicd-runner test-generate --skill test-generator

# 分析变更影响
cicd-runner analyze --skills change-analyzer
```

### Docker 运行

```bash
docker run --rm \
  -v $(pwd):/workspace \
  -e ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
  ghcr.io/cicd-ai-toolkit/cicd-runner:latest \
  review --skills code-reviewer
```

## 配置参考

完整配置示例见 [configs/.cicd-ai-toolkit.yaml](configs/.cicd-ai-toolkit.yaml)

### 核心配置项

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `claude.model` | Claude 模型 | sonnet |
| `claude.max_budget_usd` | 最大 API 花费 | 5.0 |
| `skills[].enabled` | 启用技能 | true |
| `platform.github.post_comment` | 发 PR 评论 | true |
| `global.exclude` | 排除文件模式 | *.lock, vendor/** |

## 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                      cicd-ai-toolkit                             │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Runner (Go)                          │   │
│  │  - Context Builder (Git/Logs)                           │   │
│  │  - Platform API Client (GitHub/GitLab)                  │   │
│  │  - Claude Session Manager                               │   │
│  └─────────────────────────────────────────────────────────┘   │
│                           │                                     │
│                   (Spawns Subprocess)                           │
│                           ▼                                     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   Claude Code                           │   │
│  │           (Headless / Agent Mode)                       │   │
│  └────────┬───────────────────────────────────────┬────────┘   │
│           │ (Loads)                               │ (Loads)    │
│  ┌────────▼────────┐                     ┌────────▼────────┐   │
│  │  Skill: Review  │                     │  Skill: Test    │   │
│  │   (SKILL.md)    │                     │   (SKILL.md)    │   │
│  └─────────────────┘                     └─────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## 开发

```bash
# 克隆仓库
git clone https://github.com/cicd-ai-toolkit/cicd-runner.git
cd cicd-runner

# 安装依赖
go mod download

# 运行测试
go test ./...

# 构建
go build -o bin/cicd-runner ./cmd/cicd-runner

# 运行
./bin/cicd-runner review --skills code-reviewer
```

## 路线图

### Phase 1 (MVP) - 当前
- ✅ Code Reviewer
- ✅ Test Generator
- ✅ Change Analyzer
- ✅ GitHub Actions 集成

### Phase 2
- ⏳ GitLab CI/CD 适配
- ⏳ Gitee Enterprise 深度集成
- ⏳ Security Scanner (Semgrep 集成)
- ⏳ 性能基准测试

### Phase 3
- ⏳ Self-Healing Agent
- ⏳ Multi-Agent Orchestration
- ⏳ Memory System (RAG)

## 贡献

欢迎贡献！请查看 [CONTRTRIBUTING.md](CONTRIBUTING.md)

## 许可证

Apache License 2.0 - 详见 [LICENSE](LICENSE)

## 致谢

- [Anthropic](https://www.anthropic.com/) - Claude Code
- [pr-agent](https://github.com/qodo-ai/pr-agent) - 灵感来源
- 所有贡献者

---

**文档版本**: v0.1.0 | **最后更新**: 2026-01-27
