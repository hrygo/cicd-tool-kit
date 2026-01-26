# SPEC-CORE-02: Context Injection & Smart Chunking

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24

## 1. 概述 (Overview)
由于 LLM 的 Context Window 限制（即便是 200k 也有性能和成本考量），Runner 必须智能地处理大型 Pull Request。本 Spec 定义了如何获取代码变更、清理无效上下文以及进行智能分片（Smart Chunking）。

## 2. 核心职责 (Core Responsibilities)
- **Diff 获取**: 从 Git 获取 Base 与 Head 之间的增量。
- **Context Pruning**: 移除无关文件（Lockfiles, Binary, Vendor）。
- **Token Estimation**: 估算 Token 消耗。
- **Smart Chunking**: 将大 Diff 切割为适合 LLM 处理的 Chunks。

## 3. 详细设计 (Detailed Design)

### 3.1 上下文来源 (Context Sources)
Runner 需聚合以下信息写入 Claude 的 Stdin：
1.  **Project Context**: `CLAUDE.md` 内容 (Priority: High).
2.  **Skill Instruction**: `SKILL.md` Prompt (Priority: Critical).
3.  **Diff Content**: 实际代码变更 (Priority: High).
4.  **Linter Output**: 可选，静态分析结果 (Priority: Medium).

### 3.2 剪枝算法 (Pruning Algorithm)
在获取 `git diff --name-only` 后，应用以下过滤器：
*   **Ignore List**:
    *   Glob Patterns: `*.lock`, `go.sum`, `yarn.lock`.
    *   Directories: `vendor/`, `node_modules/`, `dist/`, `build/`, `.idea/`.
    *   Extensions: `.png`, `.jpg`, `.exe`, `.so`, `.dll`.
*   **Logic**: 仅保留文本文件且大小 < 1MB 的文件。

### 3.3 分片策略 (Smart Chunking Strategy)

#### 参数定义
*   `MAX_CHUNK_TOKENS`: 默认为 32,000 (保守值，留给 Output 和 Skill Prompt)。
*   `TOKEN_RATIO`: 4 chars = 1 token (估算)。

#### Bin Packing Algorithm (装箱算法)
如果总 Token > `MAX_CHUNK_TOKENS`:

1.  **Sort**: 将文件按 Token 大小降序排列。
2.  **Allocate**:
    ```go
    chunks := [][]FileDiff{}
    currentChunk := []FileDiff{}
    currentSize := 0

    for _, file := range files {
        if file.Tokens > MAX_CHUNK_TOKENS {
             // Handle Huge File: Split by logical blocks (Functions)
             // MVP: Truncate with warning header
             chunks = append(chunks, SplitHugeFile(file)...)
             continue
        }
        
        if currentSize + file.Tokens > MAX_CHUNK_TOKENS {
            chunks = append(chunks, currentChunk)
            currentChunk = []FileDiff{}
            currentSize = 0
        }
        
        currentChunk = append(currentChunk, file)
        currentSize += file.Tokens
    }
    ```

### 3.4 注入格式 (Injection Format)
每个 Chunk 发送给 Claude 时，需包装：
```markdown
# Context Chunk 1/3

## Project Rules
(Content of CLAUDE.md)

## Instructions
(Content of SKILL.md)

## Code Changes
(File A Diff)
(File B Diff)
```

## 4. 依赖关系 (Dependencies)
- **Depends on**: `git` 二进制。
- **Used by**: [SPEC-CORE-01](./SPEC-CORE-01-Runner_Lifecycle.md) 将生成的 Chunk 写入 Stdin。

## 5. 验收标准 (Acceptance Criteria)
1.  **Pruning**: 提交包含 `main.go` (10行) 和 `pnpm-lock.yaml` (5000行) 的 PR。Runner 应仅将 `main.go` 传给 Claude。
2.  **Single Chunk**: PR Diff Token < 32k，生成 1 个 API 请求。
3.  **Multi Chunk**: PR Diff Token = 50k，Limit = 32k。应生成 2 个 Chunks。每个 Chunk 内部文件完整，且都包含 `SKILL.md` 指令。
4.  **Reproducibility**: 同样的 Diff 输入，必须生成相同的 Chunks 序列（确定性排序）。
