# dev-b 任务卡片

**开发者**: dev-b (Security & Infra Engineer)
**技术栈**: Go, YAML, GitHub Actions
**命名空间**: `pkg/security/`, `pkg/governance/`, `pkg/observability/`

---

## 当前任务

### 任务: PLAT-05 - GitHub Composite Actions

- **状态**: 🚧 In Progress
- **优先级**: P1
- **Phase**: Phase 3
- **依赖**: DIST-01 ✅
- **预估**: 1 人周

### 任务描述

将 `cicd-ai-toolkit` 的不同能力封装为可复用的 GitHub Composite Actions，使用户可以灵活组合 AI 能力到 CI/CD 流水线中。

### 核心职责

1. **Modular Design**: 将不同 Skill 封装为独立可复用的 Action
2. **Composition**: 支持组合多个 Actions
3. **Versioning**: 语义化版本管理
4. **Discovery**: 用户能方便地发现和了解可用 Actions

### 交付物

| Action | 描述 | 状态 |
|--------|------|------|
| **setup** | 基础环境安装 cicd-ai-toolkit | ⏳ |
| **review** | 执行代码审查 | ⏳ |
| **test-gen** | 测试生成 | ⏳ |
| **analyze** | 变更分析 | ⏳ |
| **security-scan** | 安全扫描 | ⏳ |
| **all** | 全功能组合 | ⏳ |

### Action 层级结构

```
actions/
├── setup/action.yml      # 基础环境安装
├── review/action.yml     # 代码审查
├── test-gen/action.yml   # 测试生成
├── analyze/action.yml    # 变更分析
├── security-scan/action.yml  # 安全扫描
└── all/action.yml        # 全功能组合
```

### 设计示例

#### Setup Action
```yaml
name: 'Setup cicd-ai-toolkit'
description: 'Install and configure cicd-ai-toolkit for AI-powered CI/CD'

inputs:
  version:
    description: 'Version to install'
    default: 'latest'
  claude-version:
    description: 'Claude Code version to use'
    default: 'latest'

outputs:
  runner-path:
    description: 'Path to the cicd-runner binary'
```

#### Review Action
```yaml
name: 'AI Code Review'
description: 'Perform AI-powered code review using Claude'

inputs:
  skills:
    default: 'code-reviewer,change-analyzer'
  severity-threshold:
    default: 'warning'
  fail-on-error:
    default: 'false'
  post-comment:
    default: 'true'

outputs:
  issues-found:
    description: 'Number of issues found'
  critical-count:
    description: 'Number of critical issues'
```

### 验收标准

- [ ] setup action 能正确安装 cicd-ai-toolkit
- [ ] review action 能执行代码审查并输出结果
- [ ] test-gen action 能生成测试代码
- [ ] analyze action 能生成变更摘要
- [ ] all action 能组合运行所有技能
- [ ] 支持 GitHub Actions Marketplace 发布
- [ ] 每个 action 有完整的文档和示例

### 相关文件

- Spec 文档: `../../specs/SPEC-PLAT-05-Composite_Actions.md`
- 依赖 Spec: `../../specs/SPEC-DIST-01-Distribution.md`

---

## 已完成任务

| Spec ID | 名称 | 完成日期 | PR |
|---------|------|----------|-----|
| DIST-01 | Distribution | 2026-01-25 | - |

---

## 工作区信息

- **当前 Worktree**: `/Users/huangzhonghui/.worktree/pr-b-PLAT-05`
- **当前分支**: `pr-b-PLAT-05`
- **锁定文件**: `governance`

---

## 开发命令

```bash
# 创建 action 目录结构
mkdir -p actions/{setup,review,test-gen,analyze,security-scan,all}

# 验证 action.yml 语法
# 使用 GitHub Actions act 工具本地测试
act -l

# 运行测试
make test
```

---

## 进度日志

| 日期 | 操作 | 状态 |
|------|------|------|
| 2026-01-25 | 分配 PLAT-05 任务 | ✅ |
