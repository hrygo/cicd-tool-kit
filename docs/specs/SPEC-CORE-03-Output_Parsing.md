# SPEC-CORE-03: Output Parsing & XML Handling

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24

## 1. 概述 (Overview)
Claude Code (特别是 Sonnet 模型) 在分析复杂问题时会产生大量的 "Thinking" 文本。为了确保 Runner 能稳定获取机器可读的结果，我们采用 `<xml>` 标签包裹 JSON 的策略，并实现鲁棒的解析器。

## 2. 核心职责 (Core Responsibilities)
- **Stream Monitoring**: 实时监听 Stdout，防止缓冲区溢出。
- **Block Extraction**: 基于 Regex 提取特定 XML 标签内容。
- **JSON Repair**: 修复常见的 LLM JSON 格式错误。
- **Schema Validation**: 验证业务字段完整性。

## 3. 详细设计 (Detailed Design)

### 3.1 输出格式定义 (Prompt Contract)
在 [SPEC-SKILL-01](./SPEC-SKILL-01-Skill_Definition.md) 中，强制要求 Claude 输出如下结构：

```xml
<thinking>
Here represents the chain of thought...
</thinking>

<json>
{
  "issues": [...]
}
</json>
```

### 3.2 解析算法 (Parsing Algorithm)
1.  **Buffering**: 读取 Stdout 直到 EOF。
2.  **Regex Match**:
    *   Pattern: `(?s)<json>(.*?)</json>`
    *   说明: `(?s)` 开启 Dot-Match-All 模式，支持跨行匹配。
3.  **Candidate Selection**:
    *   如果匹配到多个 `<json>` 块，取**最后一个**（通常是最完善的结论）。
    *   *Edge Case*: 如果因为 Tokens 截断导致没有闭合标签 `</json>`，尝试从最后一个 `<json>` 开始读取到末尾。

### 3.3 容错与修复 (Fault Tolerance)
使用 `json-iterator` 或自定义逻辑处理常见错误：
*   **Trailing Commas**: `[1, 2,]` -> `[1, 2]`
*   **Missing Quotes**: `{key: "val"}` -> `{"key": "val"}`
*   **Markdown Pollution**: 移除 JSON 内部误写的 Markdown 标记 (如 ` ``` `)。

### 3.4 验证 (Validation)
解析出的对象必须符合内部 Go Struct 定义：
```go
type AnalysisResult struct {
    Issues []struct {
        File     string `json:"file" validate:"required"`
        Line     int    `json:"line" validate:"min=1"`
        Severity string `json:"severity" validate:"oneof=critical high medium low"`
        Message  string `json:"message" validate:"required"`
    } `json:"issues"`
}
```

## 4. 依赖关系 (Dependencies)
- **Input**: 来自 [SPEC-CORE-01](./SPEC-CORE-01-Runner_Lifecycle.md) 的 Stdout。
- **Schema**: 依赖 [SPEC-SKILL-01](./SPEC-SKILL-01-Skill_Definition.md) 定义的输出契约。

## 5. 验收标准 (Acceptance Criteria)
1.  **Standard**: 输入 `<thinking>...</thinking><json>{"a":1}</json>`，提取出 `{"a":1}`。
2.  **Dirty Input**: 输入 `Sure, here is the json: <json> { "a": 1, } </json> Hope this helps!`。需成功提取并修复尾部逗号。
3.  **Truncated**: 输入 `<json>{"a":1` (EOF)。如果配置为 `AllowPartial`，尝试闭合；否则报错。
4.  **No Match**: 如果找不到 `<json>` 标签，Runner 应当记录 Warning 并将整个 Raw Output 视为 Comment 发送（降级策略）。
