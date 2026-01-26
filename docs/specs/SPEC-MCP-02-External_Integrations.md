# SPEC-MCP-02: External MCP Integrations

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24
**Covers**: PRD 9.1 (Issue Tracker/Observability MCP)

## 1. 概述 (Overview)

外部 MCP Servers 扩展 Claude Code 的上下文能力，使其能够访问项目管理系统、监控系统、文档系统等外部数据源。本 Spec 定义常用外部 MCP 的集成方案。

## 2. 核心职责 (Core Responsibilities)

- **配置管理**: 定义外部 MCP 的配置格式
- **生命周期管理**: MCP Server 的启动、监控、停止
- **Tool 映射**: 将 MCP Tool 暴露给 Claude
- **错误隔离**: 外部 MCP 崩溃不影响主流程

## 3. 详细设计 (Detailed Design)

### 3.1 配置格式 (Configuration Schema)

在 `.cicd-ai-toolkit.yaml` 中定义外部 MCP：

```yaml
# External MCP Servers
mcp_servers:
  # Linear (Issue Tracker)
  linear:
    enabled: true
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-linear"]
    env:
      LINEAR_API_KEY: "${LINEAR_API_KEY}"
    timeout: 30s
    retry: 3

  # Jira (Issue Tracker)
  jira:
    enabled: false
    command: "mcp-server-jira"
    args: ["--url", "${JIRA_URL}", "--token", "${JIRA_TOKEN}"]
    timeout: 30s

  # Datadog (Observability)
  datadog:
    enabled: true
    command: "mcp-server-datadog"
    args: ["--site", "datadoghq.com"]
    env:
      DD_API_KEY: "${DD_API_KEY}"
      DD_APP_KEY: "${DD_APP_KEY}"
    timeout: 60s

  # Prometheus (Observability)
  prometheus:
    enabled: true
    command: "mcp-server-prometheus"
    args: ["--url", "${PROMETHEUS_URL}"]
    timeout: 30s

  # Confluence (Documentation)
  confluence:
    enabled: false
    command: "mcp-server-confluence"
    args: ["--url", "${CONFLUENCE_URL}", "--token", "${CONFLUENCE_TOKEN}"]
```

### 3.2 Runner MCP Bridge (Go Implementation)

