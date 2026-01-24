# SPEC-ECO-01: Skill Marketplace & Ecosystem

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24
**Covers**: PRD 6.3 (生态增长指标, Skill Marketplace)

## 1. 概述 (Overview)

Skill Marketplace 是社区分享和发现 AI Skills 的中心平台。通过 Marketplace，开发者可以发布、发现、安装高质量的 Skills，形成良性循环的生态系统。

## 2. 核心职责 (Core Responsibilities)

- **索引服务**: 维护可发现、可搜索的 Skill 索引
- **验证机制**: Verified Skills 认证
- **版本管理**: Skills 的语义化版本控制
- **安全扫描**: 自动化安全检查
- **使用统计**: 追踪下载量、星级评分

## 3. 详细设计 (Detailed Design)

### 3.1 架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                      Skill Marketplace                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Index Service                        │   │
│  │  - Skill Metadata Store                                │   │
│  │  - Search & Discovery                                  │   │
│  │  - Version Registry                                    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                          │                                     │
│  ┌───────────────────────┼─────────────────────────────────┐   │
│  │                       │                                 │   │
│  ▼                       ▼                                 │   │
│ ┌─────────────┐    ┌─────────────┐    ┌─────────────┐      │   │
│ │   GitHub    │    │  Registry   │    │  Scanner    │      │   │
│ │  (Source)   │    │   (OCI)     │    │ (Security)  │      │   │
│ └─────────────┘    └─────────────┘    └─────────────┘      │   │
└─────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      cicd-runner                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                 Skill Install CLI                        │   │
│  │  - cicd-runner skill install <name>                     │   │
│  │  - cicd-runner skill search <query>                     │   │
│  │  - cicd-runner skill update                             │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 Skill 索引格式 (Skill Index)

Skills 通过 `skill.yaml` 元数据文件注册：

```yaml
# skill.yaml (位于 Skill 仓库根目录)
apiVersion: "cicd.ai/v1"
kind: "Skill"

metadata:
  name: "code-reviewer"
  displayName: "AI Code Reviewer"
  description: "Deep code review with security and performance focus"
  version: "1.2.0"
  author: "cicd-ai-toolkit"
  license: "MIT"
  homepage: "https://github.com/cicd-ai-toolkit/skills/tree/main/code-reviewer"
  repository: "https://github.com/cicd-ai-toolkit/skills"
  keywords:
    - "review"
    - "security"
    - "performance"
    - "code-quality"
  categories:
    - "quality"
    - "security"

spec:
  # 支持的语言 (可选)
  languages:
    - "go"
    - "python"
    - "javascript"
    - "typescript"
    - "java"
    - "rust"

  # 所需的 Claude 模型版本
  claude:
    min_version: "3.5"
    recommended_model: "claude-3-5-sonnet-20241022"

  # MCP 依赖
  mcp_servers:
    - name: "linear"
      optional: true

  # 外部工具依赖
  tools:
    - name: "grep"
      required: true
    - name: "semgrep"
      optional: true

  # 输入/输出 Schema
  input_schema:
    type: "object"
    properties:
      diff:
        type: "string"
      severity_threshold:
        type: "string"
        enum: ["critical", "high", "medium", "low"]

  output_schema:
    type: "object"
    properties:
      issues:
        type: "array"

# 统计信息 (由 Marketplace 维护)
stats:
  downloads: 125000
  stars: 450
  last_updated: "2026-01-24T00:00:00Z"
  verified: true
```

### 3.3 Marketplace API (REST API)

#### GET /skills

列出所有可用 Skills：

```
GET /api/v1/skills?page=1&limit=20&category=security
```

```json
{
  "skills": [
    {
      "name": "code-reviewer",
      "version": "1.2.0",
      "description": "Deep code review...",
      "author": "cicd-ai-toolkit",
      "stats": {
        "downloads": 125000,
        "stars": 450
      },
      "verified": true
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 45
  }
}
```

#### GET /skills/{name}

获取 Skill 详情：

```
GET /api/v1/skills/code-reviewer
```

#### GET /search?q=query

搜索 Skills：

```
GET /api/v1/search?q=security+python&category=security
```

### 3.4 Skill 安装 CLI

