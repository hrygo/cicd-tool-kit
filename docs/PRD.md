# cicd-ai-toolkit - 产品需求文档 (PRD)

> 基于 Claude Code Headerless 模式的可插拔 CI/CD 工具集

## 项目概述

| 属性         | 值                                                       |
| ------------ | -------------------------------------------------------- |
| **项目名称** | `cicd-ai-toolkit`                                        |
| **目标**     | 基于 Claude Code 构建可插拔 CI/CD 工具集，提升效能与质量 |
| **定位**     | 开源项目，面向 DevOps 和工程团队                         |
| **开发者**   | 独立开发者                                               |
| **当前状态** | **Ready for Development** (PRD v1.0 Final)               |

---

## 第一部分：背景与技术架构

### 1.1 Claude Code 核心能力

#### 基本命令
| 命令                            | 说明                | CI/CD 相关性           |
| ------------------------------- | ------------------- | ---------------------- |
| `claude -p "query"`             | Headless/Print 模式 | ⭐⭐⭐ 核心 CI/CD 集成点  |
| `claude -c`                     | 继续最近会话        | 调试和恢复             |
| `cat file \| claude -p "query"` | 管道输入处理        | ⭐⭐⭐ 日志分析、数据处理 |
| `claude --agents '{...}'`       | 动态定义子 Agent    | ⭐⭐ 任务专业化          |
| `claude -r "<session>"`         | 恢复会话            | 长任务恢复             |

#### 关键 CLI Flags (CI/CD 场景)
| Flag                             | 说明                 | CI/CD 用途       |
| -------------------------------- | -------------------- | ---------------- |
| `-p, --print`                    | 非交互模式           | ⭐ CI/CD 集成核心 |
| `--output-format stream-json`    | 流式 JSON 输出       | ⭐ 结果解析       |
| `--allowedTools`                 | 允许的工具白名单     | ⭐ 安全控制       |
| `--dangerously-skip-permissions` | 跳过权限确认         | 自动化场景       |
| `--max-turns`                    | 最大执行轮数         | 控制执行深度     |
| `--max-budget-usd`               | 最大 API 花费        | ⭐ 成本控制       |
| `--system-prompt-file`           | 从文件加载系统提示词 | ⭐ 提示词版本控制 |
| `--append-system-prompt`         | 追加系统提示词       | 自定义行为       |
| `--agents`                       | 自定义子 Agent       | 任务专业化       |
| `--json-schema`                  | JSON Schema 输出验证 | ⭐ 结构化输出     |

#### Hooks 机制
- **Setup Hooks**: 初始化时运行
- **User Prompt Hooks**: 用户提示触发
- **Pre-commit/Post-commit Hooks**: Git 操作钩子
- **Tool Hooks**: 工具调用钩子

#### 可编程接口
1. **SDK 输出格式**: `text`, `json`, `stream-json`
2. **输入格式**: `text`, `stream-json`
3. **JSON Schema**: 结构化输出验证
4. **MCP (Model Context Protocol)**: 扩展能力

---

### 1.2 Claude Agent Skills 设计模式

#### Runner + Skills 架构
```
┌─────────────────────────────────────────────────┐
│              Runner (Go/Platform Adapter)       │
│  - CI Context Injection (Diffs, Logs)           │
│  - Identity & Auth (GitHub/GitLab)              │
│  - Reporting (Comments, Checks)                 │
└─────────────────────────────────────────────────┘
         │ (Executes)
    ┌────▼──────────────────────────────┐
    │      Claude Code (Headless)       │
    └────┬────────────────────────┬─────┘
         │ (Loads)                │ (Loads)
    ┌────▼───────┐          ┌─────▼──────┐
    │ Skill:     │          │ Skill:     │
    │ CodeReview │          │ TestGen    │
    └────────────┘          └────────────┘
```

#### 关键优势
1. **原生集成**: 直接利用 Claude Code 的工具发现和执行能力
2. **开发简便**: Skill 仅需 Markdown 定义和脚本，无需编译二进制
3. **生态兼容**: 可直接使用 Anthropic 官方或社区的 Skills


#### 可插拔架构关键要素
1. **插件注册与发现**
2. **生命周期管理**
3. **依赖注入**
4. **通信总线**
5. **配置驱动**

#### GitHub Actions 参考模式
- **Composite Actions**: 步骤组合
- **Reusable Workflows**: 工作流复用
- **Custom Actions**: 自定义动作

---

