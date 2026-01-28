# Gitee 企业版深度适配方案

## 版本: 1.0.0
## 日期: 2026-01-28

---

## 目录

1. [执行摘要](#1-执行摘要)
2. [Gitee 企业版能力分析](#2-gitee-企业版能力分析)
3. [当前实现差距分析](#3-当前实现差距分析)
4. [深度适配方案](#4-深度适配方案)
5. [实施路线图](#5-实施路线图)
6. [附录](#6-附录)

---

## 1. 执行摘要

### 1.1 调研结论

Gitee 企业版提供了完整的 API v5 生态和原生的 CI/CD 能力（Gitee Go），但当前 `cicd-ai-toolkit` 的 Gitee 适配存在功能缺失，需要深度适配以达到与 GitHub/GitLab 同等的集成水平。

### 1.2 关键发现

| 类别 | 发现 | 影响 |
|------|------|------|
| **API 能力** | Gitee API v5 功能完整，涵盖 PR、Comment、Merge 等 | ✅ 支持深度集成 |
| **Webhook** | 支持 8+ 种事件类型，包括 PR 全生命周期 | ✅ 可实现实时触发 |
| **企业功能** | 内置 AI 审查、代码扫描、CodeOwners | ⚠️ 与自定义 AI 存在重叠 |
| **当前实现** | 基础功能可用，缺少接口完整性、高级评论功能 | ⚠️ 需要增强 |
| **合规能力** | 支持等保三级、密评、信创适配 | ✅ 适合中国企业 |

### 1.3 建议优先级

1. **P0 (高优先级)**: 补齐 Platform 接口、增强错误处理
2. **P1 (中优先级)**: 实现高级评论功能、Webhook 服务器
3. **P2 (低优先级)**: Gitee Go 集成、企业特性利用

---

## 2. Gitee 企业版能力分析

### 2.1 API v5 核心端点

#### 2.1.1 Pull Request 操作

| 端点 | 方法 | 说明 | 优先级 |
|------|------|------|--------|
| `/v5/repos/{owner}/{repo}/pulls` | GET | 列出 PR | P0 |
| `/v5/repos/{owner}/{repo}/pulls/{number}` | GET | 获取 PR 详情 | P0 |
| `/v5/repos/{owner}/{repo}/pulls/{number}/merge` | PUT | 合并 PR | P1 |
| `/v5/repos/{owner}/{repo}/pulls/{number}/merge` | GET | 检查合并状态 | P1 |
| `/v5/repos/{owner}/{repo}/pulls/{number}/comments` | POST | 创建评论 | P0 |
| `/v5/repos/{owner}/{repo}/pulls/{number}/comments` | GET | 获取评论列表 | P1 |

#### 2.1.2 评论系统

Gitee API v5 支持以下评论类型：

```go
// 评论类型
type CommentType string

const (
    CommentTypePR       CommentType = "PullRequest" // PR 评论
    CommentTypeCommit   CommentType = "Commit"     // 提交评论
    CommentTypeIssue    CommentType = "Issue"      // Issue 评论
)

// 评论位置 (行级评论)
type CommentPosition struct {
    Path      string `json:"path"`       // 文件路径
    Position  int    `json:"position"`   // 行位置
    Side      string `json:"side"`       // LEFT 或 RIGHT
}
```

#### 2.1.3 Webhook 事件

| 事件类型 | Hook 名称 | 触发时机 |
|----------|-----------|----------|
| Push | `push_hooks` | 代码推送 |
| PR 创建 | `merge_request_hooks` | PR 创建/更新 |
| PR 合并 | `merge_request_hooks` | PR 合并 |
| PR 评论 | `note_hooks` | PR 评论 |
| Issue | `issue_hooks` | Issue 变更 |

Webhook 请求头：
```
X-Gitee-Token: {webhook_password}
X-Gitee-Timestamp: {timestamp}
X-Gitee-Event: {event_type}
```

### 2.2 Gitee Go (原生 CI/CD)

Gitee Go 是 Gitee 企业版的内置 CI/CD 平台：

#### 2.2.1 核心概念

```
Pipeline (流水线)
    ├── Stage (阶段)
    │   ├── Task (任务)
    │   └── Task (任务)
    ├── Stage (阶段)
    └── Artifact (产物)
```

#### 2.2.2 触发方式

- **代码推送**: 自动触发
- **PR 创建**: 可配置触发
- **手动触发**: UI 操作
- **定时触发**: Cron 表达式

#### 2.2.3 集成方式

```yaml
# .gitee-pipeline.yml
stages:
  - name: build
    jobs:
      - name: ai-review
        runs-on: linux
        steps:
          - uses: actions/checkout@v1
          - name: AI Code Review
            uses: cicd-ai-toolkit/action@v1
            with:
              platform: gitee
```

### 2.3 企业版特有功能

#### 2.3.1 安全扫描 (GiteeScan)

| 功能 | 说明 | API 集成 |
|------|------|----------|
| 静态代码分析 | SAST 扫描 | `/v5/repos/{owner}/{repo}/code-check` |
| 许可证合规 | 开源协议检测 | 同上 |
| 代码克隆检测 | 重复代码识别 | 同上 |
| PR 门禁 | 扫描失败阻止合并 | Status Check API |

#### 2.3.2 CodeOwners 机制

```yaml
# .gitee/CODEOWNERS
# 每行格式: pattern @user1 @user2

*.go @backend-team @reviewer-bot
/pkg/auth/** @security-team
*.md @docs-team
```

#### 2.3.3 分支保护

- 必需审查人数量
- 状态检查要求
- 推送权限限制
- 合并策略配置

### 2.4 合规能力

| 合规项 | 说明 | 版本要求 |
|--------|------|----------|
| **等保三级** | 信息安全等级保护 | 专业版及以上 |
| **密评** | 密码应用安全性评估 | 专业版及以上 |
| **信创适配** | 国产化环境兼容 | 专业版及以上 |

---

## 3. 当前实现差距分析

### 3.1 Platform 接口合规性

当前 `gitee.go` 缺少以下 Platform 接口要求：

```go
// 缺失方法
func (g *GiteeClient) Name() string {
    return "gitee"  // ❌ 未实现
}
```

### 3.2 功能对比矩阵

| 功能 | GitHub | GitLab | Gitee (当前) | Gitee (目标) |
|------|--------|--------|--------------|--------------|
| **基础操作** |
| PR 信息获取 | ✅ | ✅ | ✅ | ✅ |
| Diff 获取 | ✅ | ✅ | ✅ | ✅ |
| 文件获取 | ✅ | ✅ | ✅ | ✅ |
| 健康检查 | ✅ | ✅ | ✅ | ✅ |
| **评论功能** |
| PR 评论 | ✅ | ✅ | ✅ | ✅ |
| 审查评论 | ✅ | ❌ | ❌ | ✅ |
| 行级评论 | ✅ | ❌ | ❌ | ✅ |
| 审查状态 | ✅ | ✅ | ❌ | ✅ |
| **高级功能** |
| 状态检查 | ✅ | ✅ | ❌ | ✅ |
| PR 合并 | ✅ | ✅ | ❌ | ✅ |
| Label 操作 | ✅ | ❌ | ❌ | P2 |
| Webhook 服务 | ✅ | ❌ | ❌ | P1 |

### 3.3 代码质量问题

#### 3.3.1 安全问题

| 问题 | 位置 | 风险等级 |
|------|------|----------|
| 缺少路径验证 | `GetFile()` | Medium |
| 缺少输入验证 | `PostComment()` | Low |
| Token 日志泄露风险 | 多处 | High |

#### 3.3.2 错误处理

```go
// 当前: 基础错误信息
return fmt.Errorf("failed to post comment (status %d): %s", resp.StatusCode, string(respBody))

// 改进: 结构化错误
type GiteeError struct {
    StatusCode int
    Message    string
    ErrorCode  string
}
```

#### 3.3.3 一致性问题

| 问题 | 描述 |
|------|------|
| HTTP 头不一致 | User-Agent 格式不统一 |
| 认证方式差异 | GitHub 用 `Bearer`，Gitee 用 `token` |
| 错误响应处理 | 未解析 Gitee 错误格式 |

---

## 4. 深度适配方案

### 4.1 架构设计

```
┌─────────────────────────────────────────────────────────────────┐
│                     Gitee 深度适配层                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    GiteeClient (增强版)                     │  │
│  │  ┌───────────┐  ┌───────────┐  ┌───────────┐            │  │
│  │  │   Core    │  │  Comment  │  │   Review  │            │  │
│  │  │   APIs    │  │  Service  │  │  Service  │            │  │
│  │  └───────────┘  └───────────┘  └───────────┘            │  │
│  └──────────────────────────────────────────────────────────┘  │
│                           │                                     │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                   Gitee Webhook Server                    │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐         │  │
│  │  │ PR Events  │  │ Push Events│  │ Validator  │         │  │
│  │  └────────────┘  └────────────┘  └────────────┘         │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 增强功能实现

#### 4.2.1 审查评论服务

```go
// pkg/platform/gitee_review.go

// ReviewComment represents a review comment on specific line
type ReviewComment struct {
    Path      string `json:"path"`
    Position  int    `json:"position"`
    Side      string `json:"side"`       // "LEFT" or "RIGHT"
    Body      string `json:"body"`
    NewCommit bool   `json:"new_commit"` // 强制使用新 commit
}

// ReviewState represents the review state
type ReviewState string

const (
    ReviewStateApproved    ReviewState = "approved"
    ReviewStateChanges     ReviewState = "changes_requested"
    ReviewStateComment     ReviewState = "commented"
    ReviewStatePending     ReviewState = "pending"
)

// PostReviewComment posts a line-level review comment
func (g *GiteeClient) PostReviewComment(ctx context.Context, prID int, comment ReviewComment) error {
    // Gitee API v5: POST /repos/{owner}/{repo}/pulls/{number}/comments
    // with position, path, body, side fields
    payload := map[string]interface{}{
        "body":     comment.Body,
        "path":     comment.Path,
        "position": comment.Position,
        "side":     comment.Side,
    }

    if comment.NewCommit {
        payload["commit_id"] = g.getLatestCommitID(ctx, prID)
    }

    // ... 实现
}

// SetReviewState sets the overall review state
func (g *GiteeClient) SetReviewState(ctx context.Context, prID int, state ReviewState, body string) error {
    // 使用 Gitee 审查 API 或状态检查 API
}
```

#### 4.2.2 状态检查服务

```go
// pkg/platform/gitee_status.go

// StatusState represents the check run state
type StatusState string

const (
    StatusPending   StatusState = "pending"
    StatusRunning   StatusState = "running"
    StatusSuccess   StatusState = "success"
    StatusFailed    StatusState = "failed"
    StatusError     StatusState = "error"
    StatusCancelled StatusState = "cancelled"
)

// CreateStatus creates a status check on a commit
func (g *GiteeClient) CreateStatus(ctx context.Context, sha string, opts StatusOptions) error {
    // Gitee API v5: POST /repos/{owner}/{repo}/statuses/{sha}
    payload := map[string]interface{}{
        "sha":           sha,
        "state":         opts.State,
        "target_url":    opts.TargetURL,
        "description":   opts.Description,
        "context":       opts.Context, // e.g., "cicd-ai-toolkit/review"
    }

    url := fmt.Sprintf("%s/repos/%s/statuses/%s", g.baseURL,
                       url.QueryEscape(g.repo), sha)
    // ... 实现
}
```

#### 4.2.3 Webhook 服务器

```go
// pkg/platform/gitee_webhook.go

// WebhookServer handles Gitee webhook events
type WebhookServer struct {
    server   *http.Server
    secret   string
    handlers map[EventType]EventHandler
}

// EventType represents Gitee webhook event types
type EventType string

const (
    EventPush           EventType = "push_hooks"
    EventMergeRequest   EventType = "merge_request_hooks"
    EventNote           EventType = "note_hooks"
)

// EventHandler processes webhook events
type EventHandler func(ctx context.Context, event *WebhookEvent) error

// WebhookEvent represents a parsed webhook event
type WebhookEvent struct {
    Type      EventType              `json:"hook_name"`
    Timestamp int64                   `json:"timestamp"`
    Repo      *GiteeRepo              `json:"repository"`
    PR        *GiteePR                `json:"pull_request"`
    Sender    *GiteeUser              `json:"sender"`
    Enterprise *GiteeEnterprise       `json:"enterprise"`
}

// Start starts the webhook server
func (s *WebhookServer) Start(addr string) error {
    s.server = &http.Server{
        Addr: addr,
        Handler: http.HandlerFunc(s.handleWebhook),
    }
    return s.server.ListenAndServe()
}

func (s *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
    // 1. 验证签名: X-Gitee-Token
    // 2. 解析事件类型: X-Gitee-Event
    // 3. 调用对应的 handler
}
```

### 4.3 错误处理增强

```go
// pkg/platform/gitee_error.go

// GiteeError represents a Gitee API error
type GiteeError struct {
    StatusCode int    `json:"status_code"`
    ErrorCode  string `json:"error_code"`
    Message    string `json:"message"`
    RequestID  string `json:"request_id"` // 追踪 ID
}

func (e *GiteeError) Error() string {
    if e.ErrorCode != "" {
        return fmt.Sprintf("gitee: %s (code: %s, status: %d)",
                          e.Message, e.ErrorCode, e.StatusCode)
    }
    return fmt.Sprintf("gitee: %s (status: %d)", e.Message, e.StatusCode)
}

// IsRetryable returns true if the error is retryable
func (e *GiteeError) IsRetryable() bool {
    return e.StatusCode == 429 || // Rate limit
           e.StatusCode >= 500 ||  // Server errors
           e.ErrorCode == "temporarily_unavailable"
}
```

### 4.4 配置增强

```yaml
# .cicd-ai-toolkit.yaml
platform:
  gitee:
    # API 配置
    base_url: "https://gitee.com/api/v5"  # 或企业版 URL
    token: "${GITEE_TOKEN}"
    repo: "owner/repo"

    # Webhook 配置
    webhook:
      enabled: true
      secret: "${GITEE_WEBHOOK_SECRET}"
      events:
        - merge_request_hooks
        - push_hooks

    # 审查配置
    review:
      post_as_review: true        # 使用审查评论而非普通评论
      require_approval: false     # 是否要求审批才能合并
      context: "cicd-ai-toolkit"  # Status context

    # 企业版特性
    enterprise:
      code_owners_enabled: true    # 利用 CodeOwners
      branch_protection: true      # 利用分支保护
      status_check: true           # 使用状态检查 API
```

---

## 5. 实施路线图

### 5.1 Phase 1: 基础增强 (1-2 周)

**目标**: 补齐 Platform 接口，提升代码质量

| 任务 | 优先级 | 估时 |
|------|--------|------|
| 实现 `Name()` 方法 | P0 | 1h |
| 统一错误处理 | P0 | 4h |
| 添加路径验证 | P0 | 2h |
| 改进日志输出 | P1 | 2h |
| 单元测试覆盖 | P1 | 8h |

**交付物**:
- 增强的 `gitee.go`
- `gitee_test.go` (覆盖率 >80%)
- `gitee_error.go`

### 5.2 Phase 2: 评论功能 (1-2 周)

**目标**: 实现行级评论和审查状态

| 任务 | 优先级 | 估时 |
|------|--------|------|
| 行级评论 API | P0 | 8h |
| 审查状态 API | P0 | 6h |
| 状态检查 API | P0 | 6h |
| 文档更新 | P1 | 2h |

**交付物**:
- `gitee_review.go`
- `gitee_status.go`
- API 使用文档

### 5.3 Phase 3: Webhook 服务器 (2 周)

**目标**: 实现实时事件处理

| 任务 | 优先级 | 估时 |
|------|--------|------|
| Webhook 服务器框架 | P1 | 12h |
| 事件解析器 | P1 | 8h |
| 签名验证 | P0 | 4h |
| 集成测试 | P1 | 6h |

**交付物**:
- `gitee_webhook.go`
- Webhook 部署文档
- 测试用例

### 5.4 Phase 4: 企业特性利用 (2-3 周)

**目标**: 深度集成 Gitee 企业版功能

| 任务 | 优先级 | 估时 |
|------|--------|------|
| Gitee Go 集成 | P2 | 16h |
| CodeOwners 支持 | P2 | 8h |
| 分支保护集成 | P2 | 6h |
| GiteeScan 结果解析 | P2 | 8h |

**交付物**:
- Gitee Go Action
- 企业特性配置指南

---

## 6. 附录

### 6.1 API 响应示例

#### 6.1.1 PR 详情

```json
{
  "id": 1234567,
  "number": 100,
  "title": "feat: 添加用户认证功能",
  "body": "实现 OAuth2 登录支持",
  "state": "open",
  "head": {
    "label": "feature:oauth",
    "ref": "feature/oauth",
    "sha": "abc123...",
    "repo": {
      "id": 123456,
      "full_name": "company/backend",
      "owner": {
        "login": "company"
      }
    }
  },
  "base": {
    "label": "main",
    "ref": "main",
    "sha": "def456...",
    "repo": {
      "id": 123456,
      "full_name": "company/backend"
    }
  },
  "user": {
    "login": "developer",
    "name": "开发者"
  },
  "created_at": "2026-01-28T10:00:00+08:00",
  "updated_at": "2026-01-28T12:00:00+08:00",
  "mergeable": true,
  "merged": false,
  "merged_at": null
}
```

#### 6.1.2 Webhook Payload

```json
{
  "hook_name": "merge_request_hooks",
  "hook_id": 1001,
  "timestamp": "1680000000000",
  "sign": "signature_hash",
  "repository": {
    "id": 123456,
    "full_name": "company/backend",
    "private": true,
    "owner": {
      "id": 789,
      "login": "company",
      "name": "公司"
    }
  },
  "pull_request": {
    "id": 1234567,
    "number": 100,
    "title": "feat: 添加用户认证功能",
    "body": "实现 OAuth2 登录支持",
    "state": "open",
    "head": "feature:oauth",
    "base": "main",
    "author": {
      "id": 1001,
      "login": "developer",
      "full_name": "开发者"
    }
  },
  "sender": {
    "id": 1001,
    "login": "developer",
    "full_name": "开发者"
  },
  "enterprise": {
    "id": 456,
    "name": "公司",
    "slug": "company"
  },
  "action": "open"
}
```

### 6.2 参考资料

| 资源 | URL |
|------|-----|
| Gitee API v5 文档 | https://gitee.com/api/v5/swagger |
| Gitee OAuth 文档 | https://gitee.com/api/v5/oauth_doc |
| Webhook 文档 | https://help.gitee.com/webhook/gitee-webhook-intro |
| Gitee Go 文档 | https://help.gitee.com/enterprise/pipeline/introduce |
| GiteeScan 文档 | https://help.gitee.com/enterprise/codescan/scan-start |
| CodeOwners 文档 | https://help.gitee.com/enterprise/code-manage/代码审查/Code%20Owner%20机制/如何使用CodeOwners功能 |

### 6.3 SDK 参考

| 语言 | 项目地址 |
|------|----------|
| Go | https://gitee.com/openeuler/go-gitee |
| Java | https://gitee.com/sdk/gitee5j |
| PHP | https://github.com/gitee-php/gitee-sdk |
| Python | https://gitee.com/catlikepuma/gitee-python-client |

---

**文档版本**: 1.0.0
**最后更新**: 2026-01-28
**作者**: CICD AI Toolkit 团队
