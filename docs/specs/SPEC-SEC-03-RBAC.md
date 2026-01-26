# SPEC-SEC-03: Role-Based Access Control (RBAC)

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24
**Covers**: PRD 4.0 Phase 3 (权限安全), 5.3 (安全要求)

## 1. 概述 (Overview)

随着 AI Agent 获得更多自主权，必须实现细粒度的访问控制。RBAC 系统定义谁（Subject）可以对什么资源（Resource）执行什么操作（Action）。本 Spec 定义 `cicd-ai-toolkit` 的权限模型。

## 2. 核心职责 (Core Responsibilities)

- **身份管理**: 管理用户和 Agent 的身份
- **角色定义**: 预定义角色（Admin, Developer, Viewer, Agent）
- **权限绑定**: 将角色与权限关联
- **访问决策**: 运行时权限检查
- **审计日志**: 记录所有授权决策

## 3. 详细设计 (Detailed Design)

### 3.1 权限模型 (Permission Model)

```
Subject (主体)
    │
    ├─ User: human user (GitHub user, GitLab user)
    ├─ Agent: AI agent (cicd-ai-toolkit)
    └─ Service: external service (GitHub Actions)
         │
         ▼
    Role (角色)
         │
         ├─ admin: 完全控制
         ├─ maintainer: 修改配置、管理 Skills
         ├─ developer: 读写代码、查看报告
         ├─ viewer: 只读
         └─ agent: 受限的 Agent 权限
         │
         ▼
    Permission (权限)
         │
         ├─ skill:read, skill:write, skill:delete
         ├─ report:read, report:write
         ├─ code:read, code:write
         ├─ pr:comment, pr:merge
         ├─ config:read, config:update
         └─ admin:*
         │
         ▼
    Resource (资源)
         │
         ├─ skills/*
         ├─ reports/*
         ├─ prs/*
         └─ config
```

### 3.2 角色定义 (Role Definitions)

| 角色 | 描述 | 权限 |
|------|------|------|
| **admin** | 系统管理员 | 所有权限 (`*:*`) |
| **maintainer** | 项目维护者 | 修改配置、管理 Skills、查看报告 |
| **developer** | 开发者 | 读写代码、查看报告、触发分析 |
| **viewer** | 只读访问者 | 查看报告、只读访问代码 |
| **agent** | AI Agent | 根据配置的受限权限 |
| **external** | 外部集成 | 通过 OAuth 授权的受限权限 |

### 3.3 配置格式 (Configuration)

在 `.cicd-ai-toolkit.yaml` 中定义 RBAC：

```yaml
# RBAC Configuration
rbac:
  enabled: true
  mode: "enforce"  # enforce | audit | disabled

  # 默认角色（未匹配到任何角色的用户）
  default_role: "viewer"

  # 角色绑定
  role_bindings:
    # 管理员
    - subject: "user:github:alice"
      role: "admin"

    # 维护者（团队）
    - subject: "team:github:maintainers"
      role: "maintainer"

    # 开发者（组织成员）
    - subject: "org:github:mycompany"
      role: "developer"

    # 只读（公开）
    - subject: "user:*"
      role: "viewer"

  # Agent 特殊权限
  agent_role:
    name: "agent"
    permissions:
      - "code:read"
      - "skill:read"
      - "report:write"
      - "pr:comment"
      # 禁止
      deny:
        - "code:write"
        - "pr:merge"
        - "config:update"

  # 资源级权限（可选的更细粒度控制）
  resource_policies:
    - resource: "skills:critical/*"
      permissions:
        - "maintainer": "read,write"
        - "developer": "read"
        - "viewer": "read"
```

### 3.4 Subject 格式 (Subject Format)

```
user:<platform>:<username>     # 单个用户
team:<platform>:<teamname>     # 团队
org:<platform>:<orgname>       # 组织
agent:<agent-name>             # AI Agent
service:<service-name>         # 外部服务
user:*                          # 所有用户（通配符）
```

### 3.5 Go 实现

