# SPEC-SKILL-01: Skill Definition & Standard Schema

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24

## 1. 概述 (Overview)
`SKILL.md` 是定义 Agent 能力的标准格式。它不仅包含 Prompt，还包含了工具权限、资源需求和输入输出契约。标准化的格式使得 Skills 可以在社区间共享。

## 2. 核心职责 (Core Responsibilities)
- **Metadata**: 描述技能名称、版本、作者。
- **Prompt Engineering**: 定义 System Prompt 和 Task Prompt。
- **Input/Output Schema**: 强类型定义。

## 3. 详细设计 (Detailed Design)

### 3.1 文件结构 (Schema)
`SKILL.md` 由 YAML Frontmatter 和 Markdown Body 组成。

```markdown
---
name: "code-reviewer"
version: "1.0.0"
description: "Expert level code review with security focus"
author: "cicd-ai-team"
license: "MIT"

# Runtime Config
options:
  temperature: 0.2
  budget_tokens: 4096   # For "Thinking" block
  
# Tool Permissions (Override Global Defaults)
tools:
  allow: ["grep", "ls", "cat"]
  
# Input Args
inputs:
  - name: diff
    type: string
    description: "The git diff to analyze"
---

# System Role
You are a Principal Software Engineer. Your goal is to find bugs...

# Task Instruction
Analyze the code provided in the `<<<DIFF_CONTEXT>>>` block.

# Output Contract
You must output in XML-wrapped JSON:
<json>
{
  "issues": [...]
}
</json>
```

### 3.2 加载逻辑 (Loading Logic)
1.  **Discovery**: Runner 扫描 `skills/` 目录。
2.  **Parsing**: 使用 `yaml` 库解析 Head，读取 Body。
3.  **Validation**: 检查 `name`, `inputs` 是否完整。
4.  **Injection**: 将 Body 部分拼接到 Claude 的 System Prompt 中。

### 3.3 标准内置技能 (Standard Built-in Skills)
*   `code-reviewer`: 通用代码审查。
*   `test-generator`: 单元测试生成（需 `edit` 权限）。
*   `committer`: 生成 Commit Message。

## 4. 依赖关系 (Dependencies)
- **Used by**: [SPEC-CORE-01](./SPEC-CORE-01-Runner_Lifecycle.md) 初始化 Session 时加载。

## 5. 验收标准 (Acceptance Criteria)
1.  **Format Parsing**: 能够正确读取 Frontmatter 中的配置（如 `temperature`）。
2.  **Validation Error**: 如果缺少 `name` 字段，加载器抛出错误。
3.  **Prompt Assembly**: 验证最终发送给 Claude 的 Prompt 确实包含了 Markdown Body 的内容。