### 1.3 AI 赋能 CI/CD 的应用场景

| 场景              | 描述                     | 价值       |
| ----------------- | ------------------------ | ---------- |
| 智能代码审查      | 超越传统 Lint 的主观审查 | 质量提升   |
| 自动化 Issue 分类 | 智能标签和优先级         | 效率提升   |
| 变更总结          | 自动生成 Changelog       | 沟通效率   |
| 测试用例生成      | 基于代码变更生成测试     | 覆盖率提升 |
| 故障诊断          | 日志分析和根因定位       | MTTR 降低  |
| 安全扫描          | 语义级安全分析           | 安全性提升 |
| 文档生成          | API 文档、架构图         | 维护效率   |

---

### 1.4 行业趋势 (2026 现状)

#### Autonomous Delivery Maturity Model (2026)

| 等级   | 名称                  | 特征                                                  | 典型工具 (2026)                          |
| :----- | :-------------------- | :---------------------------------------------------- | :--------------------------------------- |
| **L1** | **Task Automation**   | 脚本化自动化，人类定义所有步骤                        | Jenkins Pipelines, Legacy GitHub Actions |
| **L2** | **Semi-Autonomous**   | Agent 执行特定任务 (Review/Test)，人类编排            | GitHub Copilot Workspace, GitLab Duo     |
| **L3** | **Highly Autonomous** | Agent 自主决策分支合并、回滚，人类仅处理异常          | **cicd-ai-toolkit**, Devin for DevOps    |
| **L4** | **Fully Autonomous**  | 多 Agent 协作 (Swarm)，自主优化架构与成本，零接触交付 | Proprietary Enterprise Brains            |

- **Agentic DevOps**: 从 "辅助编码" 进化为 "自主运维"。Agent 不仅写代码，还负责测试、部署和故障修复的闭环。
- **Context Ops**: 上下文管理成为核心竞争力。能够高效组织全库百万行代码上下文的 Toolchain 才是赢家。


### 1.5 Claude Code 核心集成模式


#### Context Injection Strategy (上下文注入策略)
为了在无头模式(Headless Mode)下高效运行，Runner 采用以下策略注入上下文：

1.  **Project Context (`CLAUDE.md`)**:
    *   在仓库根目录生成/维护 `CLAUDE.md`。
    *   包含：项目架构简述、代码风格指南、关键目录说明。
    *   Claude Code 会自动读取此文件作为基础上下文。

2.  **Task Context (Stdin/Prompt)**:
    *   通过管道 (`|`) 将动态数据（Git Diff, Linter Report）传入。
    *   示例：`git diff main...feature | claude -p "Execute code-reviewer skill"`。

3.  **Explicit File Context (`@`)**:
    *   对于关键文件，在 Prompt 中显式引用。
    *   示例：`claude -p "Review changes in @src/main.go based on @skills/review/SKILL.md"`。

#### Output Parsing Strategy (输出解析策略)
由于 Claude Code 目前主要面向交互式使用，CLI 输出可能包含 "Thinking" 过程。
为了获得可靠的机器可读输出：
1.  **JSON Schema Enforcing**: 在 Prompt 中强制要求 JSON 格式。
2.  **Output Extraction**: Runner 需解析 stdout，提取 JSON 代码块 (` ```json ... ``` `)。
3.  **Stream Processing**: 监听 `stream-json` 格式（如未来支持）或逐行扫描标记。

#### Smart Chunking & Context Pruning (智能分片与上下文剪枝)
针对大型 PR (Diff > 1000行 或 Token > 32k) 的应对策略：
1.  **Context Pruning**: 自动移除 `*.lock`, `package-lock.json`, `vendor/`, `dist/` 等非源码文件。
2.  **Logical Chunking**: 按文件或模块粒度将 Diff 切分为多个独立的小 Context。
3.  **Batch Analysis**: Runner 串行或并行（取决于 Rate Limit）提交多个 Claude Session，最后汇总 Result。


---

## 第二部分：产品需求 (已确认)

### 2.1 核心需求确认

#### 支持平台
*   **GitHub Actions** (Tier 1)
*   **Gitee Enterprise** (Tier 1 - P0)
*   **GitLab CI/CD** (Tier 2)
*   **Jenkins** (Legacy Support)
*   **Multi-Cloud / Hybrid** (Architecture Ready)


#### 核心痛点 (按优先级)

