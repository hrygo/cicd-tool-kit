# SPEC-PLAT-03: GitLab Adapter

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24
**Covers**: PRD 2.1 (Tier 2 Platform), 3.2 (Phase 2)

## 1. 概述 (Overview)

GitLab 是 Tier 2 平台支持。与 GitHub 不同，GitLab 使用 Merge Request (MR) 而非 Pull Request，且 API 结构和 Webhook 格式有显著差异。本 Spec 定义 GitLab 适配器的实现细节。

## 2. 核心职责 (Core Responsibilities)

- **API 适配**: 将 GitLab REST API v4 映射到统一的 `Platform` 接口
- **Webhook 处理**: 接收和解析 GitLab Merge Request Webhook 事件
- **权限管理**: 处理 GitLab PAT (Personal Access Token) 和 Project Access Token
- **Comment 格式**: 支持 GitLab Markdown 格式的评论和 Discussion

## 3. 详细设计 (Detailed Design)

### 3.1 接口实现 (Platform Interface Implementation)

```go
// GitLabAdapter implements Platform interface for GitLab
type GitLabAdapter struct {
    client   *gitlab.Client
    baseURL  string
    token    string
    projectID string // Could be numeric ID or path-encoded namespace/project
}

// GetMergeRequest fetches MR details (maps to GetPullRequest)
func (g *GitLabAdapter) GetPullRequest(ctx context.Context, id string) (*UnifiedPR, error) {
    // GitLab uses IID (merge request IID) + Project ID
    // id format: "project_path:mr_iid" or "123:456"
    parts := strings.Split(id, ":")
    projectID, mrIID, err := parseID(id)

    mr, _, err := g.client.MergeRequests.GetMergeRequest(projectID, mrIID, nil)
    if err != nil {
        return nil, err
    }

    return &UnifiedPR{
        ID:          fmt.Sprintf("%d", mr.IID),
        Title:       mr.Title,
        Description: mr.Description,
        BaseRef:     mr.TargetBranch,
        HeadRef:     mr.SourceBranch,
        Author:      mr.Author.Username,
        ProjectPath: projectID,
    }, nil
}

// GetDiff retrieves MR diff
func (g *GitLabAdapter) GetDiff(ctx context.Context, id string) ([]byte, error) {
    parts := strings.Split(id, ":")
    projectID, mrIID, _ := parseID(id)

    diff, _, err := g.client.MergeRequests.GetMergeRequestDiff(projectID, mrIID, nil)
    if err != nil {
        return nil, err
    }

    // Convert GitLab diff format to unified format
    return normalizeGitLabDiff(diff), nil
}

// PostComment adds a general comment to the MR
func (g *GitLabAdapter) PostComment(ctx context.Context, id string, body string) error {
    parts := strings.Split(id, ":")
    projectID, mrIID, _ := parseID(id)

    note := &gitlab.CreateMergeRequestNoteOptions{
        Body: gitlab.Ptr(body),
    }

    _, _, err := g.client.Notes.CreateMergeRequestNote(projectID, mrIID, note)
    return err
}

// PostReview adds inline comments (GitLab Discussions)
func (g *GitLabAdapter) PostReview(ctx context.Context, id string, comments []InlineComment) error {
    parts := strings.Split(id, ":")
    projectID, mrIID, _ := parseID(id)

    for _, comment := range comments {
        // GitLab uses position-based inline comments
        position := &gitlab.Position{
            BaseSHA:  comment.BaseSHA,
            HeadSHA:  comment.HeadSHA,
            FilePath: gitlab.Ptr(comment.File),
            NewLine:  gitlab.Ptr(comment.Line),
        }

        note := &gitlab.CreateMergeRequestDiscussionNoteOptions{
            Body:    gitlab.Ptr(comment.Message),
            Position: position,
        }

        _, _, err := g.client.Discussions.CreateMergeRequestDiscussionNote(projectID, mrIID, note)
        if err != nil {
            // Log error but continue with other comments
            log.Errorf("Failed to post inline comment: %v", err)
        }
    }

    return nil
}

// CreateCheckRun creates a pipeline status (GitLab equivalent)
func (g *GitLabAdapter) CreateCheckRun(ctx context.Context, sha string, name string) (string, error) {
    // GitLab uses Commit Status API
    status := &gitlab.SetCommitStatusOptions{
        Name:      gitlab.Ptr(name),
        State:     gitlab.Ptr(gitlab.Pending),
        Context:   gitlab.Ptr("cicd-ai-toolkit"),
    }

    s, _, err := g.client.Commits.SetCommitStatus(g.projectID, sha, status)
    if err != nil {
        return "", err
    }

    return s.SHA, nil
}

// UpdateCheckRun updates the commit status
func (g *GitLabAdapter) UpdateCheckRun(ctx context.Context, runID string, status string, output CheckOutput) error {
    // Map status strings
    var gitlabState gitlab.BuildStateValue
    switch status {
    case "completed":
        if output.Success {
            gitlabState = gitlab.Success
        } else {
            gitlabState = gitlab.Failed
        }
    case "in_progress":
        gitlabState = gitlab.Running
    case "queued":
        gitlabState = gitlab.Pending
    default:
        gitlabState = gitlab.Failed
    }

    opts := &gitlab.SetCommitStatusOptions{
        Name:        gitlab.Ptr("cicd-ai-toolkit"),
        State:       gitlab.Ptr(gitlabState),
        Description: gitlab.Ptr(output.Summary),
        TargetURL:   gitlab.Ptr(output.URL),
    }

    _, _, err := g.client.Commits.SetCommitStatus(g.projectID, runID, opts)
    return err
}
```

