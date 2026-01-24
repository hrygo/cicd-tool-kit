# SPEC-PLAT-06: Gitee Enterprise Adapter

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24
**Covers**: PRD 2.1 (Gitee Enterprise Tier 1 - P0), 3.3 (Gitee API 适配), 3.8 (Gitee Go 集成)

## 1. 概述 (Overview)

Gitee Enterprise 是国内主流的企业级代码托管平台，与 GitHub Actions 高度兼容（基于 Gitea/act_runner）。本 Spec 定义如何将 `cicd-ai-toolkit` 适配到 Gitee Enterprise 环境，包括 API 集成、Webhook 处理和私有插件市场适配。

## 2. 核心职责 (Core Responsibilities)

- **API 适配**: 实现 Gitee API v5 客户端，支持企业版特有的 `enterprises/{id}` 鉴权
- **Webhook 标准化**: 将 Gitee Webhook 事件转换为统一的内部事件格式
- **Action 兼容**: 确保 action.yml 与 Gitee Go (Gitea Actions) 兼容
- **私有市场适配**: 支持 Gitee 企业版私有插件市场规范

## 3. 详细设计 (Detailed Design)

### 3.1 Gitee 平台特性

| 特性 | 说明 | 兼容性 |
|------|------|--------|
| **Gitee Go** | 基于 act_runner，与 GitHub Actions 高度兼容 | ✅ 直接复用 action.yml |
| **API v5** | RESTful API，类似 GitHub API v3 | ⚠️ 需适配差异 |
| **OAuth2** | 支持标准 OAuth2 授权 | ✅ 标准流程 |
| **企业版** | `enterprises/{id}` 命名空间 | ⚠️ 特殊处理 |
| **私有市场** | 企业版私有插件市场 | ⚠️ 需适配规范 |

### 3.2 环境变量检测 (Environment Detection)

```go
// Platform Detection
func DetectGitee(env map[string]string) bool {
    indicators := []string{
        "GITEE_ACTIONS",          // Gitee Go 原生标识
        "GITEA_ACTIONS",          // Gitea 兼容标识
        "GITEE_SERVER_URL",       // Gitee 自建实例
        "GITEE_TOKEN",            // Gitee Token
    }

    for _, key := range indicators {
        if _, ok := env[key]; ok {
            return true
        }
    }

    // 检测 Git remote
    if remote, ok := env["GIT_REMOTE_URL"]; ok {
        return strings.Contains(remote, "gitee.com") ||
               strings.Contains(remote, "gitea.")
    }

    return false
}
```

### 3.3 Gitee API v5 客户端

```go
// GiteeClient Gitee API v5 客户端
type GiteeClient struct {
    httpClient *http.Client
    baseURL    string
    token      string
    enterpriseID string // 企业版 ID
}

// GiteeConfig Gitee 配置
type GiteeConfig struct {
    // API 端点 (默认: https://api.gitee.com/v5)
    BaseURL string `yaml:"base_url" default:"https://api.gitee.com/v5"`

    // 访问令牌 (从环境变量 GITEE_TOKEN 读取)
    Token string `yaml:"token_env" default:"GITEE_TOKEN"`

    // 企业版 ID (可选，用于企业版 API)
    EnterpriseID string `yaml:"enterprise_id"`

    // 超时配置
    Timeout time.Duration `yaml:"timeout" default:"30s"`
}

// NewGiteeClient 创建 Gitee 客户端
func NewGiteeClient(cfg *GiteeConfig) (*GiteeClient, error) {
    token := os.Getenv(cfg.Token)
    if token == "" {
        return nil, fmt.Errorf("GITEE_TOKEN environment variable not set")
    }

    return &GiteeClient{
        httpClient: &http.Client{
            Timeout: cfg.Timeout,
        },
        baseURL:     cfg.BaseURL,
        token:       token,
        enterpriseID: cfg.EnterpriseID,
    }, nil
}

// doRequest 执行 API 请求
func (c *GiteeClient) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
    url := fmt.Sprintf("%s%s", c.baseURL, path)

    req, err := http.NewRequestWithContext(ctx, method, url, body)
    if err != nil {
        return nil, err
    }

    // Gitee 使用 Bearer Token 或 access_token 参数
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
    req.Header.Set("Content-Type", "application/json;charset=UTF-8")

    // 添加 User-Agent
    req.Header.Set("User-Agent", "cicd-ai-toolkit/1.0")

    return c.httpClient.Do(req)
}
```

