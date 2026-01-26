# dev-c 任务卡片

**开发者**: dev-c (AI & Skills Engineer)
**技术栈**: Go, Markdown, Prompt Engineering
**命名空间**: `pkg/skill/`, `skills/`, `pkg/mcp/`

---

## 当前任务

### 任务: LIB-01 - Standard Skills Library

- **状态**: ✅ Complete
- **优先级**: P1
- **Phase**: Phase 5
- **依赖**: SKILL-01 ✅
- **预估**: 1-2 人周

### 任务描述

实现一组高质量的 Standard Skills，覆盖最常见的 CI 场景。这些 Skill 将作为社区开发 Custom Skill 的参考模板。

### 核心职责

1. **High Quality**: 经过精心调优的 Prompts
2. **Robustness**: 包含详细的边界情况处理指令
3. **Standardization**: 作为社区参考模板

### 交付物

| Skill | 描述 | 状态 |
|-------|------|------|
| **code-reviewer** | 安全、逻辑、性能问题审查 | ✅ |
| **change-analyzer** | PR 摘要和 Release Notes | ✅ |
| **test-generator** | 单元测试生成 | ✅ |
| **log-analyzer** | CI 日志分析，定位根因 | ✅ |
| **issue-triage** | Issue 自动分类和优先级 | ✅ |

### 验收标准

- [x] code-reviewer: 检测注入的 SQL 注入漏洞为 Critical
- [x] change-analyzer: 重构代码摘要指出意图而非机械罗列
- [x] test-generator: 生成的 Go 测试能通过 go test
- [x] log-analyzer: 能从日志中定位根因并给出修复建议
- [x] issue-triage: 正确分类 Bug Issue为 `category: "bug"`
- [x] 所有 Skill 符合 SPEC-SKILL-01 格式
- [x] 每个 SKILL.md 有完整的文档和示例

### 相关文件

- Spec 文档: `../../specs/SPEC-LIB-01-Standard_Skills.md`
- 依赖 Spec: `../../specs/SPEC-SKILL-01-Skill_Definition.md`

---

## 工作区信息

- **当前 Worktree**: `~/.worktree/pr-c-LIB-01`
- **当前分支**: `pr-c-LIB-01`
- **锁定文件**: `skill`

---

## 进度日志

| 日期 | 操作 | 状态 |
|------|------|------|
| 2026-01-25 | 分配 LIB-01 任务 | ✅ |
| 2026-01-26 | 实现 5 个 Standard Skills | ✅ |
| 2026-01-26 | 验证 Skills 加载正常 | ✅ |
