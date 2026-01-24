#!/bin/bash
# worktree.sh - Git worktree 操作脚本
# 只处理 Git worktree 的创建、删除、列出
# 输入/输出均为 JSON 格式

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
WORKTREE_BASE="${WORKTREE_BASE:-$HOME/.worktree}"
STATE_FILE="${STATE_FILE:-$REPO_ROOT/.pm/state.json}"
STATE_FILE_TMP="${STATE_FILE}.tmp"

# 清理临时文件
cleanup() {
    rm -f "$STATE_FILE_TMP" 2>/dev/null || true
}
trap cleanup EXIT

# 确保 jq 可用
if ! command -v jq >/dev/null 2>&1; then
    echo '{"action": "worktree", "status": "error", "error": "需要 jq: brew install jq"}' >&2
    exit 1
fi

# 输出 JSON 辅助函数
json_output() {
    local action="$1"
    local status="$2"
    local data="${3:-{}}"
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    cat <<EOF
{
  "action": "$action",
  "status": "$status",
  "data": $data,
  "timestamp": "$timestamp"
}
EOF
}

# 验证 developer_id 格式
validate_developer_id() {
    local dev_id="$1"
    if [[ ! "$dev_id" =~ ^dev-[a-z]$ ]]; then
        json_output "worktree" "error" "{\"error\": \"无效的 developer_id 格式: $dev_id (应为 dev-a 格式)\"}"
        return 1
    fi
    return 0
}

# 验证 spec_id 格式
validate_spec_id() {
    local spec_id="$1"
    if [[ ! "$spec_id" =~ ^[A-Z]+-[0-9]+$ ]]; then
        json_output "worktree" "error" "{\"error\": \"无效的 spec_id 格式: $spec_id (应为 CORE-01 格式)\"}"
        return 1
    fi
    return 0
}

# 验证 spec_id 不包含路径穿越字符
validate_safe_spec_id() {
    local spec_id="$1"
    if [[ "$spec_id" =~ \.\. ]] || [[ "$spec_id" =~ / ]] || [[ "$spec_id" =~ \\ ]]; then
        json_output "worktree" "error" "{\"error\": \"spec_id 包含非法字符: $spec_id\"}"
        return 1
    fi
    return 0
}

# 验证开发者是否存在
check_developer_exists() {
    local dev_id="$1"
    if ! jq -e ".developers.\"$dev_id\"" "$STATE_FILE" >/dev/null 2>&1; then
        json_output "worktree" "error" "{\"error\": \"开发者不存在: $dev_id\"}"
        return 1
    fi
    return 0
}

# 添加 worktree 到 state.json
add_worktree_to_state() {
    local developer="$1"
    local spec_id="$2"
    local path="$3"
    local branch="$4"
    local now
    now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    jq --arg now "$now" \
        --arg dev "$developer" \
        --arg spec "$spec_id" \
        --arg path "$path" \
        --arg branch "$branch" \
        '.worktrees += [{developer: $dev, spec_id: $spec, path: $path, branch: $branch}] | .updated_at = $now' \
        "$STATE_FILE" > "$STATE_FILE_TMP"
    mv "$STATE_FILE_TMP" "$STATE_FILE"
}

# 从 state.json 移除 worktree
remove_worktree_from_state() {
    local developer="$1"
    local spec_id="$2"
    local now
    now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    jq --arg now "$now" \
        --arg dev "$developer" \
        --arg spec "$spec_id" \
        '.worktrees |= map(select(.developer != $dev or .spec_id != $spec)) | .updated_at = $now' \
        "$STATE_FILE" > "$STATE_FILE_TMP"
    mv "$STATE_FILE_TMP" "$STATE_FILE"
}

# create 命令
cmd_create() {
    local developer_id="$1"
    local spec_id="$2"

    # 验证输入
    validate_developer_id "$developer_id" || return 1
    validate_spec_id "$spec_id" || return 1
    validate_safe_spec_id "$spec_id" || return 1
    check_developer_exists "$developer_id" || return 1

    # 读取 state.json 获取开发者命名空间
    local namespace
    namespace=$(jq -r ".developers.\"$developer_id\".namespace[]" "$STATE_FILE" 2>/dev/null | tr '\n' ',' | sed 's/,$//')

    # 生成分支名和路径
    local dev_short="${developer_id#dev-}"
    local branch_name="pr-${dev_short}-${spec_id}"
    local worktree_path="$WORKTREE_BASE/$branch_name"

    # 检查是否已存在
    if [[ -d "$worktree_path" ]]; then
        json_output "worktree" "success" "{\"path\": \"$worktree_path\", \"branch\": \"$branch_name\", \"exists\": true, \"namespace\": \"$namespace\"}"
        return 0
    fi

    # 创建 worktree
    cd "$REPO_ROOT" || {
        json_output "worktree" "error" '{"error": "无法进入项目根目录"}'
        return 1
    }

    if ! git worktree add "$worktree_path" -b "$branch_name" 2>/dev/null; then
        json_output "worktree" "error" "{\"error\": \"创建 worktree 失败: $worktree_path\"}"
        return 1
    fi

    # 更新 state.json
    add_worktree_to_state "$developer_id" "$spec_id" "$worktree_path" "$branch_name"

    json_output "worktree" "success" "{\"path\": \"$worktree_path\", \"branch\": \"$branch_name\", \"namespace\": \"$namespace\"}"
}

