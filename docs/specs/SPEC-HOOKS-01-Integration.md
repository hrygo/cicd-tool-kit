# SPEC-HOOKS-01: Claude Code Hooks Integration

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24
**Covers**: PRD 1.1 (Claude Code Hooks 机制)

## 1. 概述 (Overview)

Claude Code 提供 Hooks 机制，允许在特定生命周期事件执行自定义脚本。本 Spec 定义 `cicd-ai-toolkit` 如何集成和利用这些 Hooks 来增强 CI/CD 流程控制、自定义行为和扩展能力。

## 2. 核心职责 (Core Responsibilities)

- **Hook 管理**: 发现、注册、执行 Claude Code Hooks
- **事件处理**: 响应 Setup、User Prompt、Pre-commit、Tool、Post-commit 事件
- **上下文注入**: 在 Hook 执行时提供 CI/CD 上下文
- **结果聚合**: 收集 Hook 输出并集成到 Runner 流程
- **沙箱执行**: 确保 Hooks 在安全环境中执行

## 3. 详细设计 (Detailed Design)

### 3.1 Claude Code Hooks 类型

Claude Code 支持以下 Hook 类型：

| Hook 类型 | 触发时机 | CI/CD 用途 |
|-----------|----------|------------|
| **setup** | 初始化时运行 | 环境准备、依赖安装、配置验证 |
| **user-prompt** | 用户提示触发 | Prompt 预处理、输入验证 |
| **pre-commit** | Git 操作前 | 代码格式化、Lint 检查 |
| **post-commit** | Git 操作后 | 通知触发、状态同步 |
| **tool** | 工具调用钩子 | 工具调用审计、权限检查 |

### 3.2 Hook 配置格式

Hooks 通过 `.claude/hooks.yaml` 或在 `.cicd-ai-toolkit.yaml` 中配置：

```yaml
# .cicd-ai-toolkit.yaml
hooks:
  # Setup Hooks - 初始化时执行
  setup:
    - name: "verify-environment"
      run: "./scripts/verify-env.sh"
      timeout: "30s"
      fail_on_error: true

    - name: "install-dependencies"
      run: "pip install -r requirements.txt"
      condition: "{{ .Language == 'python' }}"

  # User Prompt Hooks - Prompt 处理前执行
  user_prompt:
    - name: "validate-input"
      run: "./scripts/validate-prompt.py"
      input_from: "stdin"

    - name: "enrich-context"
      run: "./scripts/fetch-jira-ticket.sh"
      env:
        JIRA_URL: "${JIRA_URL}"

  # Pre-commit Hooks - Git 提交前
  pre_commit:
    - name: "format-code"
      run: "gofmt -w ."
      allowed_tools: ["write"]

    - name: "run-linters"
      run: "./scripts/lint.sh"

  # Tool Hooks - 工具调用时
  tool:
    - name: "audit-edit"
      match_tool: "edit_file"
      run: "./scripts/audit-edit.sh"
      run_as: "readonly"

    - name: "log-mcp-call"
      match_tool: "mcp_server_request"
      run: "./scripts/log-mcp.sh"
      async: true
```

### 3.3 Hook 执行流程

