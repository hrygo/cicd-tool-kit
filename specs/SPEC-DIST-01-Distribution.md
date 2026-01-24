# SPEC-DIST-01: Distribution & Installation

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24

## 1. 概述 (Overview)
本 Spec 定义了 `cicd-ai-toolkit` 如何打包、发布并交付给最终用户。目标是支持多种运行环境：Local CLI, CI Containers, 和 Serverless Workers。

## 2. 核心职责 (Core Responsibilities)
- **Artifact Build**: 构建 Go 二进制和多架构 Docker 镜像。
- **Release Channel**: 管理 Stable, Beta, Nightly 版本。
- **Installation**: 提供简便的安装脚本。

## 3. 详细设计 (Detailed Design)

### 3.1 交付物 (Artifacts)
1.  **Binary**:
    *   `cicd-runner-linux-amd64`
    *   `cicd-runner-linux-arm64`
    *   `cicd-runner-darwin-amd64` (Local Dev)
    *   `cicd-runner-darwin-arm64` (Apple Silicon)
2.  **Container Image**:
    *   `ghcr.io/cicd-ai-toolkit/runner:{version}`
    *   包含：Runner Binary, Minimal RootFS, Pre-installed Skills (Generic)。

### 3.2 Docker 镜像规范 (Container Spec)
基于 `gcr.io/distroless/static` 或 `alpine` (为了 Debug)，但在生产环境强制采用 Non-Root。

```dockerfile
FROM alpine:3.19 AS base
RUN apk add --no-cache git ca-certificates

FROM scratch
COPY --from=base /etc/ssl/certs /etc/ssl/certs
COPY cicd-runner /bin/cicd-runner
COPY skills/ /opt/cicd-ai/skills/

USER 1000:1000
ENTRYPOINT ["/bin/cicd-runner"]
```

### 3.3 安装脚本 (One-Click Install)
提供 `install.sh` 用于在 Gitee/GitLab 的私有 Runner 上快速引导。

```bash
curl -fsSL https://get.cicd-toolkit.com | bash -s -- --version v1.0.0
```
该脚本自动检测 OS/Arch，下载对应二进制，并验证 Checksum。

### 3.4 版本策略 (Versioning)
遵循 Semantic Versioning (SemVer)。
*   `v1.0.0`: Stable Release.
*   `v1.1.0-beta.1`: Pre-release.

## 4. 依赖关系 (Dependencies)
- **CI**: 需要 GitHub Actions Workflow 自动触发构建和发布。
- **Registry**: GitHub Container Registry (GHCR).

## 5. 验收标准 (Acceptance Criteria)
1.  **Multi-Arch**: 在 ARM64 机器上 `docker run` 镜像能正常启动。
2.  **Size**: 最终 Docker 镜像大小应控制在 50MB 以内 (Go Binary + Base)。
3.  **Checksum**: 下载脚本必须验证 sha256sum，防止篡改。
