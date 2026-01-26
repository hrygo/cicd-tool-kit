# SPEC-GOV-01: Policy-as-Code Governance (OPA)

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24

## 1. 概述 (Overview)
为了防止 AI 在高风险场景下误操作（如自动合并代码、部署生产环境），引入 Open Policy Agent (OPA) 作为决策引擎。所有关键操作（Action）在执行前必须通过 Policy Check。

## 2. 核心职责 (Core Responsibilities)
- **Interceptor**: 拦截 Claude 的 Tool Calls 和 Platform Actions。
- **Evaluation**: 基于 `policy/*.rego` 文件评估请求。
- **Enforcement**: 拒绝违反策略的操作并报警。

## 3. 详细设计 (Detailed Design)

### 3.1 策略定义 (Rego)
策略文件存放在 `.github/policies/` 或 `.cicd-ai-toolkit/policies/`。

```rego
package cicd.authz

default allow = false

# 允许 Review 任意非 main 分支
allow {
    input.action == "review"
    input.target_branch != "main"
}

# 禁止在周五通过 API 合并代码
deny[msg] {
    input.action == "merge"
    time.weekday(time.now_ns()) == "Friday"
    msg := "No merging on Fridays"
}
```

### 3.2 集成点 (Integration Points)
Runner 在以下时刻调用 OPA Eval：
1.  **Before Tool Execution**: 当 Claude 请求执行 `edit_file` 或 `bash` 时。
2.  **Before Platform Action**: 当 Runner 准备调用 `PostComment` 或 `MergePR` 时。

### 3.3 输入结构 (Input Schema)
```json
{
  "action": "edit_file",
  "resource": "src/auth/login.go",
  "user": "claude-bot",
  "context": {
    "branch": "feature/ui-update",
    "time": "2026-01-24T18:00:00Z"
  }
}
```

## 4. 依赖关系 (Dependencies)
- **Lib**: `github.com/open-policy-agent/opa/rego` (Go Library)。

## 5. 验收标准 (Acceptance Criteria)
1.  **Blocking**: 配置 "Deny all edits to `go.mod`"。Agent 尝试修改 `go.mod` 时，Runner 应拦截并返回 "Access Denied by Policy"。
2.  **Permitting**: 符合规则的操作应正常透传。
3.  **Audit**: 被拦截的操作应记录在 Structured Log 中，包含 Policy ID。