```bash
# 安装 Skill
cicd-runner skill install code-reviewer

# 指定版本
cicd-runner skill install code-reviewer@1.2.0

# 从 GitHub 安装
cicd-runner skill install github.com/user/repo/skills/code-reviewer

# 列出已安装
cicd-runner skill list

# 更新所有
cicd-runner skill update --all

# 搜索
cicd-runner skill search security

# 查看 Skill 信息
cicd-runner skill info code-reviewer
```

### 3.5 Skill 存储位置 (Storage Layout)

```
~/.cicd-ai-toolkit/
├── skills/
│   ├── index.json          # 本地索引缓存
│   └── installed/
│       ├── code-reviewer/
│       │   ├── skill.yaml  # 元数据
│       │   ├── SKILL.md    # Prompt 定义
│       │   └── scripts/    # 辅助脚本
│       ├── test-generator/
│       └── security-scan/
```

### 3.6 Verified Skills 程序

**Verified Skills** 标志表示：

1. ✅ 代码审查通过
2. ✅ 通过安全扫描
3. ✅ 有完整文档
4. ✅ 有测试覆盖
5. ✅ 由官方或受信任作者维护
6. ✅ 遵循最佳实践

**申请流程**：

```bash
# 作者提交申请
cicd-runner skill verify-request code-reviewer

# Marketplace 运行验证
# - Security scan
# - Quality checks
# - Manual review (if needed)

# 通过后添加 verified: true
```

### 3.7 Skill 发布流程

#### 方式一: GitHub Repository

1. 创建 GitHub 仓库
2. 添加 `skill.yaml` 和 `SKILL.md`
3. 添加主题标签 `cicd-ai-skill`
4. Marketplace 自动发现并索引

#### 方式二: OCI Registry

```bash
# 打包 Skill
cicd-runner skill package ./my-skill -o my-skill.tar.gz

# 推送到 Registry
cicd-runner skill push ghcr.io/cicd-ai/skills/my-skill:1.0.0

# 从 Registry 安装
cicd-runner skill install ghcr.io/cicd-ai/skills/my-skill:1.0.0
```

### 3.8 Skill Rating & Review

**评分维度**：

| 维度 | 说明 | 权重 |
|------|------|------|
| **Quality** | Prompt 质量、代码规范 | 30% |
| **Effectiveness** | 实际使用效果 | 40% |
| **Documentation** | 文档完整性 | 15% |
| **Maintenance** | 更新频率、响应速度 | 15% |

**评分展示**：

```yaml
stats:
  rating:
    overall: 4.5
    counts: 125
    distribution:
      5: 89
      4: 24
      3: 8
      2: 3
      1: 1
    dimensions:
      quality: 4.7
      effectiveness: 4.4
      documentation: 4.3
      maintenance: 4.6
```

### 3.9 依赖管理 (Dependency Management)

Skill 可以声明依赖其他 Skills：

```yaml
# skill.yaml
dependencies:
  skills:
    - name: "common-prompt"
      version: ">=1.0.0"
    - name: "security-analyzer"
      optional: true
```

安装时自动解析依赖：

```bash
$ cicd-runner skill install my-skill
Installing dependencies:
  ├─ common-prompt@1.2.0 (required)
  ├─ security-analyzer@2.0.0 (optional) [SKIP]
  └─ my-skill@1.0.0
```

### 3.10 Marketplace 前端 (可选)

提供 Web UI 用于浏览和发现 Skills：

```
https://skills.cicd-ai-toolkit.com

Categories:
  ├─ Quality (code-reviewer, lint-analyzer)
  ├─ Security (security-scan, vulnerability-check)
  ├─ Testing (test-generator, coverage-analyzer)
  ├─ Documentation (doc-generator, api-docs)
  └─ Operations (log-analyzer, perf-auditor)
```

### 3.11 Marketplace Implementation (Go)

```go
// MarketplaceClient interacts with the Skill Marketplace
type MarketplaceClient struct {
    baseURL    string
    httpClient *http.Client
}

type SkillInfo struct {
    Name        string            `json:"name"`
    Version     string            `json:"version"`
    Description string            `json:"description"`
    Author      string            `json:"author"`
    License     string            `json:"license"`
    Keywords    []string          `json:"keywords"`
    Categories  []string          `json:"categories"`
    Stats       SkillStats        `json:"stats"`
    Verified    bool              `json:"verified"`
    InstallURL  string            `json:"install_url"`
}

type SkillStats struct {
    Downloads int     `json:"downloads"`
    Stars     int     `json:"stars"`
    Rating    float64 `json:"rating"`
}

func NewMarketplaceClient(baseURL string) *MarketplaceClient {
    return &MarketplaceClient{
        baseURL:    strings.TrimSuffix(baseURL, "/"),
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
}

func (c *MarketplaceClient) Search(ctx context.Context, query string, category string) ([]SkillInfo, error) {
    url := fmt.Sprintf("%s/api/v1/search?q=%s", c.baseURL, url.QueryEscape(query))
    if category != "" {
        url += "&category=" + category
    }

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Skills []SkillInfo `json:"skills"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return result.Skills, nil
}

