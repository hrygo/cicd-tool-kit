# 当前任务

## 任务: DIST-01 - Distribution & Installation

**优先级**: P0
**Phase**: Phase 0
**预计工作量**: 0.5 人周
**分配日期**: 2026-01-24

### 依赖检查
- 无前置依赖，可立即开始

### 任务描述

定义 `cicd-ai-toolkit` 的打包、发布和交付机制，支持多种运行环境：Local CLI, CI Containers, 和 Serverless Workers。

### 核心交付物

1. **多架构构建**
   - `cicd-runner-linux-amd64`
   - `cicd-runner-linux-arm64`
   - `cicd-runner-darwin-amd64` (Local Dev)
   - `cicd-runner-darwin-arm64` (Apple Silicon)

2. **容器镜像**
   - 基于 `alpine` 或 `distroless/static`
   - 镜像大小控制在 50MB 以内
   - Non-Root 用户运行
   - `ghcr.io/cicd-ai-toolkit/runner:{version}`

3. **安装脚本**
   - `install.sh` 一键安装脚本
   - 自动检测 OS/Arch
   - SHA256 校验和验证
   - `curl -fsSL https://get.cicd-toolkit.com | bash`

4. **版本策略**
   - 遵循 Semantic Versioning (SemVer)
   - 支持 Stable, Beta, Nightly 渠道

### 验收标准

- [ ] Multi-Arch: 在 ARM64 机器上 `docker run` 镜像能正常启动
- [ ] Size: 最终 Docker 镜像大小应控制在 50MB 以内
- [ ] Checksum: 下载脚本必须验证 sha256sum，防止篡改
- [ ] GitHub Actions Workflow 自动触发构建和发布
- [ ] 安装脚本测试通过

### 相关文件

- Spec 文档: `specs/SPEC-DIST-01-Distribution.md`
- 实施计划: `specs/IMPLEMENTATION_PLAN.md`
- 进展跟踪: `specs/PROGRESS.md`

### 备注

- 这是 Phase 0 的任务，与 PLAT-07 并行开发
- 需要与 dev-a 协调 CI/CD 配置
- 完成后将解锁 PLAT-05 (Composite Actions) 和 LIB-02 (Extended Skills)

---

## 队列任务

| Spec ID | Spec 名称 | Phase | 优先级 | 阻塞原因 |
|---------|-----------|-------|--------|----------|
| SEC-02 | Prompt Injection | 2 | P1 | 等待 CORE-02 |
| PLAT-05 | Composite Actions | 3 | P2 | 等待 DIST-01 完成 |