```go
// MCPServer represents an external MCP server
type MCPServer struct {
    Name     string
    Config   MCPServerConfig
    Cmd      *exec.Cmd
    Stdin    io.WriteCloser
    Stdout   io.ReadCloser
    Stderr   io.ReadCloser
    Timeout  time.Duration
    StartedAt time.Time
}

type MCPServerConfig struct {
    Enabled bool
    Command string
    Args    []string
    Env     map[string]string
    Timeout time.Duration
    Retry   int
}

// MCPServerManager manages lifecycle of external MCP servers
type MCPServerManager struct {
    servers map[string]*MCPServer
    mu      sync.RWMutex
}

func NewMCPServerManager() *MCPServerManager {
    return &MCPServerManager{
        servers: make(map[string]*MCPServer),
    }
}

// Start launches an external MCP server
func (m *MCPServerManager) Start(ctx context.Context, name string, config MCPServerConfig) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if !config.Enabled {
        return nil
    }

    // Expand environment variables
    cmd := exec.CommandContext(ctx, config.Command, config.Args...)
    cmd.Env = os.Environ()
    for k, v := range config.Env {
        cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, os.ExpandEnv(v)))
    }

    // Create pipes for stdio
    stdin, _ := cmd.StdinPipe()
    stdout, _ := cmd.StdoutPipe()
    stderr, _ := cmd.StderrPipe()

    server := &MCPServer{
        Name:      name,
        Config:    config,
        Cmd:       cmd,
        Stdin:     stdin,
        Stdout:    stdout,
        Stderr:    stderr,
        Timeout:   config.Timeout,
        StartedAt: time.Now(),
    }

    // Start the process
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("failed to start MCP server %s: %w", name, err)
    }

    // Wait for initialization
    time.Sleep(100 * time.Millisecond)

    // Verify server is responsive
    if err := m.ping(ctx, server); err != nil {
        server.Stop()
        return fmt.Errorf("MCP server %s not responsive: %w", name, err)
    }

    m.servers[name] = server
    log.WithField("mcp", name).Info("External MCP server started")
    return nil
}

func (m *MCPServerManager) ping(ctx context.Context, server *MCPServer) error {
    // Send initialize request
    req := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      1,
        "method":  "initialize",
        "params": map[string]interface{}{
            "protocolVersion": "2024-11-05",
            "capabilities": map[string]interface{}{
                "roots": map[string]bool{
                    "listChanged": false,
                },
            },
            "clientInfo": map[string]string{
                "name":    "cicd-ai-toolkit",
                "version": "1.0.0",
            },
        },
    }

    if err := json.NewEncoder(server.Stdin).Encode(req); err != nil {
        return err
    }

    // Wait for response with timeout
    var resp map[string]interface{}
    decoder := json.NewDecoder(server.Stdout)
    done := make(chan error)

    go func() {
        done <- decoder.Decode(&resp)
    }()

    select {
    case <-time.After(5 * time.Second):
        return fmt.Errorf("ping timeout")
    case err := <-done:
        return err
    }
}

// Stop shuts down all MCP servers
func (m *MCPServerManager) Stop() {
    m.mu.Lock()
    defer m.mu.Unlock()

    for name, server := range m.servers {
        log.WithField("mcp", name).Info("Stopping MCP server")
        server.Stop()
    }
    m.servers = make(map[string]*MCPServer)
}

// Stop stops a single MCP server
func (s *MCPServer) Stop() {
    if s.Cmd != nil && s.Cmd.Process != nil {
        s.Cmd.Process.Signal(syscall.SIGTERM)
        time.Sleep(100 * time.Millisecond)
        if !s.Cmd.ProcessState.Exited() {
            s.Cmd.Process.Kill()
        }
    }
}
```

### 3.3 Tool Proxy (Tool Exposure)

将外部 MCP 的 Tools 暴露给 Claude：

```go
// ToolProxy aggregates tools from all MCP servers
type ToolProxy struct {
    manager *MCPServerManager
    tools   map[string]MCPTool
}

type MCPTool struct {
    Name        string
    Description string
    Server      string
    InputSchema map[string]interface{}
}

// ListTools returns all available tools from all MCP servers
func (p *ToolProxy) ListTools(ctx context.Context) ([]MCPTool, error) {
    var tools []MCPTool

    for name, server := range p.manager.servers {
        req := map[string]interface{}{
            "jsonrpc": "2.0",
            "id":      2,
            "method":  "tools/list",
        }

        if err := json.NewEncoder(server.Stdin).Encode(req); err != nil {
            log.WithError(err).WithField("mcp", name).Warn("Failed to list tools")
            continue
        }

        var resp struct {
            Result struct {
                Tools []struct {
                    Name        string                 `json:"name"`
                    Description string                 `json:"description"`
                    InputSchema map[string]interface{} `json:"inputSchema"`
                } `json:"tools"`
            } `json:"result"`
        }

        if err := json.NewDecoder(server.Stdout).Decode(&resp); err != nil {
            continue
        }

        for _, t := range resp.Result.Tools {
            tools = append(tools, MCPTool{
                Name:        fmt.Sprintf("%s__%s", name, t.Name),
                Description: t.Description,
                Server:      name,
                InputSchema: t.InputSchema,
            })
        }
    }

    return tools, nil
}

// CallTool invokes a tool on an MCP server
func (p *ToolProxy) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
    // Parse server and tool name
    parts := strings.SplitN(toolName, "__", 2)
    if len(parts) != 2 {
        return nil, fmt.Errorf("invalid tool name format: %s", toolName)
    }

    serverName, localName := parts[0], parts[1]

    server, ok := p.manager.servers[serverName]
    if !ok {
        return nil, fmt.Errorf("MCP server not found: %s", serverName)
    }

    req := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      3,
        "method":  "tools/call",
        "params": map[string]interface{}{
            "name":      localName,
            "arguments": args,
        },
    }

    if err := json.NewEncoder(server.Stdin).Encode(req); err != nil {
        return nil, err
    }

    var resp map[string]interface{}
    if err := json.NewDecoder(server.Stdout).Decode(&resp); err != nil {
        return nil, err
    }

    if result, ok := resp["result"]; ok {
        return result, nil
    }

    if errObj, ok := resp["error"]; ok {
        return nil, fmt.Errorf("MCP error: %v", errObj)
    }

    return nil, fmt.Errorf("invalid MCP response")
}
```