func (c *MarketplaceClient) GetSkill(ctx context.Context, name string) (*SkillInfo, error) {
    url := fmt.Sprintf("%s/api/v1/skills/%s", c.baseURL, name)

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var skill SkillInfo
    if err := json.NewDecoder(resp.Body).Decode(&skill); err != nil {
        return nil, err
    }

    return &skill, nil
}

func (c *MarketplaceClient) Download(ctx context.Context, name, version string, dest string) error {
    url := fmt.Sprintf("%s/api/v1/skills/%s/%s/download", c.baseURL, name, version)

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("download failed: %s", resp.Status)
    }

    // Extract tarball to dest
    return tar.Extract(resp.Body, dest)
}
```

### 3.12 离线模式 (Offline Mode)

对于企业隔离环境，支持离线安装：

```bash
# 导出 Skill
cicd-runner skill export code-reviewer -o code-reviewer.tar.gz

# 传输到隔离环境
# ...

# 离线安装
cicd-runner skill install ./code-reviewer.tar.gz --offline
```

## 4. 依赖关系 (Dependencies)

- **Related**: [SPEC-SKILL-01](./SPEC-SKILL-01-Skill_Definition.md) - Skill 定义标准
- **Related**: [SPEC-DIST-01](./SPEC-DIST-01-Distribution.md) - 分发机制

## 5. 验收标准 (Acceptance Criteria)

1. **搜索**: 能通过关键词搜索到相关 Skills
2. **安装**: 能成功安装指定 Skill 到本地
3. **版本管理**: 能安装特定版本的 Skill
4. **依赖解析**: 能自动安装 Skill 依赖
5. **更新**: 能检查并更新已安装 Skills
6. **验证**: Verified Skills 有明显标识
7. **离线**: 支持离线安装 Skills
8. **安全**: 安装前显示 Skill 的权限要求和风险提示

## 6. 社区指标 (Community Metrics)

| 指标 | 目标 | 说明 |
|------|------|------|
| **Total Skills** | 100+ | Marketplace 中的 Skills 数量 |
| **Verified Skills** | 20+ | 官方验证的 Skills |
| **Active Authors** | 50+ | 活跃的 Skill 作者 |
| **Monthly Downloads** | 10K+ | 每月下载次数 |
| **Community Forks** | 200+ | 社区二次开发的变种 |

## 7. 安全考虑

1. **沙箱**: Skills 在沙箱中运行
2. **代码审查**: 所有 Skills 提交前经过审查
3. **签名验证**: Verified Skills 有数字签名
4. **权限声明**: Skill 必须声明所需权限
5. **用户评分**: 低评分 Skills 显示警告

## 8. Workflow-as-a-Service (WaaS) 商业模式

**Covers**: PRD 6.3 (Workflow-as-a-Service), 生态激励 (2026 模型)

### 8.1 概述

Workflow-as-a-Service 允许社区开发者将复杂的 Skill 组合打包成 "Workflows"（如 "Java Legacy Migration Agent"、"Full Stack Security Auditor"），并在私有或公开市场中通过 License 变现。

### 8.2 Workflow 定义

Workflow 是多个 Skills 和配置的预组合，解决特定的垂直场景：

```yaml
# workflow.yaml
apiVersion: "cicd.ai/v1"
kind: "Workflow"

metadata:
  name: "java-legacy-migrator"
  displayName: "Java Legacy Code Migration Assistant"
  version: "1.0.0"
  author: "enterprise-tools"
  license: "COMMERCIAL"
  pricing:
    model: "subscription"
    tier: "premium"