```go
// HookExecutor manages hook execution
type HookExecutor struct {
    hooks     map[string][]Hook
    timeout   time.Duration
    sandbox   *Sandbox
    logger    *zap.Logger
}

type Hook struct {
    Name         string            `yaml:"name"`
    Run          string            `yaml:"run"`
    Timeout      time.Duration     `yaml:"timeout"`
    Condition    string            `yaml:"condition"`
    InputFrom    string            `yaml:"input_from"`
    Env          map[string]string `yaml:"env"`
    FailOnError  bool              `yaml:"fail_on_error"`
    MatchTool    string            `yaml:"match_tool"`
    RunAs        string            `yaml:"run_as"`
    Async        bool              `yaml:"async"`
}

type HookContext struct {
    EventType    string            // setup, user_prompt, pre_commit, etc.
    Language     string            // go, python, javascript
    Platform     string            // github, gitlab, gitee
    PRNumber     string
    CommitSHA    string
    ToolName     string            // For tool hooks
    Variables    map[string]string // Additional context
}

func (he *HookExecutor) Execute(ctx context.Context, hookType string, hookCtx *HookContext) error {
    hooks, ok := he.hooks[hookType]
    if !ok || len(hooks) == 0 {
        return nil
    }

    for _, hook := range hooks {
        // Check condition
        if hook.Condition != "" {
            matched, err := he.evaluateCondition(hook.Condition, hookCtx)
            if err != nil || !matched {
                continue
            }
        }

        // Check tool match
        if hook.MatchTool != "" && hook.MatchTool != hookCtx.ToolName {
            continue
        }

        // Execute hook
        if hook.Async {
            go he.executeHook(ctx, hook, hookCtx)
        } else {
            if err := he.executeHook(ctx, hook, hookCtx); err != nil {
                if hook.FailOnError {
                    return fmt.Errorf("hook %s failed: %w", hook.Name, err)
                }
                he.logger.Warn("hook failed, continuing", zap.String("hook", hook.Name), zap.Error(err))
            }
        }
    }

    return nil
}

func (he *HookExecutor) executeHook(ctx context.Context, hook Hook, hookCtx *HookContext) error {
    start := time.Now()

    // Build command
    cmd := he.buildCommand(hook, hookCtx)

    // Set timeout
    timeout := hook.Timeout
    if timeout == 0 {
        timeout = he.timeout
    }
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    // Execute in sandbox
    if hook.RunAs == "readonly" {
        return he.sandbox.ExecuteReadOnly(ctx, cmd)
    }

    output, err := cmd.CombinedOutput()
    he.logger.Info("hook completed",
        zap.String("hook", hook.Name),
        zap.Duration("duration", time.Since(start)),
        zap.Int("exit_code", exitCode),
        zap.String("output", string(output)))

    return err
}

func (he *HookExecutor) buildCommand(hook Hook, hookCtx *HookContext) *exec.Cmd {
    parts := strings.Fields(hook.Run)
    cmd := exec.Command(parts[0], parts[1:]...)

    // Set environment
    env := os.Environ()
    for k, v := range hook.Env {
        // Expand variables
        v = os.ExpandEnv(v)
        v = he.expandTemplate(v, hookCtx)
        env = append(env, fmt.Sprintf("%s=%s", k, v))
    }

    // Add hook context to environment
    env = append(env, he.buildHookEnv(hookCtx)...)
    cmd.Env = env

    return cmd
}

func (he *HookExecutor) buildHookEnv(hookCtx *HookContext) []string {
    return []string{
        fmt.Sprintf("HOOK_EVENT_TYPE=%s", hookCtx.EventType),
        fmt.Sprintf("HOOK_LANGUAGE=%s", hookCtx.Language),
        fmt.Sprintf("HOOK_PLATFORM=%s", hookCtx.Platform),
        fmt.Sprintf("HOOK_PR_NUMBER=%s", hookCtx.PRNumber),
        fmt.Sprintf("HOOK_COMMIT_SHA=%s", hookCtx.CommitSHA),
        fmt.Sprintf("HOOK_TOOL_NAME=%s", hookCtx.ToolName),
    }
}

func (he *HookExecutor) expandTemplate(template string, hookCtx *HookContext) string {
    // Simple template expansion: {{ .Variable }}
    re := regexp.MustCompile(`\{\{\s*\.\?(\w+)\s*\}\}`)
    result := re.ReplaceAllStringFunc(template, func(match string) string {
        name := re.FindStringSubmatch(match)[1]
        if val, ok := hookCtx.Variables[name]; ok {
            return val
        }
        // Try built-in fields
        switch name {
        case "Language":
            return hookCtx.Language
        case "Platform":
            return hookCtx.Platform
        case "PRNumber":
            return hookCtx.PRNumber
        default:
            return match
        }
    })
    return result
}
```

### 3.4 CI/CD 场景 Hook 示例

#### 场景 1: Setup Hook - 环境验证

```bash
#!/bin/bash
# scripts/verify-env.sh

set -e

echo "Verifying CI/CD environment..."

# Check required tools
command -v git >/dev/null 2>&1 || { echo "git required"; exit 1; }
command -v claude >/dev/null 2>&1 || { echo "claude CLI required"; exit 1; }

# Check API keys
if [[ -z "$ANTHROPIC_API_KEY" ]]; then
    echo "ANTHROPIC_API_KEY not set"
    exit 1
fi

# Verify Claude Code version
CLAUDE_VERSION=$(claude --version)
echo "Claude Code version: $CLAUDE_VERSION"

# Check disk space
AVAILABLE=$(df -BG . | tail -1 | awk '{print $4}' | sed 's/G//')
if (( AVAILABLE < 5 )); then
    echo "Insufficient disk space: ${AVAILABLE}G available, 5G required"
    exit 1
fi

echo "Environment verification complete"
```