| 痛点领域       | 具体需求                                 | AI 赋能点              |
| -------------- | ---------------------------------------- | ---------------------- |
| **代码质量**   | 性能问题、安全漏洞、逻辑缺陷、灾难性设计 | 深度语义分析、架构审查 |
| **测试效率**   | 测试生成、覆盖率优化、智能选择           | AI 测试用例生成        |
| **交付速度**   | 自动化重复任务、快速反馈                 | 流程自动化、智能决策   |
| **运维稳定性** | 故障诊断、日志分析、根因定位             | AI 故障分析            |

> **结论**: 四大领域全覆盖，代码质量深度分析是重点

#### 技术栈选择
- **Runner 架构**: Go (核心运行器) + Claude Code (CLI)
- **Skills 定义**: Markdown (标准) + Python/Bash (脚本)

#### 产品定位
- **开源项目**: 面向社区，接受外部贡献


---

### 2.2 功能模块设计


#### Phase 1 (MVP) - 核心技能 (Skills)

| 技能名称            | 功能描述                     | 优先级 | 形式                          |
| ------------------- | ---------------------------- | ------ | ----------------------------- |
| **Code Reviewer**   | 性能、安全、逻辑、架构分析   | P0     | Skill (Prompt + Linter Tools) |
| **Test Generator**  | 基于代码变更生成测试用例     | P0     | Skill (Prompt + Test Runner)  |
| **Change Analyzer** | PR 总结、影响分析、风险评分  | P1     | Skill (Prompt + Git Stats)    |
| **Log Analyzer**    | 日志分析、异常检测、根因定位 | P1     | Skill (Prompt + Log Parser)   |

#### Phase 2 - 扩展技能

| 技能名称             | 功能描述                    | 优先级 | 形式                                |
| -------------------- | --------------------------- | ------ | ----------------------------------- |
| **Security Scanner** | 语义级安全分析、供应链检查  | P1     | Skill (Integration with Trivy/Snyk) |
| **Perf Auditor**     | 性能回归检测、优化建议      | P2     | Skill (Integration with k6/JMeter)  |
| **Doc Generator**    | API 文档、架构图、Changelog | P2     | Skill (Mermaid/OpenAPI tools)       |
| **Compliance Check** | IaC 审查、策略验证          | P2     | Skill (OPA/TFSec)                   |


#### 可插拔架构 (Agent Skills)
```
┌─────────────────────────────────────────────────────────────────┐
│                      cicd-ai-toolkit                             │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Runner (Go)                          │   │
│  │  - Context Builder (Git/Logs)                           │   │
│  │  - Platform API Client (GitHub/GitLab)                  │   │
│  │  - Claude Session Manager                               │   │
│  │  - Result Reporter                                      │   │
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
│  │   (linter.py)   │                     │   (jest-run.sh) │   │
│  └─────────────────┘                     └─────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 第三部分：技术方案

### 3.1 项目信息

| 项目属性     | 值                                     |
| ------------ | -------------------------------------- |
| **项目名称** | `cicd-ai-toolkit`                      |
| **定位**     | Claude Agents 的企业级 CI/CD 运行器    |
| **核心技术** | Go (Runner) + Markdown/Python (Skills) |
| **分发方式** | 容器镜像 + 单二进制 Runner             |

### 3.2 MVP 范围 (Phase 1)

#### 功能范围
- ✅ **深度代码审查** (核心 Skill)
- ✅ **变更分析** (核心 Skill)

#### 平台支持
- ✅ **GitHub Actions** (优先): 利用 Composite Actions 分发。
- ✅ **Gitee Enterprise** (企业级 P0):
    - **Private Agent Marketplace**: 适配 Gitee 企业版私有插件市场规范。
    - **Gitee Go集成**: 提供适配 Gitee Go 流水线的原生 Runner 插件。
    - **One-Click Install**: 提供 `curl ... | bash` 脚本，支持在私有 Runner 机器上一键部署 `cicd-ai-toolkit` 二进制及依赖。
- 🔄 **GitLab CI** (Phase 2): 适配 Runner。


### 3.3 架构设计 (Multi-Platform Adapter)

#### Gitee Enterprise 适配策略
由于 Gitee Go (Gitea Actions) 底层基于 `act_runner`，与 GitHub Actions 高度兼容：
1.  **Action 兼容**: 直接复用 `action.yml` 定义，支持 Gitee Go 原生加载。
2.  **API 适配**: Runner 内置 `GiteeClient` (基于 OAuth2 + API v5)，处理企业版特有的 `enterprises/{id}` 鉴权。
3.  **Webhook 统一**: Runner 统一标准化 GitHub `pull_request_review` 和 Gitee `NoteEvent` (评论事件) 为内部 `ReviewEvent`。


### 3.4 技术栈详情

| 层级             | 技术选择               | 说明                                    |
| ---------------- | ---------------------- | --------------------------------------- |
| **Runner**       | Go 1.21+               | 负责环境准备、认证、结果回传            |
| **Intelligence** | Claude Code            | 负责推理、工具调用                      |
| **Skills**       | Markdown + Python/Bash | 定义能力的标准格式                      |
| **配置**         | YAML                   | 定义启用哪些 Skills 及其参数            |
| **容器**         | Docker/OCI             | 包含 Runner + Claude Code + 预置 Skills |


### 3.5 目录结构设计

```
cicd-ai-toolkit/
├── cmd/                    # Go Runner 入口
│   └── cicd-runner/       # 主命令
├── pkg/                    # Go 核心库
│   ├── runner/            # 运行器逻辑
│   ├── platform/          # 平台适配器 (GitHub/GitLab)
│   ├── build-context/     # 上下文构建 (Diff/Tree)
│   └── claude/            # Claude 进程管理
├── skills/                 # 内置 Skills (标准结构)
│   ├── code-reviewer/
│   │   ├── SKILL.md       # 技能定义
│   │   └── scripts/       # 辅助脚本
│   ├── test-generator/
│   │   ├── SKILL.md
│   │   └── scripts/
│   └── change-analyzer/
│       └── SKILL.md
├── configs/                # 配置示例
├── .github/                # GitHub Actions 集成
│   └── workflows/
├── Dockerfile
├── go.mod
└── README.md
```

### 3.6 配置文件格式

```yaml
# .cicd-ai-toolkit.yaml
version: "1.0"

