# SPEC-PLAT-01: Platform Adapter Interface

**Version**: 1.1
**Status**: Draft
**Date**: 2026-01-24
**Changelog**:
- v1.1: 扩展接口定义，支持 Gitee Enterprise、完整 Check Run 接口

## 1. 概述 (Overview)

为了支持 GitHub, GitLab, Gitee, Jenkins 等多个平台，Runner 必须通过统一的接口层与外部交互。Go 的 `interface` 提供了极佳的抽象能力。

## 2. 核心职责 (Core Responsibilities)

- **Abstraction**: 屏蔽 API 差异 (GitHub v3 REST vs v4 GraphQL vs Gitee v5 vs GitLab v4)
- **Auth Handling**: 自动处理 Token 刷新和签名
- **Unified Domain Model**: 将不同平台的 `PullRequest` 结构转换为内部结构
- **Feature Fallback**: 当平台不支持某特性时自动降级

## 3. 详细设计 (Detailed Design)

### 3.1 平台支持矩阵

| 平台 | Tier | 状态 | Spec | 关键特性 |
|------|------|------|------|----------|
| **GitHub** | 1 | ✅ 完整 | PLAT-05 | Check Runs, Actions, Composite |
| **Gitee Enterprise** | 1 | ✅ 完整 | PLAT-06 | 企业版 API, 私有市场, OAuth2 |
| **GitLab** | 2 | ✅ 完整 | PLAT-03 | Merge Request, Pipeline API |
| **Jenkins** | Legacy | ✅ 完整 | PLAT-04 | Plugin, Build Step |

### 3.2 统一接口定义 (Go Interface)

```go
// Platform defines the operations required by the Runner
type Platform interface {
    // Identity
    Name() string
    Type() PlatformType

    // Environment Detection
    Detect(env map[string]string) bool
    GetEnvInfo(ctx context.Context) (*CIEnvInfo, error)

    // Pull Request / Merge Request Operations
    GetPullRequest(ctx context.Context, number int) (*PullRequest, error)
    GetDiff(ctx context.Context, number int) (string, error)
    GetCommits(ctx context.Context, number int) ([]*Commit, error)

    // Comment Operations
    PostComment(ctx context.Context, number int, body string) error
    UpdateComment(ctx context.Context, commentID string, body string) error
    DeleteComment(ctx context.Context, commentID string) error

    // Review Operations (Inline Comments)
    PostReview(ctx context.Context, number int, review *Review) error
    ResolveComment(ctx context.Context, commentID string) error

    // Status / Check Run Operations
    CreateStatus(ctx context.Context, sha string, status *CheckStatus) error
    UpdateCheckRun(ctx context.Context, runID string, status *CheckStatus) error

    // Label Operations
    AddLabels(ctx context.Context, number int, labels []string) error
    RemoveLabels(ctx context.Context, number int, labels []string) error
}

// PlatformType 平台类型
type PlatformType string

const (
    PlatformGitHub  PlatformType = "github"
    PlatformGitee   PlatformType = "gitee"
    PlatformGitLab  PlatformType = "gitlab"
    PlatformJenkins PlatformType = "jenkins"
)

// CIEnvInfo CI 环境信息
type CIEnvInfo struct {
    Platform    PlatformType `json:"platform"`
    Action      string       `json:"action"`
    RunID       string       `json:"run_id"`
    RunNumber   string       `json:"run_number"`
    JobID       string       `json:"job_id"`
    Repository  string       `json:"repository"`
    Ref         string       `json:"ref"`
    SHA         string       `json:"sha"`
    Actor       string       `json:"actor"`
    Workspace   string       `json:"workspace"`
    EventName   string       `json:"event_name"`
    EventPath   string       `json:"event_path"`
}

// PullRequest 统一的 PR/MR 结构
type PullRequest struct {
    Number       int       `json:"number"`
    Title        string    `json:"title"`
    Body         string    `json:"body"`
    State        string    `json:"state"`      // open, closed, merged
    HeadBranch   string    `json:"head_branch"`
    BaseBranch   string    `json:"base_branch"`
    HeadSHA      string    `json:"head_sha"`
    BaseSHA      string    `json:"base_sha"`
    Author       string    `json:"author"`
    AuthorEmail  string    `json:"author_email"`
    HTMLURL      string    `json:"html_url"`
    DiffURL      string    `json:"diff_url"`
    PatchURL     string    `json:"patch_url"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
    Mergeable    bool      `json:"mergeable"`
    Draft        bool      `json:"draft"`
    Commits      int       `json:"commits"`
    Additions    int       `json:"additions"`
    Deletions    int       `json:"deletions"`
    ChangedFiles int       `json:"changed_files"`
}