#### 场景 2: User Prompt Hook - 上下文增强

```python
#!/usr/bin/env python3
# scripts/enrich-context.py

import os
import sys
import json
import re

def extract_jira_ticket(text):
    """Extract JIRA ticket ID from text"""
    pattern = r'[A-Z]+-\d+'
    matches = re.findall(pattern, text)
    return matches[0] if matches else None

def fetch_ticket_details(ticket_id):
    """Fetch JIRA ticket details"""
    jira_url = os.environ.get('JIRA_URL')
    jira_token = os.environ.get('JIRA_TOKEN')

    if not jira_url or not jira_token:
        return None

    import requests
    response = requests.get(
        f"{jira_url}/rest/api/2/issue/{ticket_id}",
        headers={"Authorization": f"Bearer {jira_token}"}
    )

    if response.status_code == 200:
        issue = response.json()
        return {
            "id": issue["key"],
            "summary": issue["fields"]["summary"],
            "description": issue["fields"]["description"],
            "status": issue["fields"]["status"]["name"]
        }
    return None

def main():
    # Read stdin
    prompt_text = sys.stdin.read()

    # Extract JIRA ticket
    ticket_id = extract_jira_ticket(prompt_text)

    if ticket_id:
        details = fetch_ticket_details(ticket_id)

        if details:
            # Append to output
            enrichment = f"\n\n--- Context from JIRA ---\n"
            enrichment += f"Ticket: {details['id']}\n"
            enrichment += f"Summary: {details['summary']}\n"
            enrichment += f"Status: {details['status']}\n"

            # Write to temp file for Claude to read
            with open('/tmp/jira-context.txt', 'w') as f:
                f.write(enrichment)

            print(f"Enriched context with JIRA ticket {ticket_id}", file=sys.stderr)
            return 0

    return 0

if __name__ == "__main__":
    sys.exit(main())
```

#### 场景 3: Tool Hook - 审计文件编辑

```bash
#!/bin/bash
# scripts/audit-edit.sh

# Read hook environment
TOOL_NAME="${HOOK_TOOL_NAME}"
FILE_PATH="${HOOK_FILE_PATH}"

echo "Auditing edit operation: $TOOL_NAME on $FILE_PATH"

# Security checks
if [[ "$FILE_PATH" == *"config"*"secret"* ]]; then
    echo "SECURITY: Attempt to modify secrets config blocked"
    exit 1
fi

if [[ "$FILE_PATH" == *"go.mod"* ]] || [[ "$FILE_PATH" == *"package.json"* ]]; then
    echo "SECURITY: Dependency file modification detected"
    # Log but allow
    echo "$(date): Dependency modification: $FILE_PATH" >> /var/log/cicd-audit.log
fi

# Log all edits
echo "$(date): $TOOL_NAME: $FILE_PATH" >> /var/log/cicd-edit.log

exit 0
```

#### 场景 4: Post-commit Hook - 通知触发

```bash
#!/bin/bash
# scripts/notify-commit.sh

WEBHOOK_URL="${SLACK_WEBHOOK_URL}"
COMMIT_SHA="${HOOK_COMMIT_SHA}"
PR_NUMBER="${HOOK_PR_NUMBER}"

# Get commit message
MESSAGE=$(git log -1 --pretty=%B "$COMMIT_SHA")

# Build notification
PAYLOAD=$(cat <<EOF
{
  "text": "New commit in PR #$PR_NUMBER",
  "blocks": [
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": "*New Commit*\n*SHA:* <$REPO_URL/commit/$COMMIT_SHA|$COMMIT_SHA>\n*Message:* $MESSAGE"
      }
    }
  ]
}
EOF
)

# Send notification
curl -X POST -H "Content-Type: application/json" -d "$PAYLOAD" "$WEBHOOK_URL"
```

### 3.5 Hook 发现机制

Runner 按以下顺序发现 Hooks：

1. **内置 Hooks**: `skills/*/hooks.yaml`
2. **项目 Hooks**: `.claude/hooks.yaml`
3. **用户 Hooks**: `~/.claude/hooks.yaml`
4. **配置 Hooks**: `.cicd-ai-toolkit.yaml` 中的 `hooks` 字段