spec:
  description: "Automated migration assistant for Java legacy codebases"

  # Required Skills (自动安装)
  skills:
    - name: "code-analyzer"
      version: ">=1.0.0"
      config:
        language: "java"
        legacy_patterns: true
    - name: "test-generator"
      version: ">=2.0.0"
      config:
        framework: "junit"
    - name: "refactoring-assistant"
      version: ">=1.0.0"

  # Workflow Steps (编排逻辑)
  steps:
    - name: "analyze-legacy"
      skill: "code-analyzer"
      output: "legacy_report.json"

    - name: "generate-tests"
      skill: "test-generator"
      input:
        files_from: "analyze-legacy"
      condition: "${{ legacy_report.has_legacy_code == true }}"

    - name: "suggest-refactors"
      skill: "refactoring-assistant"
      input:
        analysis: "legacy_report.json"

  # MCP Dependencies
  mcp_servers:
    - name: "sonarqube"
      required: true

  # Platform Integration
  platforms:
    - github
    - gitlab
```

### 8.3 定价模型

#### 8.3.1 定价层级

| 层级 | 价格 | 功能 |
|------|------|------|
| **Free** | $0 | 社区 Skills，基础支持 |
| **Personal** | $9/月 | 全部 Skills，优先支持 |
| **Team** | $49/月 | 5 用户，共享配额，团队协作 |
| **Enterprise** | $199/月 | 无限用户，私有 Workflow，SLA |
| **Custom Workflow** | 按议 | Workflow 作者定价 |

#### 8.3.2 Workflow 收益分成

社区作者创建付费 Workflow 可获得收益分成：

```
收益分配 (Workflow 销售价 $100)
├── Marketplace (30%)     $30  - 平台运营、基础设施
├── Payment Processing (5%)  $5   - Stripe/PayPal 费用
└── Author (65%)            $65  - Workflow 作者
```

### 8.4 License 类型

| License | 说明 | 可转售 | 可修改 | 分成比例 |
|---------|------|--------|--------|----------|
| **MIT/Apache-2.0** | 开源，免费 | ✅ | ✅ | 0% |
| **Community** | 社区使用，免费 | ❌ | ❌ | 0% |
| **Commercial** | 商业使用需付费 | ❌ | ❌ | 作者定价 |
| **Enterprise** | 企业级，含 SLA | ❌ | ❌ | 作者定价 |
| **Custom** | 自定义条款 | 协商 | 协商 | 协商 |

### 8.5 工作流发布流程

```go
type WorkflowPublisher struct {
    client     *MarketplaceClient
    signer     *SignatureSigner
    scanner    *SecurityScanner
}

func (wp *WorkflowPublisher) Publish(wf *Workflow) (*PublishResult, error) {
    // 1. Validate workflow
    if err := wp.validate(wf); err != nil {
        return nil, err
    }

    // 2. Security scan
    report, err := wp.scanner.Scan(wf)
    if err != nil {
        return nil, err
    }

    if report.HasCriticalIssues {
        return nil, fmt.Errorf("workflow has critical security issues")
    }

    // 3. Sign workflow
    signature, err := wp.signer.Sign(wf)
    if err != nil {
        return nil, err
    }

    // 4. Upload to registry
    return wp.client.UploadWorkflow(wf, signature)
}
```

### 8.6 License Enforcement

```go
type LicenseChecker struct {
    licenses   *LicenseRegistry
    keys       *KeyManager
}

type LicenseKey struct {
    ID         string
    Workflow   string
    Customer   string
    ValidFrom  time.Time
    ValidUntil time.Time
    Tier       string
    UsageLimit *UsageLimit
    Features   []string
}

func (lc *LicenseChecker) Validate(key string, workflow *Workflow) (*LicenseStatus, error) {
    // Decrypt and validate key
    lic, err := lc.keys.Decrypt(key)
    if err != nil {
        return nil, ErrInvalidKey
    }

    // Check expiration
    if time.Now().After(lic.ValidUntil) {
        return nil, ErrExpiredKey
    }

    // Check workflow match
    if lic.Workflow != workflow.Name {
        return nil, ErrWrongWorkflow
    }

    // Check usage limits
    if lic.UsageLimit != nil {
        usage := lc.getUsage(lic.Customer, lic.Workflow)
        if usage > lic.UsageLimit.Max {
            return nil, ErrUsageExceeded
        }
    }

    return &LicenseStatus{
        Valid:     true,
        Tier:      lic.Tier,
        Features:  lic.Features,
    }, nil
}
```

### 8.7 支付集成

```go
type PaymentGateway interface {
    CreateCheckout(customer *Customer, items ...*CartItem) (*CheckoutURL, error)
    ProcessWebhook(payload []byte) (*PaymentEvent, error)
    GenerateLicense(customer *Customer, workflow *Workflow) (*LicenseKey, error)
}