### 3.4 Pull Request 操作

```go
// GiteePR Gitee Pull Request 信息
type GiteePR struct {
    ID          int    `json:"id"`
    Number      int    `json:"number"`
    Title       string `json:"title"`
    Body        string `json:"body"`
    State       string `json:"state"`
    Head        string `json:"head"`
    Base        string `json:"base"`
    Author      struct {
        ID       int    `json:"id"`
        Login    string `json:"login"`
        Name     string `json:"name"`
        Email    string `json:"email"`
    } `json:"user"`
    URL         string `json:"url"`
    HTMLURL     string `json:"html_url"`
    DiffURL     string `json:"diff_url"`
    PatchURL    string `json:"patch_url"`
    CreatedAt   string `json:"created_at"`
    UpdatedAt   string `json:"updated_at"`
    MergedAt    *string `json:"merged_at"`
}

// GetPR 获取 PR 信息
func (c *GiteeClient) GetPR(ctx context.Context, owner, repo string, number int) (*GiteePR, error) {
    path := fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number)

    // 企业版路径
    if c.enterpriseID != "" {
        path = fmt.Sprintf("/enterprises/%s/repos/%s/%s/pulls/%d",
            c.enterpriseID, owner, repo, number)
    }

    resp, err := c.doRequest(ctx, "GET", path, nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("Gitee API error: %s", resp.Status)
    }

    var pr GiteePR
    if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
        return nil, err
    }

    return &pr, nil
}

// GetDiff 获取 PR Diff
func (c *GiteeClient) GetDiff(ctx context.Context, owner, repo string, number int) (string, error) {
    pr, err := c.GetPR(ctx, owner, repo, number)
    if err != nil {
        return "", err
    }

    resp, err := c.doRequest(ctx, "GET", pr.DiffURL, nil)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    diff, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    return string(diff), nil
}
```

### 3.5 评论操作 (Comments)

```go
// GiteeComment Gitee 评论
type GiteeComment struct {
    ID        int    `json:"id"`
    Body      string `json:"body"`
    User      struct {
        Login string `json:"login"`
        Name  string `json:"name"`
    } `json:"user"`
    CreatedAt string `json:"created_at"`
    HTMLURL   string `json:"html_url"`
}

// CreateComment 创建 PR 评论
func (c *GiteeClient) CreateComment(ctx context.Context, owner, repo string, number int, body string) error {
    path := fmt.Sprintf("/repos/%s/%s/pulls/%d/comments", owner, repo, number)

    if c.enterpriseID != "" {
        path = fmt.Sprintf("/enterprises/%s/repos/%s/%s/pulls/%d/comments",
            c.enterpriseID, owner, repo, number)
    }

    payload := map[string]string{
        "body": body,
    }

    bodyBytes, _ := json.Marshal(payload)

    resp, err := c.doRequest(ctx, "POST", path, bytes.NewReader(bodyBytes))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        return fmt.Errorf("failed to create comment: %s", resp.Status)
    }

    return nil
}

// FindComment 查找是否存在相同内容的评论
func (c *GiteeClient) FindComment(ctx context.Context, owner, repo string, number int, body string) (*GiteeComment, error) {
    path := fmt.Sprintf("/repos/%s/%s/pulls/%d/comments", owner, repo, number)

    if c.enterpriseID != "" {
        path = fmt.Sprintf("/enterprises/%s/repos/%s/%s/pulls/%d/comments",
            c.enterpriseID, owner, repo, number)
    }

    resp, err := c.doRequest(ctx, "GET", path, nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var comments []GiteeComment
    if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
        return nil, err
    }

    // 计算目标内容的 hash
    targetHash := sha256.Sum256([]byte(body))

    for _, comment := range comments {
        commentHash := sha256.Sum256([]byte(comment.Body))
        if commentHash == targetHash {
            return &comment, nil
        }
    }

    return nil, fmt.Errorf("comment not found")
}
```

### 3.6 Check Status 操作

