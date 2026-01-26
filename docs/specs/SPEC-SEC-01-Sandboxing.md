# SPEC-SEC-01: Sandboxing & Container Security

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24

## 1. 概述 (Overview)
由于 `cicd-ai-toolkit` 允许 Agent 具有一定的自主性（执行 Shell、读写文件），如果不加以严格限制，可能导致 Supply Chain 攻击。本 Spec 定义了 "Defense-in-Depth" (纵深防御) 的沙箱策略。

## 2. 核心职责 (Core Responsibilities)
- **Filesystem Isolation**: 限制 Agent 只能写入特定目录。
- **Network Segmentation**: 阻断非必要的外部连接。
- **Privilege Separation**: 防止 Agent 逃逸到宿主。

## 3. 详细设计 (Detailed Design)

### 3.1 容器运行时配置 (Docker Runtime)
在 `Dockerfile` 或启动参数中强制应用：

1.  **Read-Only RootFS**:
    *   `--read-only`: 容器根文件系统不可写。
    *   效果：Agent 即使获取 Shell 也无法修改 `/bin`, `/lib` 或安装 Rootkit。
2.  **Mounts Strategy**:
    *   `/workspace`: **Copy-on-Write** (CoW) 挂载。Agent 可以修改代码，但不会直接污染宿主 Volume（除非任务成功后 Explicit Merge）。
    *   `/tmp`: `tmpfs` 挂载，用于存放临时生成的脚本，容器重启即焚毁。
3.  **User Identity**:
    *   `USER nonroot`: 必须以非 Root 用户运行 Claude。UID/GID 需映射到宿主 CI 用户。

### 3.2 网络防火墙 (Network Firewall)
默认情况 `NetworkMode: none` 是最安全的，但 Agent 需要访问 API 和 Git。

*   **Allowlist**:
    *   `api.anthropic.com` (Claude API)
    *   `github.com` / `gitee.com` (Git Operations)
    *   Internal Artifact Repo (if configured)
*   **Implementation**: 使用 `IPTables` 或 Docker Network Plugin 实现 Egress 过滤。

### 3.3 敏感信息屏蔽 (Secrets Redaction)
*   禁止将 `secrets.*` 直接挂载到 Environment。
*   Runner 内部维护一个 `SecretScrubber`：在 stdout/stderr 流向 Claude 之前，正则匹配常见 Key 格式 (AWS Key, Private Key) 并替换为 `***`。

## 4. 依赖关系 (Dependencies)
- **Depends on**: Docker / OCI Runtime 环境。
- **Used by**: [SPEC-CORE-01](./SPEC-CORE-01-Runner_Lifecycle.md) 启动 Claude 时应用。

## 5. 验收标准 (Acceptance Criteria)
1.  **Write Protection**: 在 Skill 中执行 `echo "pwned" > /bin/ls`，必须报错 `Read-only file system`.
2.  **Network Block**: 在 Skill 中执行 `curl malicious-site.com`，必须超时或连接被拒。
3.  **Privilege Escalation**: 尝试 `sudo apt-get install` 必须失败（无 sudo 权限且 FS 只读）。