# Claude Code 配置
claude:
  model: "sonnet"           # sonnet | opus | haiku
  max_budget_usd: 5.0       # 成本控制
  max_turns: 10             # 最大轮数
  timeout: 300s             # 超时时间

# 技能 (Skills) 配置
skills:
  - name: code-reviewer
    path: ./skills/code-reviewer  # 本地路径或 git url
    enabled: true
    config:
      severity_threshold: "warning"

  - name: change-analyzer
    enabled: true


# 平台配置
platform:
  github:
    post_comment: true
    fail_on_error: false
```

### 3.7 GitHub Actions 集成示例

```yaml
# .github/workflows/ai-review.yml
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
          run_skills: "code-reviewer,change-analyzer"
          config: .cicd-ai-toolkit.yaml
          github_token: ${{ secrets.GITHUB_TOKEN }}
```

### 3.8 Gitee Go (Gitea Actions) 集成示例

Gitee Go 与 GitHub Actions 语法高度兼容，但需注意 Token 和 Runner 标记：

```yaml
# .gitee/workflows/ai-review.yml
name: AI Code Review
on: [pull_request]

jobs:
  ai-review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        
      - name: AI Code Review
        uses: cicd-ai-toolkit/action@v1
        with:
          run_skills: "code-reviewer"
          config: .cicd-ai-toolkit.yaml
          # Gitee Go 注入的 Token
          gitee_token: ${{ secrets.GITEE_TOKEN }}
          # 企业版 API 端点
          gitee_api_url: "https://api.gitee.com/v5"