```go
// GiteeCheckStatus Gitee Check 状态
type GiteeCheckStatus struct {
    Context     string `json:"context"`
    Description string `json:"description"`
    State       string `json:"state"` // pending, success, error, failure
    TargetURL   string `json:"target_url,omitempty"`
}

// CreateStatus 创建 commit status
func (c *GiteeClient) CreateStatus(ctx context.Context, owner, repo, sha string, status *GiteeCheckStatus) error {
    path := fmt.Sprintf("/repos/%s/%s/statuses/%s", owner, repo, sha)

    if c.enterpriseID != "" {
        path = fmt.Sprintf("/enterprises/%s/repos/%s/%s/statuses/%s",
            c.enterpriseID, owner, repo, sha)
    }

    payload := map[string]interface{}{
        "context":     status.Context,
        "description": status.Description,
        "state":       status.State,
    }

    if status.TargetURL != "" {
        payload["target_url"] = status.TargetURL
    }

    bodyBytes, _ := json.Marshal(payload)

    resp, err := c.doRequest(ctx, "POST", path, bytes.NewReader(bodyBytes))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        return fmt.Errorf("failed to create status: %s", resp.Status)
    }

    return nil
}
```

### 3.7 Webhook 事件处理

```go
// GiteeWebhookEvent Gitee Webhook 事件
type GiteeWebhookEvent struct {
    // 事件类型
    Action     string `json:"action"` // opened, updated, closed, etc.
    // PR 信息
    PullRequest GiteePR `json:"pull_request"`
    // 仓库信息
    Repository struct {
        ID       int    `json:"id"`
        Name     string `json:"name"`
        FullName string `json:"full_name"`
        Owner    struct {
            Login string `json:"login"`
            Name  string `json:"name"`
        } `json:"owner"`
        Private bool   `json:"private"`
        HTMLURL string `json:"html_url"`
    } `json:"repository"`
    // 发送者
    Sender struct {
        Login string `json:"login"`
        Name  string `json:"name"`
    } `json:"sender"`
}

// ToInternalEvent 转换为内部事件格式
func (e *GiteeWebhookEvent) ToInternalEvent() *InternalReviewEvent {
    return &InternalReviewEvent{
        Platform:      "gitee",
        EventType:     "pull_request",
        Action:        e.Action,
        PRNumber:      e.PullRequest.Number,
        PRTitle:       e.PullRequest.Title,
        PRBody:        e.PullRequest.Body,
        HeadSHA:       "", // 需要额外获取
        HeadBranch:    e.PullRequest.Head,
        BaseBranch:    e.PullRequest.Base,
        Author:        e.PullRequest.Author.Login,
        Repository:    e.Repository.FullName,
        IsPrivate:     e.Repository.Private,
        HTMLURL:       e.PullRequest.HTMLURL,
    }
}
```

### 3.8 Platform Adapter 接口实现