// Commit 提交信息
type Commit struct {
    SHA       string    `json:"sha"`
    Message   string    `json:"message"`
    Author    string    `json:"author"`
    AuthorEmail string `json:"author_email"`
    Timestamp time.Time `json:"timestamp"`
    URL       string    `json:"url"`
}

// Review 代码审查
type Review struct {
    Body         string          `json:"body"`
    Comments     []InlineComment `json:"comments"`
    Event        ReviewEvent     `json:"event"` // APPROVE, REQUEST_CHANGES, COMMENT
}

// ReviewEvent 审查事件
type ReviewEvent string

const (
    ReviewApprove        ReviewEvent = "APPROVE"
    ReviewRequestChanges ReviewEvent = "REQUEST_CHANGES"
    ReviewComment        ReviewEvent = "COMMENT"
)

// InlineComment 行内评论
type InlineComment struct {
    Path       string `json:"path"`
    Line       int    `json:"line"`
    StartLine  int    `json:"start_line,omitempty"` // For multi-line comments
    Body       string `json:"body"`
    Side       string `json:"side,omitempty"` // LEFT, RIGHT (for diff comments)
}

// CheckStatus 状态检查
type CheckStatus struct {
    Context     string `json:"context"`              // e.g., "cicd-ai-review"
    State       string `json:"state"`                // pending, success, error, failure
    Description string `json:"description"`
    TargetURL   string `json:"target_url,omitempty"`
}
```

### 3.3 适配器实现 (Implementations)

#### GitHub Adapter

```go
// GitHubPlatform GitHub 平台适配器
type GitHubPlatform struct {
    client *github.Client
    config *GitHubConfig
}

type GitHubConfig struct {
    Token        string `env:"GITHUB_TOKEN"`
    APIURL       string `env:"GITHUB_API_URL"`
    UploadURL    string `env:"GITHUB_UPLOAD_URL"`
}

func NewGitHubPlatform(cfg *GitHubConfig) (*GitHubPlatform, error) {
    ts := oauth2.NewTokenSource(
        oauth2.NoContext,
        &oauth2.Token{AccessToken: cfg.Token},
    )

    client := github.NewClient(ts).WithEnterpriseURLs(
        cfg.APIURL,
        cfg.UploadURL,
    )

    return &GitHubPlatform{client: client, config: cfg}, nil
}

func (p *GitHubPlatform) Name() string { return "github" }
func (p *GitHubPlatform) Type() PlatformType { return PlatformGitHub }

func (p *GitHubPlatform) Detect(env map[string]string) bool {
    return env["GITHUB_ACTIONS"] == "true"
}
```

#### Gitee Adapter

```go
// GiteePlatform Gitee 平台适配器
type GiteePlatform struct {
    client      *GiteeClient
    config      *GiteeConfig
    enterpriseID string // 企业版 ID
}

type GiteeConfig struct {
    Token       string `env:"GITEE_TOKEN"`
    BaseURL     string `env:"GITEE_API_URL"` // 默认: https://api.gitee.com/v5
    EnterpriseID string `yaml:"enterprise_id"` // 企业版 ID
}

func NewGiteePlatform(cfg *GiteeConfig) (*GiteePlatform, error) {
    client, err := NewGiteeClient(cfg)
    if err != nil {
        return nil, err
    }

    return &GiteePlatform{
        client:      client,
        config:      cfg,
        enterpriseID: cfg.EnterpriseID,
    }, nil
}

func (p *GiteePlatform) Name() string { return "gitee" }
func (p *GiteePlatform) Type() PlatformType { return PlatformGitee }

func (p *GiteePlatform) Detect(env map[string]string) bool {
    indicators := []string{
        "GITEE_ACTIONS",
        "GITEA_ACTIONS",
        "GITEE_SERVER_URL",
        "GITEE_TOKEN",
    }

    for _, key := range indicators {
        if _, ok := env[key]; ok {
            return true
        }
    }

    if remote, ok := env["GIT_REMOTE_URL"]; ok {
        return strings.Contains(remote, "gitee.com") ||
               strings.Contains(remote, "gitea.")
    }

    return false
}

// Gitee 特殊处理: Note Event 转换
func (p *GiteePlatform) PostReview(ctx context.Context, number int, review *Review) error {
    // Gitee 使用 Note Event 表示评论
    // 不支持 GitHub 的 Pull Request Review API
    // 降级为普通评论
    return p.PostComment(ctx, number, review.Body)
}
```

#### GitLab Adapter

```go
// GitLabPlatform GitLab 平台适配器
type GitLabPlatform struct {
    client *gitlab.Client
    config *GitLabConfig
}

