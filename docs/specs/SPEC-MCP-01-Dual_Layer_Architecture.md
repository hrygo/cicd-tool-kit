# SPEC-MCP-01: Dual-Layer MCP Architecture

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24

## 1. 概述 (Overview)
Model Context Protocol (MCP) 是 Claude 获取上下文的标准协议。Runner 采用 "Dual-Layer" 架构：既作为 MCP Server 为 Claude 提供 CI 环境信息，又作为 Host 挂载外部 MCP Client 以扩展能力。

## 2. 核心职责 (Core Responsibilities)
- **Internal Layer**: Runner 内置轻量级 MCP Server，暴露 `get_env`, `get_secrets`。
- **External Layer**: 允许通过配置挂载第三方 MCP Servers (如 Linear, Datadog)。
- **Security**: 确保 Secrets 仅通过受控的 MCP Tool 暴露，而非直接注入 Prompt。

## 3. 详细设计 (Detailed Design)

### 3.1 内部 MCP Server (Runner-Hosted)
Runner 在启动 Claude 子进程时，通过 `--mcp-server` 参数（或 stdin transport）注入自身提供的 MCP 服务。

#### Tools 列表
*   `get_ci_context()`: 返回 `pr_id`, `repo_url`, `author`, `commit_sha`。
*   `get_file_content_secure(path)`: 读取文件内容（受 Sandbox 限制）。
*   `get_deployment_key(env)`: 获取部署所需的临时 Token (仅限 Deploy Skill 使用)。

### 3.2 外部 MCP 挂载 (External MCP Mounts)
用户可在 `.cicd-ai-toolkit.yaml` 中配置外部 MCP：

```yaml
mcp_servers:
  linear:
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-linear"]
    env:
      LINEAR_API_KEY: "${LINEAR_KEY}"
```

Runner 负责：
1.  解析配置。
2.  启动外部 MCP 进程。
3.  将外部 MCP 的 Stdout/Stdin 桥接给 Claude Code (通过 SDK 或配置)。

## 4. 依赖关系 (Dependencies)
- **Deps**: 依赖 `github.com/mark3labs/mcp-go` (假设存在 Go SDK) 实现 Server。

## 5. 验收标准 (Acceptance Criteria)
1.  **Internal Call**: 在 Skill 中调用 `get_ci_context()`，能正确获取当前 GHA 的 Run ID。
2.  **External Integ**: 配置 Linear MCP 后，Claude 能够通过 `linear_get_issue` 工具读取关联的 Issue 描述。
3.  **Isolation**: 外部 MCP崩溃不应导致 Runner 崩溃。