```go
// GiteePlatform 实现 Platform 接口
type GiteePlatform struct {
    client     *GiteeClient
    config     *GiteeConfig
}

// NewGiteePlatform 创建 Gitee 平台适配器
func NewGiteePlatform(cfg *GiteeConfig) (*GiteePlatform, error) {
    client, err := NewGiteeClient(cfg)
    if err != nil {
        return nil, err
    }

    return &GiteePlatform{
        client: client,
        config: cfg,
    }, nil
}

// Name 返回平台名称
func (p *GiteePlatform) Name() string {
    return "gitee"
}

// GetEnvInfo 获取 CI 环境信息
func (p *GiteePlatform) GetEnvInfo(ctx context.Context) (*CIEnvInfo, error) {
    return &CIEnvInfo{
        Platform:       "gitee",
        Action:         os.Getenv("GITEE_ACTION_NAME"),
        RunID:          os.Getenv("GITEE_RUN_ID"),
        RunNumber:      os.Getenv("GITEE_RUN_NUMBER"),
        JobID:          os.Getenv("GITEE_JOB_ID"),
        Repository:     os.Getenv("GITEE_REPOSITORY"),
        Ref:            os.Getenv("GITEE_REF_NAME"),
        SHA:            os.Getenv("GITEE_SHA"),
        Actor:          os.Getenv("GITEE_ACTOR"),
        Workspace:      os.Getenv("GITEE_WORKSPACE"),
        EventName:      os.Getenv("GITEE_EVENT_NAME"),
        EventPath:      os.Getenv("GITEE_EVENT_PATH"),
    }, nil
}

// GetPullRequest 获取当前 PR 信息
func (p *GiteePlatform) GetPullRequest(ctx context.Context, number int) (*PullRequest, error) {
    repo := os.Getenv("GITEE_REPOSITORY")
    parts := strings.Split(repo, "/")
    if len(parts) != 2 {
        return nil, fmt.Errorf("invalid GITEE_REPOSITORY format")
    }
    owner, name := parts[0], parts[1]

    giteePR, err := p.client.GetPR(ctx, owner, name, number)
    if err != nil {
        return nil, err
    }

    return &PullRequest{
        Number:      giteePR.Number,
        Title:       giteePR.Title,
        Body:        giteePR.Body,
        State:       giteePR.State,
        HeadBranch:  giteePR.Head,
        BaseBranch:  giteePR.Base,
        Author:      giteePR.Author.Login,
        HTMLURL:     giteePR.HTMLURL,
        CreatedAt:   giteePR.CreatedAt,
        UpdatedAt:   giteePR.UpdatedAt,
    }, nil
}

// GetDiff 获取 PR Diff
func (p *GiteePlatform) GetDiff(ctx context.Context, number int) (string, error) {
    repo := os.Getenv("GITEE_REPOSITORY")
    parts := strings.Split(repo, "/")
    owner, name := parts[0], parts[1]

    return p.client.GetDiff(ctx, owner, name, number)
}

// PostComment 发表 PR 评论
func (p *GiteePlatform) PostComment(ctx context.Context, number int, body string) error {
    repo := os.Getenv("GITEE_REPOSITORY")
    parts := strings.Split(repo, "/")
    owner, name := parts[0], parts[1]

    return p.client.CreateComment(ctx, owner, name, number, body)
}

// UpdateStatus 更新 commit status
func (p *GiteePlatform) UpdateStatus(ctx context.Context, sha string, status *CheckStatus) error {
    repo := os.Getenv("GITEE_REPOSITORY")
    parts := strings.Split(repo, "/")
    owner, name := parts[0], parts[1]

    giteeStatus := &GiteeCheckStatus{
        Context:     status.Context,
        Description: status.Description,
        State:       status.State,
        TargetURL:   status.TargetURL,
    }

    return p.client.CreateStatus(ctx, owner, name, sha, giteeStatus)
}
```

### 3.9 Gitee Go Workflow 集成

```yaml
# .gitee/workflows/ai-review.yml
name: AI Code Review
on:
  pull_request:
    types: [opened, synchronize, reopened]

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
          # Gitee 环境变量自动注入
          gitee_token: ${{ secrets.GITEE_TOKEN }}

  ai-security:
    runs-on: ubuntu-latest
    if: github.event.action == 'opened'
    steps:
      - uses: actions/checkout@v4

      - name: Security Scan
        uses: cicd-ai-toolkit/action@v1
        with:
          run_skills: "security-scanner"
          severity_threshold: "high"
```

### 3.10 私有插件市场适配

Gitee 企业版支持私有插件市场，规范如下：

```yaml
# Gitee 私有市场规范
# marketplace.yaml (插件元数据)
apiVersion: "gitee.com/v1"
kind: "Plugin"

metadata:
  name: "cicd-ai-toolkit"
  displayName: "AI 代码审查工具"
  version: "1.0.0"
  description: "基于 Claude Code 的智能代码审查"

# 企业版市场要求
enterprise:
  # 插件分类
  category: "code-quality"

  # 兼容的 Gitee 版本
  gitee_versions:
    - "enterprise-8.0+"
    - "community-6.0+"

  # 权限声明
  permissions:
    - "pull_request:read"
    - "pull_request:comment"
    - "commit_status:write"

  # 安全合规
  security:
    sandboxed: true
    network_policy: "restricted"
```

```go
// GiteeMarketplaceClient Gitee 私有市场客户端
type GiteeMarketplaceClient struct {
    client *GiteeClient
}

// PublishPlugin 发布插件到私有市场
func (mc *GiteeMarketplaceClient) PublishPlugin(ctx context.Context, plugin *PluginManifest) error {
    // 上传插件元数据
    path := fmt.Sprintf("/enterprises/%s/marketplace/plugins", mc.client.enterpriseID)

    payload, _ := json.Marshal(plugin)
    resp, err := mc.client.doRequest(ctx, "POST", path, bytes.NewReader(payload))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        return fmt.Errorf("failed to publish plugin: %s", resp.Status)
    }

    return nil
}
```

### 3.11 One-Click 安装脚本