type GitLabConfig struct {
    Token   string `env:"GITLAB_TOKEN"`
    BaseURL string `env:"GITLAB_API_URL"`
}

func NewGitLabPlatform(cfg *GitLabConfig) (*GitLabPlatform, error) {
    client, err := gitlab.NewClient(cfg.Token, gitlab.WithBaseURL(cfg.BaseURL))
    if err != nil {
        return nil, err
    }

    return &GitLabPlatform{client: client, config: cfg}, nil
}

func (p *GitLabPlatform) Name() string { return "gitlab" }
func (p *GitLabPlatform) Type() PlatformType { return PlatformGitLab }

func (p *GitLabPlatform) Detect(env map[string]string) bool {
    return env["GITLAB_CI"] == "true"
}

// GitLab 使用 Merge Request 而非 Pull Request
func (p *GitLabPlatform) GetPullRequest(ctx context.Context, number int) (*PullRequest, error) {
    // GitLab 使用 IID (内部 ID) 而非全局编号
    mr, _, err := p.client.MergeRequests.GetProjectMergeRequest(
        p.getProjectID(),
        number,
        nil,
    )
    if err != nil {
        return nil, err
    }

    return &PullRequest{
        Number:     mr.IID,
        Title:      mr.Title,
        Body:       mr.Description,
        State:      mr.State,
        HeadBranch: mr.SourceBranch,
        BaseBranch: mr.TargetBranch,
        HeadSHA:    mr.SHA,
        Author:     mr.Author.Username,
        HTMLURL:    mr.WebURL,
        CreatedAt:  *mr.CreatedAt,
        UpdatedAt:  *mr.UpdatedAt,
    }, nil
}
```

### 3.4 工厂模式 (Factory)

```go
// PlatformRegistry 平台注册表
type PlatformRegistry struct {
    platforms map[PlatformType]PlatformFactory
}

type PlatformFactory func(cfg interface{}) (Platform, error)

// DefaultRegistry 默认平台注册表
var DefaultRegistry = &PlatformRegistry{
    platforms: map[PlatformType]PlatformFactory{
        PlatformGitHub:  func(cfg interface{}) (Platform, error) { return NewGitHubPlatform(cfg.(*GitHubConfig)) },
        PlatformGitee:   func(cfg interface{}) (Platform, error) { return NewGiteePlatform(cfg.(*GiteeConfig)) },
        PlatformGitLab:  func(cfg interface{}) (Platform, error) { return NewGitLabPlatform(cfg.(*GitLabConfig)) },
        PlatformJenkins: func(cfg interface{}) (Platform, error) { return NewJenkinsPlatform(cfg.(*JenkinsConfig)) },
    },
}

// NewPlatform 从环境变量自动检测并创建平台
func NewPlatform() (Platform, error) {
    env := getEnvMap()

    // 优先级: GitHub > Gitee > GitLab > Jenkins
    for _, factory := range []struct {
        typ      PlatformType
        detector func(map[string]string) bool
        factory  PlatformFactory
    }{
        {PlatformGitHub, detectGitHub, DefaultRegistry.platforms[PlatformGitHub]},
        {PlatformGitee, detectGitee, DefaultRegistry.platforms[PlatformGitee]},
        {PlatformGitLab, detectGitLab, DefaultRegistry.platforms[PlatformGitLab]},
        {PlatformJenkins, detectJenkins, DefaultRegistry.platforms[PlatformJenkins]},
    } {
        if factory.detector != nil && factory.detector(env) {
            return factory.factory(nil)
        }
    }

    return nil, fmt.Errorf("no supported platform detected")
}

func detectGitHub(env map[string]string) bool {
    return env["GITHUB_ACTIONS"] == "true"
}

func detectGitee(env map[string]string) bool {
    return env["GITEE_ACTIONS"] == "true" || env["GITEE_TOKEN"] != ""
}

func detectGitLab(env map[string]string) bool {
    return env["GITLAB_CI"] == "true"
}

func detectJenkins(env map[string]string) bool {
    return env["JENKINS_HOME"] != "" || env["JENKINS_URL"] != ""
}
```

### 3.5 特性降级 (Feature Fallback)

```go
// Capability 平台能力
type Capability int

const (
    CapabilityCheckRun Capability = 1 << iota  // 详细 Check Run API
    CapabilityInlineReview                      // 行内评论
    CapabilityMergeRequest                      // Merge Request (vs PR)
    CapabilityPipelineStatus                    // Pipeline Status API
    CapabilityPrivateMarketplace               // 私有插件市场
    CapabilityEnterpriseAPI                     // 企业版 API
)