### 3.4 线性集成 (Linear Integration)

Linear MCP 提供的能力：

| Tool | 说明 | 用例 |
|------|------|------|
| `linear_search_issues` | 搜索 Issue | 查找相关需求 |
| `linear_get_issue` | 获取 Issue 详情 | 理解验收标准 |
| `linear_list_projects` | 列出项目 | 关联项目 |
| `linear_create_issue` | 创建 Issue | 自动化任务创建 |

**使用示例**：

```yaml
# .cicd-ai-toolkit.yaml
mcp_servers:
  linear:
    enabled: true
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-linear"]
    env:
      LINEAR_API_KEY: "${LINEAR_API_KEY}"
```

```yaml
# skills/requirements-validator/SKILL.md
---
name: "requirements-validator"
description: "Validate code changes against Linear requirements"
mcp_servers:
  - linear
---

# Requirements Validation

You are validating that code changes implement the requirements correctly.

## Instructions

1. Get the associated Linear issue from the PR description (e.g., "Closes ABC-123")
2. Use `linear_get_issue` to fetch the requirements
3. Review the code changes
4. Report if the implementation matches the requirements

## Required Tools

- `linear_get_issue`: Fetch issue details
- `linear_search_issues`: Search for related issues
```

### 3.5 Jira 集成 (Jira Integration)

Jira MCP 提供的能力：

| Tool | 说明 | 用例 |
|------|------|------|
| `jira_get_issue` | 获取 Issue | 需求追溯 |
| `jira_search` | 搜索 Issue | 影响分析 |
| `jira_add_comment` | 添加评论 | 状态同步 |
| `jira_transition` | 状态流转 | 自动化 |

**配置示例**：

```yaml
mcp_servers:
  jira:
    enabled: true
    command: "mcp-server-jira"
    args: [
      "--url", "${JIRA_URL}",
      "--email", "${JIRA_EMAIL}",
      "--token", "${JIRA_API_TOKEN}"
    ]
    timeout: 30s
```

### 3.6 Datadog 集成 (Observability)

Datadog MCP 提供的能力：

| Tool | 说明 | 用例 |
|------|------|------|
| `datadog_query_metrics` | 查询指标 | 性能基准对比 |
| `datadog_get_logs` | 获取日志 | 错误分析 |
| `datadog_list_monitors` | 列出监控 | 告警关联 |

**配置示例**：

```yaml
mcp_servers:
  datadog:
    enabled: true
    command: "mcp-server-datadog"
    args: ["--site", "datadoghq.com"]
    env:
      DD_API_KEY: "${DD_API_KEY}"
      DD_APP_KEY: "${DD_APP_KEY}"
```

**使用场景** (性能审核)：

```yaml
# skills/perf-auditor/SKILL.md
---
name: "perf-auditor"
mcp_servers:
  - datadog
---

# Performance Auditor

You are analyzing code changes for potential performance issues.

## Baseline Comparison

1. Use `datadog_query_metrics` to get current baseline:
   - `system.cpu.usage{service:my-service}`
   - `trace.latency{service:my-service,env:prod}`

2. Identify if changes affect hot paths (high CPU/latency areas)

3. Report potential performance regressions
```

### 3.7 Prometheus 集成

Prometheus MCP 提供的能力：