```go
func (he *HookExecutor) DiscoverHooks() error {
    he.hooks = make(map[string][]Hook)

    // Load from built-in skills
    if err := he.loadHooksFromDir("skills/*/", "hooks.yaml"); err != nil {
        return err
    }

    // Load from project
    if err := he.loadHooksFile(".claude/hooks.yaml"); err != nil {
        return err
    }

    // Load from user home
    homeDir, _ := os.UserHomeDir()
    if err := he.loadHooksFile(filepath.Join(homeDir, ".claude/hooks.yaml")); err != nil {
        return err
    }

    // Load from toolkit config
    if err := he.loadHooksFromConfig(".cicd-ai-toolkit.yaml"); err != nil {
        return err
    }

    return nil
}
```

### 3.6 Hook 输出处理

Hooks 可以通过以下方式影响 Runner 行为：

| 输出类型 | 格式 | 效果 |
|----------|------|------|
| **环境变量** | `export KEY=value` | 添加到后续环境 |
| **修改输入** | 写入 stdout | 传递给 Claude |
| **JSON 元数据** | `::set-output name=value` | 可在后续 Hook 使用 |
| **退出码** | 非 0 | 根据 `fail_on_error` 决定是否失败 |

```bash
# Hook 输出示例
echo "::set-output name=ticket-id::ABC-123"
echo "export ENHANCED_CONTEXT=true"
```

### 3.7 Hook 沙箱执行

为确保安全，Hooks 在沙箱中执行：

```go
type Sandbox struct {
    rootfs     string
    mounts     []Mount
    network    NetworkMode
}

func (s *Sandbox) ExecuteReadOnly(ctx context.Context, cmd *exec.Cmd) error {
    // Execute with read-only mounts
    return s.execute(ctx, cmd, ReadOnly)
}

type NetworkMode string

const (
    NetworkNone   NetworkMode = "none"
    NetworkStrict NetworkMode = "strict"  // Only allow specific endpoints
    NetworkFull   NetworkMode = "full"
)
```

### 3.8 Hook 性能考虑

| Hook 类型 | 超时建议 | 异步支持 |
|-----------|----------|----------|
| setup | 60s | ❌ |
| user_prompt | 10s | ❌ |
| pre_commit | 30s | ❌ |
| post_commit | 30s | ✅ |
| tool | 5s | ✅ |

### 3.9 Hook 调试

启用详细日志：

```yaml
# .cicd-ai-toolkit.yaml
hooks:
  debug: true
  log_file: "/var/log/cicd-hooks.log"
  preserve_output: true  # Keep hook output for debugging
```

## 4. 依赖关系 (Dependencies)

- **Related**: [SPEC-CORE-01](./SPEC-CORE-01-Runner_Lifecycle.md) - Runner 生命周期集成
- **Related**: [SPEC-SEC-01](./SPEC-SEC-01-Sandboxing.md) - 沙箱执行
- **Related**: [SPEC-OPS-01](./SPEC-OPS-01-Observability.md) - Hook 日志记录

## 5. 验收标准 (Acceptance Criteria)

1. **Hook 发现**: Runner 能发现并加载所有来源的 Hooks
2. **条件执行**: 带 `condition` 的 Hook 仅在条件满足时执行
3. **Tool 匹配**: Tool Hooks 仅在匹配的工具调用时执行
4. **上下文注入**: Hook 能访问完整的 CI/CD 上下文
5. **错误处理**: `fail_on_error: false` 时 Hook 失败不阻塞流程
6. **异步执行**: `async: true` 的 Hook 并发执行
7. **沙箱隔离**: Hook 无法修改宿主敏感文件
8. **性能**: 所有同步 Hooks 总执行时间 < 30s

## 6. 最佳实践

1. **幂等性**: Hooks 应该是幂等的，多次执行结果一致
2. **快速失败**: 尽早失败，避免浪费时间
3. **清晰日志**: 使用 stderr 输出调试信息
4. **资源清理**: Hooks 应清理临时文件
5. **超时控制**: 始终设置合理的超时

## 7. Hook 模板库

推荐 Hooks 目录结构：

```
.claude/
├── hooks/
│   ├── setup/
│   │   ├── verify-env.sh
│   │   └── install-deps.sh
│   ├── user-prompt/
│   │   ├── validate-input.py
│   │   └── enrich-context.sh
│   ├── pre-commit/
│   │   ├── format-code.sh
│   │   └── run-linters.sh
│   ├── post-commit/
│   │   └── notify.sh
│   └── tool/
│       ├── audit-edit.sh
│       └── log-mcp.sh
└── hooks.yaml
```