### 3.2 环境检测 (Environment Detection)

```go
func DetectGitLab(env map[string]string) bool {
    return env["GITLAB_CI"] == "true" ||
           env["CI_SERVER_URL"] != "" && env["GITLAB_TOKEN"] != ""
}
```

**GitLab CI 环境变量** (自动注入):

| 变量 | 说明 | 用途 |
|------|------|------|
| `GITLAB_CI` | 始终为 "true" | 检测 GitLab CI 环境 |
| `CI_PROJECT_ID` | 项目 ID | API 调用 |
| `CI_MERGE_REQUEST_IID` | MR IID | 获取 MR 详情 |
| `CI_MERGE_REQUEST_TARGET_BRANCH_NAME` | 目标分支 | Diff 对比 |
| `CI_COMMIT_SHA` | 当前 commit SHA | Status 更新 |
| `CI_SERVER_URL` | GitLab 实例 URL | API endpoint |

### 3.3 认证 (Authentication)

GitLab 支持多种 Token 类型：

| Token 类型 | 用途 | 环境变量 |
|-----------|------|----------|
| **Personal Access Token** | 开发测试 | `GITLAB_TOKEN` |
| **Project Access Token** | 项目级自动化 | `GITLAB_TOKEN` |
| **OAuth2 Token** | 企业级 SSO | `GITLAB_OAUTH_TOKEN` |
| **Job Token (CI_JOB_TOKEN)** | CI 内部调用 | `CI_JOB_TOKEN` |

```go
func NewGitLabAdapter(baseURL, token string) (*GitLabAdapter, error) {
    if token == "" {
        token = os.Getenv("GITLAB_TOKEN")
    }
    if token == "" {
        token = os.Getenv("CI_JOB_TOKEN") // Fallback to CI_JOB_TOKEN
    }

    if baseURL == "" {
        baseURL = os.Getenv("CI_SERVER_URL")
        if baseURL == "" {
            baseURL = "https://gitlab.com"
        }
    }

    client, err := gitlab.NewClient(token, gitlab.WithBaseURL(baseURL+"/api/v4"))
    if err != nil {
        return nil, err
    }

    projectID := os.Getenv("CI_PROJECT_ID")
    if projectID == "" {
        // For local development, may need explicit config
        return nil, fmt.Errorf("CI_PROJECT_ID not set")
    }

    return &GitLabAdapter{
        client:    client,
        baseURL:   baseURL,
        token:     token,
        projectID: projectID,
    }, nil
}
```

### 3.4 Webhook 处理 (Webhook Handler)

GitLab Merge Request Webhook 事件结构：

```go
// GitLabMergeRequestEvent represents GitLab MR webhook
type GitLabMergeRequestEvent struct {
    ObjectKind string `json:"object_kind"` // "merge_request"
    User       struct {
        Name     string `json:"name"`
        Username string `json:"username"`
    } `json:"user"`
    Project struct {
        ID          int    `json:"id"`
        Name        string `json:"name"`
        PathWithNamespace string `json:"path_with_namespace"`
    } `json:"project"`
    ObjectAttributes struct {
        ID              int    `json:"id"`
        IID             int    `json:"iid"`
        Title           string `json:"title"`
        Description     string `json:"description"`
        SourceBranch    string `json:"source_branch"`
        TargetBranch    string `json:"target_branch"`
        State           string `json:"state"`
        Action          string `json:"action"` // "open", "update", "merge", "close"
        AuthorID        int    `json:"author_id"`
        LastCommit      struct {
            ID        string `json:"id"`
            Message   string `json:"message"`
            Timestamp string `json:"timestamp"`
        } `json:"last_commit"`
    } `json:"object_attributes"`
}

// Convert to UnifiedPR
func (e *GitLabMergeRequestEvent) ToUnifiedPR() *UnifiedPR {
    return &UnifiedPR{
        ID:          fmt.Sprintf("%d", e.ObjectAttributes.IID),
        Title:       e.ObjectAttributes.Title,
        Description: e.ObjectAttributes.Description,
        BaseRef:     e.ObjectAttributes.TargetBranch,
        HeadRef:     e.ObjectAttributes.SourceBranch,
        Author:      e.User.Username,
        ProjectID:   e.Project.ID,
        ProjectPath: e.Project.PathWithNamespace,
        SHA:         e.ObjectAttributes.LastCommit.ID,
    }
}
```