// Capabilities 获取平台能力
func (p *GitHubPlatform) Capabilities() Capability {
    return CapabilityCheckRun | CapabilityInlineReview
}

func (p *GiteePlatform) Capabilities() Capability {
    caps := CapabilityInlineReview | CapabilityPrivateMarketplace
    if p.enterpriseID != "" {
        caps |= CapabilityEnterpriseAPI
    }
    return caps
}

func (p *GitLabPlatform) Capabilities() Capability {
    return CapabilityMergeRequest | CapabilityPipelineStatus
}

// HasCapability 检查是否支持某特性
func HasCapability(p Platform, cap Capability) bool {
    return p.Capabilities()&cap != 0
}

// FallbackHandler 特性降级处理器
type FallbackHandler struct{}

// CreateStatus 状态创建 (带降级)
func (fh *FallbackHandler) CreateStatus(ctx context.Context, p Platform, sha string, status *CheckStatus) error {
    if HasCapability(p, CapabilityCheckRun) {
        // 使用 Check Run API
        return p.CreateCheckRun(ctx, sha, status)
    }

    // 降级为 Commit Status API
    return p.CreateStatus(ctx, sha, status)
}
```

### 3.6 平台配置

```yaml
# .cicd-ai-toolkit.yaml
platform:
  # 自动检测平台 (默认)
  auto_detect: true

  # 手动指定平台 (优先级高于自动检测)
  type: ""  # github | gitee | gitlab | jenkins

  # GitHub 配置
  github:
    token_env: "GITHUB_TOKEN"
    api_url: ""  # 企业版 GitHub API
    post_comment: true
    fail_on_error: false
    max_comment_length: 65536

  # Gitee 配置
  gitee:
    token_env: "GITEE_TOKEN"
    api_url: "https://api.gitee.com/v5"
    enterprise_id: ""  # 企业版 ID (可选)
    post_comment: true
    fail_on_error: false

  # GitLab 配置
  gitlab:
    token_env: "GITLAB_TOKEN"
    api_url: "https://gitlab.com/api/v4"
    post_comment: true
    fail_on_error: false

  # Jenkins 配置
  jenkins:
    url_env: "JENKINS_URL"
    username_env: "JENKINS_USERNAME"
    token_env: "JENKINS_TOKEN"
```

## 4. 平台差异对照表

| 特性 | GitHub | Gitee | GitLab | Jenkins |
|------|--------|-------|--------|---------|
| **PR/MR 术语** | Pull Request | Pull Request | Merge Request | Pull Request |
| **状态 API** | Check Runs API | Status API | Pipeline API | Build API |
| **行内评论** | ✅ Pull Request Review | ⚠️ Comment only | ✅ Discussion | ✅ |
| **Webhook** | ✅ | ✅ | ✅ | ✅ |
| **OAuth** | ✅ | ✅ | ✅ | - |
| **企业版** | ✅ GitHub Server | ✅ Enterprise | ✅ Self-Hosted | - |
| **私有市场** | - | ✅ | - | - |

## 5. 依赖关系 (Dependencies)

- **Used by**: [SPEC-CORE-01](./SPEC-CORE-01-Runner_Lifecycle.md) 在 Init 阶段调用 `NewPlatform`。
- **Related**: [SPEC-PLAT-06](./SPEC-PLAT-06-Gitee_Adapter.md) - Gitee 详细实现
- **Related**: [SPEC-PLAT-03](./SPEC-PLAT-03-GitLab_Adapter.md) - GitLab 详细实现
- **Related**: [SPEC-PLAT-04](./SPEC-PLAT-04-Jenkins_Plugin.md) - Jenkins 详细实现

## 6. 验收标准 (Acceptance Criteria)

1. **Mock Test**: 使用 `MockPlatform` 验证 Runner 逻辑，确保 `PostComment` 被正确调用。
2. **GitHub Live Test**: 配置真实 GHA Token，验证能在 PR 下方发表评论。
3. **Gitee Live Test**: 模拟 Gitee 环境，验证能调用 Gitee V5 API 获取 Diff。
4. **GitLab Live Test**: 验证能正确处理 Merge Request 而非 Pull Request。
5. **Auto Detection**: 在不同 CI 环境中，`NewPlatform()` 自动返回正确的平台。
6. **Feature Fallback**: 在不支持 Check Run 的平台上自动降级为 Status API。
7. **Unified Model**: 所有平台的 PR 都能正确转换为 `PullRequest` 结构。