```

---

## 第四部分：里程碑规划

### Phase 1: MVP (预计 4-6 周)

**目标**: 可用的代码审查 + 变更分析工具

| 任务          | 说明                                  | 产出              |
| ------------- | ------------------------------------- | ----------------- |
| 核心 Runner   | 上下文构建、平台适配、Claude 进程管理 | Go Runner         |
| Skills 移植   | 将 Prompt 移植为 `SKILL.md` 标准格式  | Skill Definitions |
| GitHub Action | 封装 Runner 为 Action                 | action.yml        |
| GitHub Action | 官方 Action 集成                      | action.yml        |
| 文档          | README、配置示例、快速开始            | Docs              |

### Phase 2: GitLab 支持与扩展 (预计 6-8 周)

| 任务           | 说明           |
| -------------- | -------------- |
| GitLab CI 适配 | 平台抽象层实现 |
| GitLab Bot     | MR 评论集成    |


### Phase 2: 扩展 (预计 6-8 周)

| 功能         | 说明            |
| ------------ | --------------- |
| 测试生成插件 | AI 测试用例生成 |
| 安全深度扫描 | 语义级安全分析  |
| 性能基准     | 性能回归检测    |
| Jenkins 支持 | Jenkins 插件    |

### Phase 3: 企业化 (预计 4-6 周)

| 功能       | 说明             |
| ---------- | ---------------- |
| 监控可观测 | Metrics、Tracing |
| 权限安全   | RBAC、审计日志   |
| 性能优化   | 并发处理、缓存   |

---

## 第五部分：非功能性需求

### 5.1 性能要求

| 指标         | 目标值               |
| ------------ | -------------------- |
| 单次分析耗时 | < 60s (中等 PR，P90) |
| 内存占用     | < 512MB (CLI)        |
| 冷启动时间   | < 5s                 |
| 缓存命中率   | > 40% (目标)         |

#### Result Caching Strategy (结果缓存策略)
为了降低成本和延迟，Runner 必须实现两级缓存：
1.  **File-Level Cache**: 计算 `Hash(FileContent + SkillInstruction)`。如果文件未变更且 Skill 定义未变，直接返回上次的 Issues。
2.  **Global Cache**: 存储在 CI 系统的 Cache 挂载卷或 S3 中 (`.cicd-ai-cache/`)。

### 5.2 可靠性要求

| 指标     | 目标值                         |
| -------- | ------------------------------ |
| 可用性   | 99.5% (本地运行)               |
| 降级策略 | Claude 不可用时跳过，不阻塞 CI |
| 幂等性   | 重复运行结果一致               |

### 5.3 安全要求

| 要求         | 说明                                                                                                                                                   |
| ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| API Key      | 从环境变量读取，不写入配置                                                                                                                             |
| 代码隐私     | 仅发送 diff，不发送完整代码库                                                                                                                          |
| 审计日志     | 记录所有 API 调用                                                                                                                                      |
| 输出验证     | JSON Schema 验证所有输出                                                                                                                               |
| **沙箱隔离** | **Strict Sandboxing**: Runner 运行在 Read-Only RootFS 容器中，仅挂载 `/workspace` 和 `/tmp`。网络层仅允许访问 Anthropic API 和内部 Git/Artifact 仓库。 |

---

## 第六部分：开源策略

### 6.1 许可证
- **主项目**: Apache 2.0
- **文档**: CC BY 4.0

### 6.2 社区建设
- GitHub Issues 讨论和追踪
- Contributing Guide
- RFC 流程 (重大变更)
- Roadmap 公开

### 6.3 贡献与生态激励 (2026 模型)
- **Contributors**: 传统的代码贡献。
- **Workflow-as-a-Service**: 允许社区开发者将复杂的 Skill 组合打包成 Workflows (如 "Java Legacy Migration Agent") 并在企业私有市场中通过 License 变现。
- **Skill Marketplace**: 官方维护 Skills 索引，支持 "Verified Skills" 认证。


---

## 第七部分：竞品分析与差异化

### 7.1 竞品对比

| 项目                            | 类型 | 技术栈     | Stars | 特点                       |
| ------------------------------- | ---- | ---------- | ----- | -------------------------- |
| **claude-code-security-review** | 官方 | Python     | 2.9k  | 专注安全审查，单点解决方案 |
| **pr-agent (qodo-ai)**          | 开源 | Python     | 10k+  | 功能全面，PR 全流程支持    |
| **claude-code-action**          | 官方 | TypeScript | -     | 通用 GitHub Action         |

### 7.2 claude-code-security-review 分析

**优点:**
- 官方出品，与 Claude Code 深度集成
- 专注安全领域，检测能力专业
- 支持自定义过滤指令
- MIT 许可证

**可借鉴:**
- False Positive 过滤机制
- 自定义扫描指令配置
- `/security-review` slash command 模式

**差距:**
- 仅专注安全，无通用代码审查
- 仅支持 GitHub Actions
- 无插件架构

### 7.3 pr-agent (qodo-ai) 分析

**优点:**
- 功能丰富 (`/review`, `/improve`, `/describe`, `/ask` 等)
- 多平台支持 (GitHub, GitLab, BitBucket, Azure DevOps)
- PR Compression 策略处理大 PR
- 高度可配置 (TOML 配置)
- CLI + Webhook + Action 多种部署方式

**可借鉴:**
- PR 压缩策略 (处理 token 限制)
- 平台抽象层设计
- 配置驱动架构
- Slash command 设计

**差距:**
- 使用通用 LLM API，非 Claude Code 原生
- Python 单语言，性能和分发不如 Go
- 配置复杂度高

### 7.4 cicd-ai-toolkit 差异化定位

| 维度         | cicd-ai-toolkit    | pr-agent         | claude-code-security-review |
| ------------ | ------------------ | ---------------- | --------------------------- |
| **核心技术** | Claude Code 原生   | 通用 LLM API     | Claude Code 原生            |
| **架构**     | Runner + Skills    | 单体架构         | 单体架构                    |
| **性能**     | Go 高性能 Runner   | Python           | Python                      |
| **分发**     | 单二进制 + 容器    | Python 包 / 容器 | 容器                        |
| **扩展性**   | Skills (Markdown)  | 配置定制         | 配置定制                    |
| **功能范围** | 质量 + 测试 + 运维 | PR 全流程        | 安全专项                    |

**核心差异化:**

1.  **Claude Code 原生集成**
    *   利用 Claude Code 的工具生态 (Bash, Edit, Read, MCP)
    *   Headless 模式深度优化
    *   标准 Agent Skills 支持

2.  **Agent Skills 架构**
    *   社区驱动的技能生态 (`SKILL.md`)
    *   技能定义标准化，易于编写和复用
    *   支持热插拔和动态加载

3.  **Go Runner + Native Skills**
    *   Go 负责稳健的流程控制 (CI/CD)
    *   Claude 负责智能决策与执行
    *   无缝集成 GitHub/GitLab

4.  **深度代码分析**
    *   不只是 PR review
    *   性能问题检测
    *   灾难性设计预警
    *   架构层面审查

### 7.5 生态复用策略

**外部生态复用:**
- PR 压缩策略 (pr-agent)
- False Positive 过滤 (claude-code-security-review)
- 配置文件设计 (YAML/TOML)
- 平台适配器抽象


### 7.5 生态复用策略

**直接复用:**
- PR 压缩策略 (pr-agent)
- False Positive 过滤 (claude-code-security-review)
- 配置文件设计 (YAML/TOML)
- 平台适配器抽象

**核心创新领域:**
- Claude Code 原生集成方案
- Agent Skills 编排引擎 (Runner)
- 深度代码分析能力


---

## 第八部分：技能架构设计详细方案

### 8.1 技能架构：Standard Agent Skills

**选择理由:**
- **原生支持**: Claude Code 能够直接理解和加载 `SKILL.md`
- **低维护成本**: 只要维护 Text/Markdown 定义，无需二进制兼容性
- **灵活性**: 随时可以热加载/卸载技能

### 8.2 Runner 流程
1.  **Init**: 加载配置，识别目标平台 (GitHub/GitLab)
2.  **Context**: 从 Git 获取 diff，从 Linter 获取报告
3.  **Session**: 启动 `claude` 子进程，注入 Context
4.  **Execute**: 指示 Claude 加载指定 Skills (如 `/review-code`)
5.  **Report**: 解析 Claude 的 JSON 输出，调用平台 API 发表评论

#### Async Execution Flow (异步执行流)
为避免阻塞 CI Job 和应对 API 延迟：
1.  **Start**: CI 触发，Runner 立即响应 "Analysis Pending" 状态检查。
2.  **Process**: Runner 后台运行 Claude Code 进行分析（若环境允许）或提交任务到独立 Worker。
3.  **Callback**: 分析完成后，通过 Webhook 或直接 API 调用的方式，回写评论和 Check Status。
4.  **Timeout**: 设置硬超时（如 10分钟），防止僵尸任务。

### 8.3 技能定义标准 (SKILL.md)

**文件位置**: `skills/<skill-name>/SKILL.md`

**内容规范**:
````markdown
---
name: code-reviewer
description: Analyzes code changes using advanced reasoning.
options:
  thinking:
    budget_tokens: 4096  # Enable Chain of Thought for deep analysis
  tools:
    - grep
    - ls
---

# Code Review Process

You are an expert code reviewer acting as a quality gate.

## 1. Analysis Scope (Deep Reasoning)
Review the provided code diffs. BEFORE generating findings, output a `<thinking>` block to analyze:
1. **Architectural Impact**: Does this change violate layer boundaries?
2. **Security & Data Flow**: Trace user input to database sinks.
3. **Concurrency**: Check for race conditions in new goroutines/async functions.

## 2. Context Handling
- The code changes are provided via standard input (stdin) or referenced files.
- You must ignore `vendor/` directories and auto-generated files (e.g., `*.pb.go`).

## 3. Output Format
Report findings in the following XML-wrapped JSON format ONLY (to ensure robust parsing):

```xml
<thinking>
[Step-by-step reasoning goes here...]
</thinking>
<json>
{
  "issues": [
    {
      "severity": "critical | high | medium | low",
      "file": "string",
      "line": "number",
      "column": "number",
      "category": "security | performance | logic | style",
      "message": "string",
      "suggestion": "string"
    }
  ]
}
</json>
```
````



### 8.4 Runner 实现细节 (Go)

```go
// Runner Implementation of Context Injection
func (r *Runner) Review(ctx context.Context, diff string) error {
    // 1. Build Command
    // Use --print for non-interactive mode
    // Use --dangerously-skip-permissions to avoid prompts in CI
    args := []string{
        "-p", "Execute code-reviewer skill. Input diff is provided via stdin.",
        "--dangerously-skip-permissions", 
    }

    cmd := exec.CommandContext(ctx, "claude", args...)
    
    // 2. Inject Context via Stdin (Best Practice for large diffs)
    cmd.Stdin = strings.NewReader(diff)
    
    // 3. Capture Output
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("claude execution failed: %v, output: %s", err, output)
    }

    // 4. Parse JSON from Markdown block
    findings, err := r.extractJSON(output)
    if err != nil {
        return err
    }
    
    // 5. Post to Platform
    return r.Platform.PostComment(findings)
}
```

### 8.5 配置文件规范

```yaml
# .cicd-ai-toolkit.yaml
version: "1.0"