type CartItem struct {
    Type       string // "workflow_subscription" or "workflow_purchase"
    WorkflowID string
    Tier       string // "monthly" or "yearly"
    Price      float64
}

type StripeGateway struct {
    client     *stripe.Client
    webhookKey string
}

func (sg *StripeGateway) CreateCheckout(customer *Customer, items ...*CartItem) (*CheckoutURL, error) {
    lineItems := []*stripe.CheckoutSessionLineItemParams{}

    for _, item := range items {
        priceID := sg.getPriceID(item.WorkflowID, item.Tier)
        lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
            Price:    stripe.String(priceID),
            Quantity: stripe.Int64(1),
        })
    }

    params := &stripe.CheckoutSessionParams{
        Customer:       stripe.String(customer.ID),
        PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
        LineItems:      lineItems,
        Mode:           stripe.String(string(stripe.ModeSubscription)),
        SuccessURL:     stripe.String("https://skills.cicd-ai-toolkit.com/success"),
        CancelURL:      stripe.String("https://skills.cicd-ai-toolkit.com/cancel"),
    }

    session, err := sg.checkoutSessions.New(params)
    if err != nil {
        return nil, err
    }

    return &CheckoutURL{URL: session.URL}, nil
}
```

### 8.8 使用计费

对于按使用量计费的 Workflow：

```go
type UsageMeter struct {
    registry  *UsageRegistry
    prometheus *PrometheusClient
}

type UsageEvent struct {
    Workflow   string
    Customer   string
    Timestamp  time.Time
    Quantity   int
    Unit       string // "runs", "tokens", "hours"
    Metadata   map[string]string
}

func (um *UsageMeter) Record(event *UsageEvent) error {
    // Record to registry
    if err := um.registry.Record(event); err != nil {
        return err
    }

    // Calculate cost
    cost := um.calculateCost(event)

    // Report to billing system
    return um.prometheus.RecordUsage(event, cost)
}

func (um *UsageMeter) calculateCost(event *UsageEvent) float64 {
    // Tiered pricing
    switch {
    case event.Quantity <= 100:
        return float64(event.Quantity) * 0.01
    case event.Quantity <= 1000:
        return 1.0 + float64(event.Quantity-100) * 0.005
    default:
        return 5.5 + float64(event.Quantity-1000) * 0.002
    }
}
```

### 8.9 企业私有 Marketplace

企业可以部署私有 Marketplace，内部发布和共享 Workflows：

```yaml
# private-marketplace-config.yaml
marketplace:
  type: "private"
  url: "https://skills.company.com"
  auth:
    method: "saml"
    provider: "okta"

  # Internal Workflows
  workflows:
    - name: "compliance-auditor"
      visibility: "internal"
      authorized: ["security-team", "compliance-team"]

    - name: "deploy-automation"
      visibility: "internal"
      authorized: ["devops-team"]
```

### 8.10 收益报告与支付

作者可以查看收益报告并请求提现：

```bash
# 查看收益报告
cicd-runner marketplace earnings --period monthly

# 请求提现
cicd-runner marketplace payout --method bank_transfer --amount 500
```

```go
type EarningsReport struct {
    Author         string
    Period         DateRange
    GrossRevenue   float64
    PlatformFee    float64
    PaymentFee     float64
    NetRevenue     float64
    Currency       string
    Transactions   []Transaction
    Status         string // "available", "pending", "paid"
}

func (er *EarningsReport) GenerateStatement() string {
    // Generate PDF statement for payout
}
```

### 8.11 Roadmap 更新

| 阶段 | 功能 | 时间 |
|------|------|------|
| **Phase 1** | 基础 CLI + GitHub 发现 | Q1 2026 |
| **Phase 2** | Marketplace API + Web UI | Q2 2026 |
| **Phase 3** | OCI Registry + Verified Skills | Q3 2026 |
| **Phase 4** | 付费 Skills + 收益分成 | Q4 2026 |
| **Phase 5** | Workflow-as-a-Service | Q1 2027 |
| **Phase 6** | 企业私有 Marketplace | Q2 2027 |