### 3.5 Markdown 格式差异 (Markdown Differences)

GitLab Markdown 与 GitHub 有细微差异：

| 特性 | GitHub | GitLab | 处理 |
|------|--------|--------|------|
| 任务列表 | `- [ ]` | `- [ ]` | 兼容 |
| 代码块 | \```lang | \```lang | 兼容 |
| 引用 | `@user` | `@user` | 兼容 |
| Issue/MR 链接 | `#123` | `!123` (MR) | 需转换 |

```go
// Convert GitHub-style references to GitLab
func adaptMarkdownForGitLab(md string) string {
    // Convert #123 to !123 for MR references in comments
    // This is a simplified version; production should use proper parsing
    md = regexp.MustCompile(`#(\d+)`).ReplaceAllString(md, "!$1")
    return md
}
```

### 3.6 与 GitHub 的关键差异

| 功能 | GitHub | GitLab | 实现注意事项 |
|------|--------|--------|-------------|
| **PR 概念** | Pull Request | Merge Request | 统称为 PR in UnifiedPR |
| **ID 格式** | 单一数字 | ProjectID + IID | 复合 ID: `"project:iid"` |
| **Review API** | Reviews + Comments | Discussions API | 需分别处理 |
| **Status Checks** | Check Runs API | Commit Status API | 功能对等，API 不同 |
| **Inline Comment** | Line-based + Position-based | Position-based (SHA required) | 需要 BaseSHA/HeadSHA |
| **Webhook** | Separate events | Single event with action | 根据 action 字段分发 |

### 3.7 工厂集成 (Factory Integration)

```go
// In SPEC-PLAT-01 Platform factory
func NewPlatform(env Environment) (Platform, error) {
    if env["GITHUB_ACTIONS"] == "true" {
        return NewGitHubAdapter(env["GITHUB_TOKEN"])
    }
    if env["GITEE_GO"] == "true" {
        return NewGiteeAdapter(env["GITEE_TOKEN"])
    }
    if env["GITLAB_CI"] == "true" || env["CI_SERVER_URL"] != "" {
        return NewGitLabAdapter(env["CI_SERVER_URL"], env["GITLAB_TOKEN"])
    }
    return nil, fmt.Errorf("unknown platform")
}
```

## 4. 配置 (Configuration)

在 `.cicd-ai-toolkit.yaml` 中添加 GitLab 配置：

```yaml
platform:
  gitlab:
    # API endpoint (defaults to https://gitlab.com)
    base_url: "https://gitlab.example.com"
    # Token source (env var name)
    token_env: "GITLAB_TOKEN"
    # Project ID (can be auto-detected in CI)
    project_id: ""  # Optional: "123" or "group/project"
    # Comment behavior
    post_comment: true
    # Resolve existing discussions before posting new ones
    resolve_discussions: false
    # Maximum comment length (GitLab default: 1MB)
    max_comment_length: 1000000
```

## 5. 依赖关系 (Dependencies)

- **Library**: `github.com/xanzy/go-gitlab` (Official GitLab Go SDK)
- **Used by**: [SPEC-PLAT-01](./SPEC-PLAT-01-Platform_Adapter.md) Platform interface
- **Related**: [SPEC-DIST-01](./SPEC-DIST-01-Distribution.md) for container image with GitLab support

## 6. 验收标准 (Acceptance Criteria)

1. **环境检测**: 在 GitLab CI 环境中，Runner 能自动检测并初始化 GitLab Adapter
2. **MR 获取**: 能正确获取 MR 的 title, description, source/target branch
3. **Diff 获取**: 能获取并解析 MR 的 diff
4. **评论发布**: 能在 MR 下发表普通评论
5. **行内评论**: 能在指定文件的指定行发表行内评论
6. **状态更新**: 能更新 commit 的 status (pending/running/success/failed)
7. **Self-Managed**: 能连接到自托管的 GitLab 实例 (非 gitlab.com)
8. **权限边界**: 使用 CI_JOB_TOKEN 时，仅能访问当前项目

## 7. 限制与已知问题 (Limitations)

1. **CI_JOB_TOKEN 限制**: 仅能在当前项目内使用，无法访问其他项目
2. **MR Draft 状态**: GitLab Draft MR (WIP) 需要特殊处理
3. **Discussion Resolution**: GitLab 允许 resolved discussions，需要适配 UI
4. **批量 API**: GitLab API 有速率限制，批量操作需要限流