# Claude Code 配置
claude:
  model: "sonnet"           # sonnet | opus | haiku
  max_budget_usd: 5.0       # 成本控制
  max_turns: 10             # 最大轮数
  timeout: 300s             # 超时时间

# 技能配置
skills:
  - name: code-reviewer
    path: ./skills/code-reviewer
    enabled: true
    config:
      severity_threshold: "warning"

  - name: change-analyzer
    enabled: true
    priority: 1

# 平台配置
platform:
  github:
    post_comment: true
    fail_on_error: false
    max_comment_length: 65536
    emoji_reactions: true

  gitee:
    api_url: "https://gitee.com/api/v5"
    post_comment: true
    enterprise_id: ""  # Optional

  gitlab:

    post_comment: true
    fail_on_error: false

# 全局配置
global:
  log_level: "info"          # debug | info | warn | error
  cache_dir: ".cicd-ai-cache"
```

---

## 第九部分：关键技术与最佳实践

### 9.1 MCP (Model Context Protocol) 集成策略

虽然 Claude Code 已经内置了文件系统和终端访问能力，但在 CI/CD 场景下，我们通过 **Dual-Layer MCP Strategy** 增强其能力：

**1. Infrastructure Context (Hosted by Runner)**
Runner (Go) 启动轻量级 MCP Server，提供 CI 环境信息：
- `get_env_info`: 获取当前 CI 运行环境 (GitHub Actions / Gitee Go context).
  - *Gitee Specific*: 自动探测 `GITEE_REPO_URL`, `GITEE_PULL_REQUEST_ID`.
- `get_secrets`: 安全地获取仅限 CI 使用的部署密钥 (不直接暴露给 Prompt).


**2. Domain Context (External MCP Servers)**
通过 Claude Code 的配置挂载外部 MCP Servers，获取更广泛的上下文：
- **Issue Tracker MCP** (Jira/Linear): 获取 PR 关联的需求描述、验收标准 user story。
  - *价值*: AI Reviewer 可以根据 "验收标准" 检查代码是否完成了功能，而不仅仅是检查代码错误。
- **Observability MCP** (Prometheus/Datadog): 获取相关服务的线上性能基线。
  - *价值*: 在 "Perf Auditor" 技能中，对比变更前后的性能预期。

```
┌─────────────────────────────────────────────────────────────┐
│                    Claude Code (Subprocess)                 │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                   MCP Client                         │   │
│  │  - [Internal] Runner MCP (CI Env, Safe Secrets)      │   │
│  │  - [External] Linear/Jira MCP (Requirements)         │   │
│  │  - [External] Datadog MCP (Performance Baseline)     │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```


### 9.2 Policy-as-Code for Agents (Agent 治理)

在 2026 年，为了安全地放权给 Autonomous Agents，必须使用 **Policy-as-Code** (PaC) 进行约束，而非简单的 Prompt 提示。

#### 1. Policy Engine Integration (OPA/Polar)
Runner 集成 Open Policy Agent (OPA) 或 Polar (Oso) 引擎，在 Agent 执行高危操作（如 Merge PR、Deploy）前进行拦截。

**示例 Policy (Rego - OPA):**
```rego
package cicd.agent.authz

