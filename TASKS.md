# 当前任务

## ✅ 已完成: SKILL-01 - Skill Definition & Standard Schema

**完成日期**: 2026-01-25
**优先级**: P0
**Phase**: Phase 1

### 验收标准

- [x] Format Parsing: 能够正确读取 Frontmatter 中的配置（如 `temperature`）
- [x] Validation Error: 如果缺少 `name` 字段，加载器抛出错误
- [x] Prompt Assembly: 验证最终发送给 Claude 的 Prompt 确实包含了 Markdown Body 的内容
- [x] 单元测试覆盖率 > 80% (实际: 86.2%)

### 交付物

1. **SKILL.md Schema** - `pkg/skill/skill.go`
   - YAML Frontmatter: metadata, options, tools, inputs
   - Markdown Body: System Prompt, Task Instruction, Output Contract

2. **加载逻辑** - `pkg/skill/loader.go`
   - Discovery: 扫描 `skills/` 目录
   - Parsing: 使用 `yaml` 库解析 Head，读取 Body
   - Validation: 检查 `name`, `inputs` 是否完整

3. **Prompt 注入器** - `pkg/skill/injector.go`
   - 将 Body 部分拼接到 Claude 的 System Prompt 中
   - 支持占位符替换 (使用 `strings.Replacer` 优化性能)

4. **Skill 注册表** - `pkg/skill/registry.go`
   - 线程安全的 Skill 管理

5. **标准内置技能**
   - `skills/code-reviewer/`: 通用代码审查
   - `skills/test-generator/`: 单元测试生成
   - `skills/committer/`: 生成 Commit Message

### 解锁任务

SKILL-01 完成后解锁以下 Spec：
- CORE-01, CORE-03 (Runner 核心功能)
- LIB-01, LIB-02, LIB-03, LIB-04 (标准技能库)
- PLAT-05 (Composite Actions)
- ECO-01 (Skill Marketplace)
- MCP-02 (External Integrations)
- RFC-01 (RFC Process)

---

## 队列任务

| Spec ID | Spec 名称 | Phase | 优先级 | 状态 |
|---------|-----------|-------|--------|------|
| LIB-01 | Standard Skills | 5 | P0 | 🟢 可开始 (SKILL-01 已完成) |
| MCP-01 | Dual Layer Architecture | 7 | P1 | 🟢 可并行 (无阻塞) |
