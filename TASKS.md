# 当前任务

## ✅ 已完成

### 任务: DIST-01 - Distribution & Installation

**优先级**: P0
**Phase**: Phase 0
**预计工作量**: 0.5 人周
**分配日期**: 2026-01-24
**完成日期**: 2026-01-24

### 验收标准

- [x] Multi-Arch: 在 ARM64 机器上 `docker run` 镜像能正常启动
- [x] Size: 最终 Docker 镜像大小应控制在 50MB 以内
- [x] Checksum: 下载脚本必须验证 sha256sum，防止篡改
- [x] GitHub Actions Workflow 自动触发构建和发布
- [x] 安装脚本测试通过

### 交付物

1. **多架构构建** ✅
   - `build/packaging/build-all.sh` 支持 linux/amd64,linux/arm64,darwin/amd64,darwin/arm64

2. **容器镜像** ✅
   - `Dockerfile` - 基于 gcr.io/distroless/static:nonroot
   - `Dockerfile.slim` - 基于 alpine (用于调试)
   - Non-Root 用户 (UID 65532)

3. **安装脚本** ✅
   - `build/packaging/install.sh` - 一键安装脚本
   - 自动检测 OS/Arch
   - SHA256 校验和验证
   - Cosign 签名验证支持

4. **版本策略** ✅
   - `pkg/version/version.go` - ldflags 注入版本信息
   - 遵循 Semantic Versioning (SemVer)

### 已解锁的依赖

- **PLAT-05**: Composite Actions (可开始)
- **LIB-02**: Extended Skills (可开始)

---

## 下一步任务

| Spec ID | Spec 名称 | Phase | 优先级 | 状态 |
|---------|-----------|-------|--------|------|
| PLAT-07 | Project Structure | 0 | P0 | 🔄 进行中 (dev-a) |
| CONF-01 | Configuration | 1 | P0 | ⏳ 可开始 |
| SKILL-01 | Skill Definition | 1 | P0 | ⏳ 可开始 |
| PLAT-05 | Composite Actions | 3 | P2 | ⏳ 可开始 (DIST-01 已完成) |
| LIB-02 | Extended Skills | 5 | P1 | ⏳ 可开始 (DIST-01 已完成) |