```go
// RBAC Engine
type RBAC struct {
    enabled       bool
    mode          RBACMode
    roles         map[string]*Role
    bindings      []RoleBinding
    defaultRole   string
    agentRole     string
}

type RBACMode string

const (
    RBACModeEnforce RBACMode = "enforce" // 拒绝未授权访问
    RBACModeAudit   RBACMode = "audit"   // 记录但不拒绝
    RBACModeDisabled RBACMode = "disabled"
)

type Subject struct {
    Type    string // user, team, org, agent, service
    Platform string // github, gitlab, etc.
    ID      string // username, teamname, etc.
}

func (s *Subject) String() string {
    if s.Type == "agent" {
        return fmt.Sprintf("agent:%s", s.ID)
    }
    return fmt.Sprintf("%s:%s:%s", s.Type, s.Platform, s.ID)
}

type Permission struct {
    Resource string // skill, report, code, pr, config
    Action   string // read, write, delete, merge, etc.
}

func (p *Permission) String() string {
    return fmt.Sprintf("%s:%s", p.Resource, p.Action)
}

type Role struct {
    Name        string
    Permissions []string
    Deny        []string // 显式拒绝的权限
}

type RoleBinding struct {
    Subject string
    Role    string
}

// Check 验证权限
func (r *RBAC) Check(subject *Subject, permission *Permission) (bool, error) {
    if !r.enabled || r.mode == RBACModeDisabled {
        return true, nil // RBAC 禁用时允许所有
    }

    // 获取 Subject 的角色
    role := r.getRole(subject)
    if role == nil {
        role = r.roles[r.defaultRole]
    }

    // 检查显式拒绝
    for _, deny := range role.Deny {
        if r.matchPermission(deny, permission) {
            r.logAudit(subject, permission, false, "denied by role policy")
            return false, nil
        }
    }

    // 检查允许
    for _, perm := range role.Permissions {
        if r.matchPermission(perm, permission) {
            r.logAudit(subject, permission, true, "allowed by role")
            return true, nil
        }
    }

    r.logAudit(subject, permission, false, "no matching permission")
    return r.mode == RBACModeAudit, nil
}

func (r *RBAC) matchPermission(pattern string, perm *Permission) bool {
    // 支持 "resource:*" 和 "*:action" 通配符
    parts := strings.Split(pattern, ":")
    if len(parts) != 2 {
        return false
    }

    resourceMatch := parts[0] == "*" || parts[0] == perm.Resource
    actionMatch := parts[1] == "*" || parts[1] == perm.Action

    return resourceMatch && actionMatch
}

func (r *RBAC) getRole(subject *Subject) *Role {
    // 精确匹配
    subjectStr := subject.String()
    for _, binding := range r.bindings {
        if binding.Subject == subjectStr {
            return r.roles[binding.Role]
        }
    }

    // 通配符匹配
    for _, binding := range r.bindings {
        if r.matchSubject(binding.Subject, subjectStr) {
            return r.roles[binding.Role]
        }
    }

    return nil
}

func (r *RBAC) matchSubject(pattern, subject string) bool {
    // user:* 匹配所有 user
    if strings.HasSuffix(pattern, ":*") {
        prefix := strings.TrimSuffix(pattern, ":*")
        return strings.HasPrefix(subject, prefix)
    }
    return pattern == subject
}

func (r *RBAC) logAudit(subject *Subject, perm *Permission, allowed bool, reason string) {
    log.WithFields(log.Fields{
        "subject":   subject.String(),
        "permission": perm.String(),
        "allowed":   allowed,
        "reason":    reason,
        "rbac_mode": r.mode,
    }).Info("RBAC decision")
}
```

### 3.6 使用示例

```go
// 在 Runner 中使用 RBAC
func (r *Runner) PostComment(ctx context.Context, comment string) error {
    subject := r.getSubject() // 从 Platform Context 获取
    permission := &Permission{Resource: "pr", Action: "comment"}

    allowed, err := r.rbac.Check(subject, permission)
    if err != nil {
        return err
    }

    if !allowed {
        return fmt.Errorf("permission denied: %s cannot %s:%s",
            subject.String(), permission.Resource, permission.Action)
    }

    return r.platform.PostComment(ctx, comment)
}
```

### 3.7 Agent 身份 (Agent Identity)

Agent 使用专用的受限身份：

```yaml
# Agent 角色配置
agent_role:
  name: "cicd-ai-agent"
  display_name: "AI Code Review Agent"

  # 基础权限
  permissions:
    - "code:read"
    - "skill:read"
    - "report:write"
    - "pr:comment"
    - "mcp:read"  # 只读 MCP 调用

  # 禁止的操作（安全边界）
  deny:
    - "code:write"        # 不能直接修改代码
    - "pr:merge"          # 不能自动合并
    - "config:update"     # 不能修改配置
    - "secret:read"       # 不能读取敏感信息
    - "mcp:write"         # 不能通过 MCP 写入

  # 临时权限升级（需要人类批准）
  elevate:
    - permission: "code:write"
      requires: "human_approval"
      duration: "1h"
      max_uses: 1
```