| Tool | 说明 | 用例 |
|------|------|------|
| `prometheus_query` | PromQL 查询 | 指标查询 |
| `prometheus_query_range` | 范围查询 | 趋势分析 |
| `prometheus_series` | 列出序列 | 元数据查询 |

**配置示例**：

```yaml
mcp_servers:
  prometheus:
    enabled: true
    command: "mcp-server-prometheus"
    args: ["--url", "${PROMETHEUS_URL}", "--token", "${PROMETHEUS_TOKEN}"]
```

### 3.8 Confluence 集成 (Documentation)

Confluence MCP 提供的能力：

| Tool | 说明 | 用例 |
|------|------|------|
| `confluence_get_page` | 获取页面 | 文档验证 |
| `confluence_search` | 搜索 | 相关文档 |
| `confluence_update_page` | 更新页面 | 自动化文档 |

**配置示例**：

```yaml
mcp_servers:
  confluence:
    enabled: true
    command: "mcp-server-confluence"
    args: ["--url", "${CONFLUENCE_URL}"]
    env:
      CONFLUENCE_TOKEN: "${CONFLUENCE_TOKEN}"
```

### 3.9 错误处理 (Error Handling)

```go
// SafeMCP wraps MCP calls with error handling
type SafeMCP struct {
    proxy *ToolProxy
}

func (s *SafeMCP) CallTool(ctx context.Context, tool string, args map[string]interface{}) (interface{}, error) {
    // With timeout
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    result, err := s.proxy.CallTool(ctx, tool, args)
    if err != nil {
        log.WithError(err).WithField("tool", tool).Warn("MCP tool call failed")
        // Return safe default instead of failing
        return map[string]interface{}{
            "error":   "tool_unavailable",
            "message": err.Error(),
        }, nil
    }

    return result, nil
}
```

### 3.10 MCP Server 健康检查

```go
func (m *MCPServerManager) HealthCheck() map[string]string {
    m.mu.RLock()
    defer m.mu.RUnlock()

    status := make(map[string]string)

    for name, server := range m.servers {
        if server.Cmd.Process == nil {
            status[name] = "not_started"
            continue
        }

        if server.Cmd.ProcessState == nil {
            status[name] = "running"
            status[name+"_uptime"] = time.Since(server.StartedAt).String()
        } else if server.Cmd.ProcessState.Exited() {
            status[name] = fmt.Sprintf("exited_%d", server.Cmd.ProcessState.ExitCode())
        } else {
            status[name] = "unknown"
        }
    }

    return status
}
```

## 4. 依赖关系 (Dependencies)

- **Depends on**: [SPEC-MCP-01](./SPEC-MCP-01-Dual_Layer_Architecture.md) - MCP 架构基础
- **Related**: [SPEC-SKILL-01](./SPEC-SKILL-01-Skill_Definition.md) - Skill 中声明 MCP 依赖

## 5. 验收标准 (Acceptance Criteria)

1. **配置解析**: 能正确解析 MCP 服务器配置
2. **进程管理**: 能启动和停止外部 MCP 进程
3. **工具发现**: 能列出外部 MCP 提供的所有工具
4. **工具调用**: 能成功调用外部 MCP 工具并返回结果
5. **错误隔离**: 外部 MCP 崩溃不影响 Runner 主流程
6. **超时控制**: 工具调用超时后能正确返回
7. **资源清理**: Runner 退出时能清理所有 MCP 进程

## 6. 已知外部 MCP Servers

| MCP Server | URL | 维护状态 |
|------------|-----|----------|
| Linear MCP | https://github.com/modelcontextprotocol/servers | Official |
| GitHub MCP | https://github.com/modelcontextprotocol/servers | Official |
| Google Drive MCP | https://github.com/modelcontextprotocol/servers | Official |
| Postgres MCP | https://github.com/modelcontextprotocol/servers | Official |
| Puppeteer MCP | https://github.com/modelcontextprotocol/servers | Official |
| Datadog MCP (社区) | - | Community |
| Prometheus MCP (社区) | - | Community |
