#!/bin/bash
# gen-task.sh - 辅助脚本：调用 AI 生成 TASK.md
# 脚本只负责准备材料，AI 负责创造性工作

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

usage() {
    cat <<'EOF'
用法: gen-task <spec_id> <developer_id>

示例:
  gen-task CORE-01 dev-a
  gen-task LIB-01 dev-c

脚本工作：
  1. 读取 Spec 文档
  2. 读取 state.json 获取开发者信息
  3. 调用 AI 生成 TASK.md
  4. 写入 worktree/TASK.md

AI 负责：
  - 理解 Spec 内容
  - 提取关键信息
  - 组织易读格式
  - 添加合理的验收标准
EOF
    exit 1
}

[[ $# -ne 2 ]] && usage

SPEC_ID="$1"
DEV_ID="$2"

# 验证输入
[[ ! "$SPEC_ID" =~ ^[A-Z]+-[0-9]+$ ]] && { echo "❌ Spec ID 格式错误"; exit 1; }
[[ ! "$DEV_ID" =~ ^dev-[a-z]$ ]] && { echo "❌ Developer ID 格式错误"; exit 1; }

# 查找 Spec 文件
SPEC_FILE=$(find "$PROJECT_ROOT/specs" -name "SPEC-${SPEC_ID}-*.md" 2>/dev/null | head -1)
[[ -z "$SPEC_FILE" ]] && { echo "❌ Spec 文件不存在: SPEC-${SPEC_ID}-*.md"; exit 1; }

# 读取 state.json 获取开发者信息
STATE_FILE="$PROJECT_ROOT/.pm/state.json"
[[ ! -f "$STATE_FILE" ]] && { echo "❌ state.json 不存在"; exit 1; }

DEV_INFO=$(jq -r ".developers[\"$DEV_ID\"]" "$STATE_FILE")
WORKTREE=$(echo "$DEV_INFO" | jq -r '.worktree')
[[ "$WORKTREE" == "null" ]] && { echo "❌ 开发者 worktree 未设置"; exit 1; }

# 准备上下文（脚本做搬运工）
SPEC_CONTENT=$(cat "$SPEC_FILE")
DEV_NAME=$(echo "$DEV_INFO" | jq -r '.name')
NAMESPACE=$(echo "$DEV_INFO" | jq -r '.namespace | join(", ")')

# 构建 Prompt
PROMPT=$(cat <<'PROMPT_END'
你是一个技术项目负责人。请根据以下信息生成一个 TASK.md 任务卡片。

## 要求
1. 简洁明了，突出重点
2. 包含：任务描述、核心职责、交付物、验收标准
3. 从 Spec 中提取关键信息，不要复制粘贴
4. 验收标准要具体可测试

## 上下文

### Spec 文档内容
<<<SPEC>>>

### 开发者信息
- 开发者: <<<DEV>>> (<<<DEV_NAME>>>)
- 命名空间: <<<NAMESPACE>>>
- Worktree: <<<WORKTREE>>>

### 输出要求
直接输出 TASK.md 内容（不需要代码块），格式为 Markdown。

PROMPT_END
)

# 替换变量
PROMPT="${PROMPT//<<<SPEC>>>/$SPEC_CONTENT}"
PROMPT="${PROMPT//<<<DEV>>>/$DEV_ID}"
PROMPT="${PROMPT//<<<DEV_NAME>>>/$DEV_NAME}"
PROMPT="${PROMPT//<<<NAMESPACE>>>/$NAMESPACE}"
PROMPT="${PROMPT//<<<WORKTREE>>>/$WORKTREE}"

# 调用 AI 生成（AI 做创造性工作）
echo "🤖 正在调用 AI 生成 TASK.md..."

OUTPUT_FILE="$WORKTREE/TASK.md"

# 使用 claude CLI 生成
if command -v claude &>/dev/null; then
    echo "$PROMPT" | claude -p > "$OUTPUT_FILE" 2>/dev/null || {
        echo "❌ AI 生成失败，请检查 claude CLI"
        exit 1
    }
else
    # 降级：使用简单模板
    cat > "$OUTPUT_FILE" <<EOF
# $DEV_ID 任务卡片

**开发者**: $DEV_ID ($DEV_NAME)
**命名空间**: $NAMESPACE

## 当前任务

### $SPEC_ID - $(grep "^#.*:" "$SPEC_FILE" | head -1 | sed 's/^#* //')

详细内容请参考: \`../../specs/$(basename "$SPEC_FILE")\`

---
生成时间: $(date +%Y-%m-%d)
EOF
fi

echo "✅ $OUTPUT_FILE"