# 禁止 Agent 在周五下午自动合并代码
deny[msg] {
    input.action == "merge_pr"
    input.agent_role == "autonomous"
    time.weekday(time.now_ns()) == "Friday"
    time.clock(time.now_ns())[0] >= 16
    msg := "Autonomous agents cannot merge on Friday afternoons"
}

# 强制要求高风险变更经过 Human Approval
deny[msg] {
    input.action == "deploy_prod"
    input.risk_score >= 8.0
    not input.human_approved
    msg := "High risk deployment requires human approval"
}
```

#### 2. Agent Identity & Sandboxing
- **Workload Identity**: 为每个 Agent 分配独立的非人类身份 (Non-human Identity)，而非共享 Admin Token。
- **Dynamic Sandboxing**: Agent 运行在临时的、网络隔离的容器沙箱中，防止 Supply Chain 攻击。
- **Prompt Injection Mitigation**: 即使在 `--dangerously-skip-permissions` 模式下，也必须通过系统层面的各个 Read-Only 挂载限制 Agent 对宿主机的修改能力。


### 9.3 Autonomous Quality Gates (自主质量门禁)

不再是简单的 PASS/FAIL，而是自主决策的门禁系统。

#### Adaptive Gates
Agent 根据代码变更的风险等级（Risk Score），动态调整门禁严格度：
- **Low Risk (Docs)**: 仅需拼写检查，自动 Merge。
- **High Risk (Auth/Payment)**: 触发 "Deep Verification" 模式，自动生成针对性模糊测试用例 (Fuzzing) 并运行，通过后才放行。


#### Fail-Fast 策略

```yaml
# .cicd-ai-toolkit/gates.yaml
quality_gates:
  - name: "critical-security"
    priority: 0
    fail_fast: true
    conditions:
      - category: "security"
        severity: ["critical", "high"]
    action: "block_merge"

  - name: "performance-regression"
    priority: 1
    fail_fast: false
    conditions:
      - category: "performance"
        severity: ["high"]
    action: "warning"
