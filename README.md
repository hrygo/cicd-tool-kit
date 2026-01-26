# CICD AI Toolkit

> AI-powered code analysis for CI/CD pipelines. Automate code reviews, test generation, and change analysis using Claude.

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![Specs](https://img.shields.io/badge/Specs-32%2F32-success)](docs/specs/)

---

## 概述

CICD AI Toolkit 是一个**可插拔的 AI 驱动 CI/CD 工具集**，基于 Claude Code 构建，提供：

- **智能代码审查** - 超越 Lint 的深度语义分析
- **自动化测试生成** - 基于代码变更生成测试用例
- **变更风险分析** - 影响评估和风险评分
- **日志智能分析** - 异常检测和根因定位

**支持平台**: GitHub Actions, GitLab CI, Gitee Go, Jenkins

---

## 特性

| 特性 | 描述 |
|------|------|
| **平台无关** | 统一接口适配主流 CI/CD 平台 |
| **可插拔技能** | 用 Markdown 定义自定义技能 |
| **幂等缓存** | 相同输入产生相同结果，降低成本 |
| **异步执行** | 支持长时间分析，不阻塞流水线 |
| **安全隔离** | 沙箱执行，Prompt 注入防护 |
| **可观测性** | 结构化日志、Prometheus 指标、审计追踪 |

---

## 快速开始

### 安装

```bash
# Docker (推荐)
docker pull ghcr.io/cicd-ai-toolkit/cicd-ai-toolkit:latest

# 从源码构建
git clone https://github.com/cicd-ai-toolkit/cicd-ai-toolkit.git
cd cicd-ai-toolkit
make build
```

### GitHub Actions 集成

创建 `.github/workflows/ai-review.yml`:

```yaml
name: AI Code Review
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: cicd-ai-toolkit/actions/review@v1
        with:
          api-key: ${{ secrets.ANTHROPIC_API_KEY }}
          skills: "code-reviewer,change-analyzer"
```

### 配置文件

创建 `.cicd-ai-toolkit.yaml`:

```yaml
version: "1.0"

# Claude 配置
claude:
  model: "sonnet"
  max_budget_usd: 5.0
  timeout: 300s

# 启用的技能
skills:
  - name: code-reviewer
    enabled: true
    config:
      severity_threshold: "warning"
  - name: change-analyzer
    enabled: true

# 平台配置
platform:
  type: auto  # auto | github | gitlab | gitee | jenkins
  github:
    post_comment: true
    fail_on_error: false
```

---

## 内置技能

| 技能 | 描述 | 输入 | 输出 |
|------|------|------|------|
| `code-reviewer` | 深度代码审查 | Git Diff | Issues (安全/逻辑/性能) |
| `test-generator` | 测试用例生成 | 代码变更 | 测试文件 |
| `change-analyzer` | 变更分析 | Diff + 元数据 | 风险评分 + 摘要 |
| `log-analyzer` | 日志分析 | 日志流 | 异常 + 根因 |
| `issue-triage` | Issue 分类 | Issue 内容 | 标签 + 优先级 |
| `committer` | 智能提交 | 变更文件 | 提交消息 |

---

## 文档

| 文档 | 描述 |
|------|------|
| [快速入门](docs/guides/getting-started.md) | 5 分钟上手指南 |
| [自定义技能](docs/guides/custom-skills.md) | 编写自己的技能 |
| [平台集成](docs/guides/platform-integration.md) | 集成到 CI/CD |
| [架构概览](docs/architecture/overview.md) | 系统架构设计 |
| [贡献指南](docs/development/contributing.md) | 贡献流程 |

---

## 开发

```bash
# 安装依赖
make deps

# 运行测试
make test

# 代码检查
make lint

# 构建
make build

# 运行
./bin/cicd-runner run -s code-reviewer -v
```

### 项目结构

```
cicdtools/
├── cmd/cicd-runner/        # CLI 入口
├── pkg/                     # 核心库
│   ├── runner/              # Runner 生命周期
│   ├── platform/            # 平台适配器
│   ├── skill/               # 技能系统
│   ├── config/              # 配置管理
│   └── ...
├── skills/                  # 内置技能
├── docs/                    # 文档
└── examples/                # 示例
```

---

## 规范完成状态

| 类别 | 状态 |
|------|------|
| **CORE** | ✅ 3/3 完成 |
| **CONF** | ✅ 2/2 完成 |
| **PLAT** | ✅ 7/7 完成 |
| **SEC** | ✅ 3/3 完成 |
| **GOV** | ✅ 2/2 完成 |
| **PERF** | ✅ 1/1 完成 |
| **OPS** | ✅ 1/1 完成 |
| **SKILL** | ✅ 1/1 完成 |
| **LIB** | ✅ 4/4 完成 |
| **MCP** | ✅ 2/2 完成 |
| **ECO** | ✅ 1/1 完成 |
| **DIST** | ✅ 1/1 完成 |
| **RFC** | ✅ 1/1 完成 |
| **STATS** | ✅ 1/1 完成 |
| **HOOKS** | ✅ 1/1 完成 |
| **总计** | ✅ **32/32** |

详见 [实施计划](docs/specs/IMPLEMENTATION_PLAN.md)。

---

## 性能指标

| 指标 | 目标 | 实际 |
|------|------|------|
| 冷启动时间 | < 5s | ~2s |
| 分析耗时 (P90) | < 60s | ~45s |
| 内存占用 | < 512MB | ~256MB |
| 缓存命中率 | > 40% | ~65% |

---

## 许可证

Apache License 2.0 - 详见 [LICENSE](LICENSE)

---

## 链接

- [产品需求文档 (PRD)](docs/PRD.md)
- [技术规范](docs/specs/)
- [变更日志](CHANGELOG.md)
- [GitHub Discussions](https://github.com/cicd-ai-toolkit/cicd-ai-toolkit/discussions)