# remove 命令
cmd_remove() {
    local developer_id="$1"
    local spec_id="$2"

    # 验证输入
    validate_developer_id "$developer_id" || return 1
    validate_spec_id "$spec_id" || return 1
    validate_safe_spec_id "$spec_id" || return 1

    local dev_short="${developer_id#dev-}"
    local branch_name="pr-${dev_short}-${spec_id}"
    local worktree_path="$WORKTREE_BASE/$branch_name"

    cd "$REPO_ROOT" || {
        json_output "worktree" "error" '{"error": "无法进入项目根目录"}'
        return 1
    }

    # 尝试删除 worktree
    local removed=false
    if [[ -d "$worktree_path" ]]; then
        if git worktree remove "$worktree_path" 2>/dev/null; then
            removed=true
        else
            rm -rf "$worktree_path"
            git worktree prune
            removed=true
        fi
    fi

    # 从 state.json 中移除
    remove_worktree_from_state "$developer_id" "$spec_id"

    if [[ "$removed" == "true" ]]; then
        json_output "worktree" "success" "{\"removed\": \"$worktree_path\"}"
    else
        json_output "worktree" "success" "{\"removed\": null, \"note\": \"worktree 不存在，已从 state.json 中移除记录\"}"
    fi
}

# list 命令
cmd_list() {
    cd "$REPO_ROOT" || {
        json_output "worktree" "error" '{"error": "无法进入项目根目录"}'
        return 1
    }

    local worktrees_json="["
    local first=true
    while IFS= read -r line; do
        if [[ "$line" =~ ^(/.+)\ ([a-f0-9]+)\ \[(.+)\]$ ]]; then
            local path="${BASH_REMATCH[1]}"
            local commit="${BASH_REMATCH[2]}"
            local branch="${BASH_REMATCH[3]}"
            if [[ "$first" == "true" ]]; then
                first=false
            else
                worktrees_json+=","
            fi
            # 转义路径中的双引号
            path="${path//\"/\\\"}"
            worktrees_json+="{\"path\": \"$path\", \"commit\": \"$commit\", \"branch\": \"$branch\"}"
        fi
    done < <(git worktree list)
    worktrees_json+="]"

    json_output "worktree" "success" "{\"worktrees\": $worktrees_json}"
}

# sync 命令 - 同步 state.json 中的 worktrees 与实际 Git worktrees
cmd_sync() {
    cd "$REPO_ROOT" || {
        json_output "worktree" "error" '{"error": "无法进入项目根目录"}'
        return 1
    }

    local now
    now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # 获取实际的 worktree 列表
    local actual_worktrees=()
    while IFS= read -r line; do
        if [[ "$line" =~ ^(/.+)\ [a-f0-9]+\ \[(.+)\]$ ]]; then
            actual_worktrees+=("${BASH_REMATCH[1]}")
        fi
    done < <(git worktree list)

    # 获取 state.json 中的 worktree 列表
    local state_worktrees
    state_worktrees=$(jq -r '.worktrees[].path' "$STATE_FILE" 2>/dev/null || echo "")

    # 清理 state.json 中不存在的 worktree
    local new_worktrees="["
    local first=true
    for wt_path in $state_worktrees; do
        if [[ " ${actual_worktrees[*]} " =~ " ${wt_path} " ]]; then
            if [[ "$first" == "true" ]]; then
                first=false
            else
                new_worktrees+=","
            fi
            new_worktrees+=$(jq -c --arg p "$wt_path" '.worktrees[] | select(.path == $p)' "$STATE_FILE")
        fi
    done
    new_worktrees+="]"

    jq --arg now "$now" \
        --arg wt "$new_worktrees" \
        '.worktrees = $wt | .updated_at = $now' \
        "$STATE_FILE" > "$STATE_FILE_TMP"
    mv "$STATE_FILE_TMP" "$STATE_FILE"

    local actual_count=${#actual_worktrees[@]}
    local state_count=$(echo "$state_worktrees" | wc -w | xargs)
    json_output "worktree" "success" "{\"actual_count\": $actual_count, \"state_count\": $state_count, \"synced\": true}"
}

# 操作分发
ACTION="${1:-}"

case "$ACTION" in
    create)
        cmd_create "$2" "$3"
        ;;
    remove)
        cmd_remove "$2" "$3"
        ;;
    list)
        cmd_list
        ;;
    sync)
        cmd_sync
        ;;
    *)
        cat <<EOF
用法: $(basename "$0") <command> [args]

命令:
  create <dev> <spec>  创建 worktree
  remove <dev> <spec>  删除 worktree
  list                 列出所有 worktree
  sync                 同步 state.json 与实际 worktree

示例:
  $(basename "$0") create dev-a CORE-01
  $(basename "$0") remove dev-a CORE-01
  $(basename "$0") list
  $(basename "$0") sync

EOF
        exit 1
        ;;
esac