```bash
#!/bin/bash
# install-gitee.sh - Gitee Runner 一键安装脚本

set -e

GITEE_VERSION="${GITEE_VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

echo "Installing cicd-ai-toolkit for Gitee..."

# 检测架构
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# 下载二进制
URL="https://github.com/cicd-ai-toolkit/releases/download/${GITEE_VERSION}/cicd-runner-linux-${ARCH}"
echo "Downloading from $URL"

curl -fsSL "$URL" -o "${INSTALL_DIR}/cicd-runner"
chmod +x "${INSTALL_DIR}/cicd-runner"

# 验证安装
cicd-runner --version

# 配置 Gitee Token
if [ -z "$GITEE_TOKEN" ]; then
    echo "Please set GITEE_TOKEN environment variable"
    echo "export GITEE_TOKEN=your_token_here"
fi

echo "Installation complete!"
echo ""
echo "Add to your .gitee/workflows/review.yml:"
echo ""
cat <<'EOF'
jobs:
  ai-review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: cicd-ai-toolkit/action@v1
        with:
          run_skills: "code-reviewer"
EOF
```

## 4. 配置示例

```yaml
# .cicd-ai-toolkit.yaml

# Gitee 平台配置
platform:
  gitee:
    # API 端点 (默认: https://api.gitee.com/v5)
    api_url: "https://api.gitee.com/v5"

    # 企业版 ID (可选)
    enterprise_id: ""

    # Token 环境变量
    token_env: "GITEE_TOKEN"

    # 评论配置
    post_comment: true
    comment_style: "markdown"  # markdown | plain

    # 状态配置
    post_status: true
    status_context: "cicd-ai-review"

    # 失败处理
    fail_on_error: false
    timeout: "300s"
```

## 5. 依赖关系 (Dependencies)

- **Depends on**: [SPEC-PLAT-01](./SPEC-PLAT-01-Platform_Adapter.md) - Platform 接口定义
- **Related**: [SPEC-SEC-03](./SPEC-SEC-03-RBAC.md) - 企业版鉴权
- **Related**: [SPEC-DIST-01](./SPEC-DIST-01-Distribution.md) - 分发与安装

## 6. 验收标准 (Acceptance Criteria)

1. **环境检测**: 在 Gitee Go 环境中，`DetectGitee()` 返回 true
2. **API 调用**: 成功调用 Gitee API v5 获取 PR 信息
3. **企业版支持**: 使用 `enterprise_id` 能正确调用企业版 API
4. **评论发布**: 能在 Gitee PR 上成功发表评论
5. **状态更新**: 能正确更新 commit status
6. **Webhook 处理**: 能正确解析 Gitee Webhook 并转换为内部事件
7. **Workflow 兼容**: action.yml 能在 Gitee Go 中正常运行
8. **一键安装**: 安装脚本能在 Linux 环境成功安装
9. **私有市场**: 能发布插件到 Gitee 私有市场

## 7. Gitee API 差异说明

| 功能 | GitHub API | Gitee API v5 | 兼容性 |
|------|-----------|-------------|--------|
| **获取 PR** | `GET /repos/{owner}/{repo}/pulls/{number}` | `GET /repos/{owner}/{repo}/pulls/{number}` | ✅ 兼容 |
| **创建评论** | `POST /repos/{owner}/{repo}/issues/{number}/comments` | `POST /repos/{owner}/{repo}/pulls/{number}/comments` | ⚠️ 端点不同 |
| **更新状态** | `POST /repos/{owner}/{repo}/statuses/{sha}` | `POST /repos/{owner}/{repo}/statuses/{sha}` | ✅ 兼容 |
| **企业版** | - | `/enterprises/{id}/repos/...` | ❌ Gitee 特有 |
| **鉴权** | `Authorization: token {token}` | `Authorization: Bearer {token}` | ⚠️ 格式不同 |

## 8. 故障排查

```bash
# 检查 Gitee Token
curl -H "Authorization: Bearer $GITEE_TOKEN" https://api.gitee.com/v5/user

# 检查企业版 API
curl -H "Authorization: Bearer $GITEE_TOKEN" \
  https://api.gitee.com/v5/enterprises/{id}/repos

# 测试 Webhook 接收
# 查看 Gitee Go 日志中的 GITEE_EVENT_PATH 内容
```

## 9. 参考文档

- [Gitee API v5 文档](https://gitee.com/api/v5/swagger)
- [Gitee Go 文档](https://gitee.com/help)
- [Gitea Actions 文档](https://docs.gitea.com/usage/actions/overview)