### 3.8 权限升级 (Permission Elevation)

高风险操作需要人类批准：

```go
type ElevationRequest struct {
    ID        string
    Subject   *Subject
    Permission *Permission
    Reason    string
    ExpiresAt time.Time
    Approved  bool
}

func (r *RBAC) RequestElevation(ctx context.Context, subject *Subject, perm *Permission, reason string) (*ElevationRequest, error) {
    req := &ElevationRequest{
        ID:        uuid.New().String(),
        Subject:   subject,
        Permission: perm,
        Reason:    reason,
        ExpiresAt: time.Now().Add(1 * time.Hour),
    }

    // 创建批准请求
    r.platform.CreateApprovalRequest(req)

    return req, nil
}

// 检查权限（支持升级）
func (r *RBAC) CheckWithElevation(ctx context.Context, subject *Subject, perm *Permission) (bool, error) {
    allowed, err := r.Check(subject, perm)
    if err != nil {
        return false, err
    }

    if allowed {
        return true, nil
    }

    // 检查是否有有效的权限升级
    if r.hasValidElevation(subject, perm) {
        return true, nil
    }

    return false, nil
}
```

### 3.9 审计日志 (Audit Logging)

所有权限检查记录审计日志：

```json
{
  "timestamp": "2026-01-24T10:00:00Z",
  "event": "rbac_check",
  "subject": "user:github:alice",
  "permission": "pr:merge",
  "resource": "prs/123",
  "allowed": false,
  "reason": "denied by role policy",
  "role": "developer",
  "trace_id": "abc-123"
}
```

### 3.10 与平台集成 (Platform Integration)

从 Git 平台获取用户身份：

```go
func (p *GitHubAdapter) GetSubject(ctx context.Context) (*Subject, error) {
    // 从 JWT token 或 API 获取当前用户
    user, err := p.client.Users.Get(ctx, "")
    if err != nil {
        return nil, err
    }

    return &Subject{
        Type:    "user",
        Platform: "github",
        ID:      user.GetLogin(),
    }, nil
}
```

### 3.11 配置验证 (Configuration Validation)

```go
func ValidateRBACConfig(config *RBACConfig) error {
    // 验证角色存在
    for _, binding := range config.RoleBindings {
        if _, ok := config.Roles[binding.Role]; !ok {
            return fmt.Errorf("role not found: %s", binding.Role)
        }
    }

    // 验证默认角色存在
    if _, ok := config.Roles[config.DefaultRole]; !ok {
        return fmt.Errorf("default role not found: %s", config.DefaultRole)
    }

    // 验证权限格式
    for _, role := range config.Roles {
        for _, perm := range role.Permissions {
            if !isValidPermission(perm) {
                return fmt.Errorf("invalid permission: %s", perm)
            }
        }
    }

    return nil
}
```

## 4. 依赖关系 (Dependencies)

- **Related**: [SPEC-GOV-01](./SPEC-GOV-01-Policy_As_Code.md) - OPA 策略引擎
- **Related**: [SPEC-SEC-01](./SPEC-SEC-01-Sandboxing.md) - 沙箱隔离
- **Related**: [SPEC-OPS-01](./SPEC-OPS-01-Observability.md) - 审计日志

## 5. 验收标准 (Acceptance Criteria)

1. **角色隔离**: Viewer 角色无法执行写入操作
2. **Agent 限制**: Agent 无法自动合并 PR
3. **权限升级**: 高风险操作需要人类批准
4. **审计完整**: 所有权限决策有审计记录
5. **配置验证**: 无效配置在启动时拒绝
6. **平台集成**: 能从 GitHub/GitLab 获取用户身份
7. **性能**: 权限检查延迟 < 1ms

## 6. 安全最佳实践

1. **最小权限原则**: 默认角色为 viewer
2. **显式拒绝**: 使用 deny 列表明确禁止危险操作
3. **权限隔离**: Agent 使用专用受限身份
4. **定期审查**: 定期审查角色绑定
5. **MFA**: 管理操作要求多因素认证
6. **临时凭证**: 权限升级使用短期有效的凭证
