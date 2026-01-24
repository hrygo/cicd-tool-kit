# 当前任务

## 任务: SKILL-01 - Skill Definition & Standard Schema

**优先级**: P0
**Phase**: Phase 1
**预计工作量**: 1 人周
**分配日期**: 2026-01-24

### 依赖检查
- 无前置依赖，可立即开始

### 任务描述

定义 `SKILL.md` 标准格式，包含元数据、Prompt、工具权限、资源需求和输入输出契约。实现 Skill 加载器和验证器。

### 核心交付物

1. **SKILL.md Schema**
   - YAML Frontmatter: metadata, options, tools, inputs
   - Markdown Body: System Prompt, Task Instruction, Output Contract

2. **加载逻辑**
   - Discovery: 扫描 `skills/` 目录
   - Parsing: 使用 `yaml` 库解析 Head，读取 Body
   - Validation: 检查 `name`, `inputs` 是否完整
   - Injection: 将 Body 部分拼接到 Claude 的 System Prompt 中

3. **标准内置技能**
   - `code-reviewer`: 通用代码审查
   - `test-generator`: 单元测试生成（需 `edit` 权限）
   - `committer`: 生成 Commit Message

### 验收标准

- [ ] Format Parsing: 能够正确读取 Frontmatter 中的配置（如 `temperature`）
- [ ] Validation Error: 如果缺少 `name` 字段，加载器抛出错误
- [ ] Prompt Assembly: 验证最终发送给 Claude 的 Prompt 确实包含了 Markdown Body 的内容
- [ ] 单元测试覆盖率 > 80%

### 相关文件

- Spec 文档: `specs/SPEC-SKILL-01-Skill_Definition.md`
- 实施计划: `specs/IMPLEMENTATION_PLAN.md`
- 进展跟踪: `specs/PROGRESS.md`

### 备注

- 这是 Phase 1 的关键任务，被 10 个其他 Spec 依赖
- 完成后将解锁: CORE-01, CORE-03, LIB-01, LIB-02, LIB-03, LIB-04, PLAT-05, ECO-01, MCP-02, RFC-01
- 需要与 dev-a 协调 CORE-01 的接口设计

---

## 队列任务

| Spec ID | Spec 名称 | Phase | 优先级 | 阻塞原因 |
|---------|-----------|-------|--------|----------|
| LIB-01 | Standard Skills | 5 | P0 | 等待 SKILL-01 完成 |
| MCP-01 | Dual Layer Architecture | 7 | P1 | 无阻塞，可并行 |
