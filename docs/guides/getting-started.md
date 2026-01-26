# 快速入门

欢迎使用 CICD AI Toolkit！本指南将帮助你在 5 分钟内完成设置并运行第一次 AI 代码审查。

---

## 前置要求

- **Go 1.21+** (从源码构建时需要)
- **Docker** (可选，用于容器化部署)
- **Anthropic API Key** - 从 [console.anthropic.com](https://console.anthropic.com/) 获取
- **Git 仓库** - 需要分析的代码仓库

---

## 安装

### 方式一：使用 Docker (推荐)

```bash
docker pull ghcr.io/cicd-ai-toolkit/cicd-ai-toolkit:latest
```

### 方式二：从源码构建

```bash
# 克隆仓库
git clone https://github.com/cicd-ai-toolkit/cicd-ai-toolkit.git
cd cicd-ai-toolkit

# 构建
make build

# 二进制文件将输出到 ./bin/cicd-runner
```

### 方式三：下载预编译二进制

访问 [Releases](https://github.com/cicd-ai-toolkit/cicd-ai-toolkit/releases) 下载适合你系统的二进制文件。

---

## 配置

### 1. 设置 API Key

```bash
# Linux/macOS
export ANTHROPIC_API_KEY="your-api-key-here"

# Windows PowerShell
$env:ANTHROPIC_API_KEY="your-api-key-here"
```

### 2. 创建配置文件

在你的 Git 仓库根目录创建 `.cicd-ai-toolkit.yaml`:

```yaml
version: "1.0"

# Claude 配置
claude:
  model: "sonnet"           # 可选: haiku (快), sonnet (平衡), opus (深)
  max_budget_usd: 5.0       # 单次分析最大花费
  timeout: 300s             # 超时时间

# 启用的技能
skills:
  - name: code-reviewer
    enabled: true
    config:
      severity_threshold: "medium"  # 只报告 medium 及以上问题

  - name: change-analyzer
    enabled: true

# 平台配置
platform:
  type: auto  # 自动检测平台
  github:
    post_comment: true      # 发表 PR 评论
    fail_on_error: false    # 分析失败不阻塞 CI
```

### 3. 验证配置

```bash
cicd-runner skill validate ./skills/code-reviewer
```

---

## 本地运行

### 分析当前变更

```bash
# 分析未暂存的变更
cicd-runner run -s code-reviewer

# 分析已暂存的变更
cicd-runner run -s code-reviewer --staged

# 详细输出
cicd-runner run -s code-reviewer -v

# 干运行 (不发表评论)
cicd-runner run -s code-reviewer --dry-run
```

### 运行多个技能

```bash
cicd-runner run -s "code-reviewer,change-analyzer,test-generator"
```

---

## CI/CD 集成

### GitHub Actions

创建 `.github/workflows/ai-review.yml`:

```yaml
name: AI Code Review
on:
  pull_request:
    types: [opened, synchronize]
    paths:
      - '**.go'
      - '**.js'
      - '**.ts'
      - '**.py'

jobs:
  ai-review:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: AI Review
        uses: cicd-ai-toolkit/actions/review@v1
        with:
          api-key: ${{ secrets.ANTHROPIC_API_KEY }}
          skills: "code-reviewer,change-analyzer"
          config: .cicd-ai-toolkit.yaml
```

### GitLab CI

创建 `.gitlab-ci.yml`:

```yaml
ai-review:
  stage: test
  image: ghcr.io/cicd-ai-toolkit/cicd-ai-toolkit:latest
  script:
    - cicd-runner run -s code-reviewer
  only:
    - merge_requests
```

### Jenkins Pipeline

```groovy
pipeline {
    agent any

    stages {
        stage('AI Review') {
            steps {
                script {
                    docker.image('ghcr.io/cicd-ai-toolkit/cicd-ai-toolkit:latest').inside {
                        sh 'cicd-runner run -s code-reviewer'
                    }
                }
            }
        }
    }
}
```

---

## 下一步

| 文档 | 描述 |
|------|------|
| [自定义技能](custom-skills.md) | 编写自己的技能 |
| [平台集成](platform-integration.md) | 深入集成配置 |
| [配置参考](../configuration.md) | 完整配置选项 |

---

## 故障排查

### 问题：Bootstrap 失败

**错误**: `workspace is not a git repository`

**解决**: 确保在 Git 仓库中运行

### 问题：API 超时

**错误**: `context deadline exceeded`

**解决**: 增加配置中的 `timeout` 值或启用分片

### 问题：成本过高

**解决**:
1. 启用缓存 (自动)
2. 调整 `max_budget_usd`
3. 使用 `haiku` 模型进行初筛

---

## 获取帮助

- [GitHub Issues](https://github.com/cicd-ai-toolkit/cicd-ai-toolkit/issues)
- [GitHub Discussions](https://github.com/cicd-ai-toolkit/cicd-ai-toolkit/discussions)
- [文档首页](../README.md)
