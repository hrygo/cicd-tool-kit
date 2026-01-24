# SPEC-CONF-01: Configuration System

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24

## 1. 概述 (Overview)
为了适应不同的 CI/CD 环境（GitHub Actions, Gitee Go, Local CLI），Runner 需要一个灵活且安全的配置系统。该系统负责从文件、环境变量和默认值中加载配置，并处理优先级和合并逻辑。

## 2. 核心职责 (Core Responsibilities)
- **Hierarchical Loading**: 支持多层级配置加载（Defaults < File < Env）。
- **Schema Validation**: 确保配置合法性（Valid YAML, Range Checks）。
- **Secret Resolution**: 安全地处理敏感信息引用。

## 3. 详细设计 (Detailed Design)

### 3.1 优先级 (Precedence)
Runner 按以下顺序加载配置（后加载覆盖先加载）：
1.  **Defaults**: 硬编码在二进制中的安全默认值。
2.  **Global Config**: `$HOME/.cicd-ai-toolkit/config.yaml` (仅 CLI 模式)。
3.  **Project Config**: `./.cicd-ai-toolkit.yaml` (Repository Root)。
4.  **Environment Variables**: `CICD_TOOLKIT_*` (最高优先级)。

### 3.2 配置结构 (Schema)
对应 Go Struct 定义：

```go
type Config struct {
    Claude   ClaudeConfig   `yaml:"claude"`
    Skills   []SkillConfig  `yaml:"skills"`
    Platform PlatformConfig `yaml:"platform"`
    Global   GlobalConfig   `yaml:"global"`
}

type ClaudeConfig struct {
    Model        string        `yaml:"model" default:"sonnet"`
    MaxBudget    float64       `yaml:"max_budget_usd"`
    Timeout      time.Duration `yaml:"timeout" default:"300s"`
    AllowedTools []string      `yaml:"allowed_tools"`
}
```

### 3.3 环境变量覆盖 (Env Override)
支持使用双下划线 (`__`) 分隔层级来覆盖特定配置：
*   `CICD_TOOLKIT_CLAUDE__MODEL="opus"` -> 覆盖 `claude.model`
*   `CICD_TOOLKIT_GLOBAL__LOG_LEVEL="debug"` -> 覆盖 `global.log_level`

### 3.4 敏感信息 (Secret Handling)
配置文件中**严禁**直接通过明文存储 Secrets（如 API Keys）。
必须使用环境变量引用：
```yaml
# Good
platform:
  github:
    token_env: "GITHUB_TOKEN" # Runner will read os.Getenv("GITHUB_TOKEN")

# Bad (Validation Error)
platform:
  github:
    token: "ghp_xxxx" 
```

## 4. 依赖关系 (Dependencies)
- **Lib**: `github.com/spf13/viper` 或 `knadh/koanf` 用于配置加载。
- **Used by**: [SPEC-CORE-01](./SPEC-CORE-01-Runner_Lifecycle.md) 启动第一步。
- **Related**: [SPEC-CONF-02](./SPEC-CONF-02-Idempotency.md) - 幂等性配置。

## 5. 验收标准 (Acceptance Criteria)
1.  **Override Test**: 在配置文件中设置 `model: sonnet`，环境变量设置 `CICD_TOOLKIT_CLAUDE__MODEL=opus`。Runner 启动后应使用 `opus`。
2.  **Validation**: 配置文件中包含非法字段或类型错误（如 `timeout: "fast"`），启动应报错并提示具体位置。
3.  **Default Fallback**: 不提供任何配置文件，Runner 应使用默认值（如 `sonnet`, `300s`）成功启动。