```

#### 关键指标

| 指标                      | 说明            | 目标  |
| ------------------------- | --------------- | ----- |
| **Pipeline Success Rate** | CI 流水线成功率 | > 95% |
| **User Acceptance Rate**  | AI 建议采纳率   | > 20% |
| **False Positive Rate**   | 误报率          | < 10% |
| **Execution Time**        | 分析耗时        | < 90s |

---

## 第十部分：开源项目运营

### 10.1 生态增长指标 (2026 目标)

| 指标                    | 说明                           | 目标  |
| ----------------------- | ------------------------------ | ----- |
| **Agent Installs**      | 被多少个 Agentic Workflow 引用 | 500+  |
| **Skill Forks**         | 社区二次开发的 Skill 变种数    | 100+  |
| **Autonomous Fix Rate** | 无需人类干预的修复比例         | > 30% |


### 10.2 推荐架构总结

**Go Runner + Native Skills**：
- **Runner (Go)**: 负责“脏活累活”（Git操作、API调用、进程管理、成本控制）。
- **Brain (Claude)**: 负责“思考与决策”（代码理解、模式识别、逻辑分析）。
- **Skills (Markdown)**: 负责“定义能力”（提示词工程、工具定义）。

这种架构最大程度降低了维护成本，同时最大化了 Claude 的原生能力。

### 10.3 成功案例参考 (2026 Benchmarks)

| 项目         | Stars | 增长策略               | 可借鉴点                                |
| ------------ | ----- | ---------------------- | --------------------------------------- |
| **Devin**    | -     | Agentic Engineer       | 自主规划、执行、验证的闭环能力          |
| **Renovate** | 16k+  | Autonomous Maintenance | 真正 L3 级别的自主依赖更新 (Auto-Merge) |
| **dagger**   | 12k+  | 跨平台流水线           | CI Runner 设计借鉴，"Programmatic CI"   |


### 10.4 附录：资料来源
- [Claude Code CLI Reference](https://code.claude.com/docs/en/cli-reference)
- [Anthropic Agent Skills](https://github.com/anthropic/agent-skills)
- [Model Context Protocol](https://modelcontextprotocol.io/)
