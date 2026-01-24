#!/bin/bash
# worktree.sh - Git worktree 操作脚本
# 只处理 Git worktree 的创建、删除、列出
# 输入/输出均为 JSON 格式

set -euo pipefail

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 加载公共库
# shellcheck source=lib/lib.sh
# shellcheck source=lib/validate.sh
source "$SCRIPT_DIR/lib/lib.sh"
source "$SCRIPT_DIR/lib/validate.sh"

# ============================================
# worktree 特定配置
# ============================================
readonly WORKTREE_BASE="${WORKTREE_BASE:-$HOME/.worktree}"

# ============================================
# 确保依赖
# ============================================
if ! command -v jq >/dev/null 2>&1; then
    pm_json_output "worktree" "error" '{"error": "需要 jq: brew install jq"}' >&2
    exit 1
fi

if ! command -v git >/dev/null 2>&1; then
    pm_json_output "worktree" "error" '{"error": "需要 git"}' >&2
    exit 1
fi

# ============================================
# 内部辅助函数
# ============================================

# 添加 worktree 到 state.json
_add_worktree_to_state() {
    local developer="$1"
    local spec_id="$2"
    local path="$3"
    local branch="$4"

    if ! pm_lock_acquire; then
        return 1
    fi

    local now
    now=$(pm_now_iso)
    jq --arg now "$now" \
        --arg dev "$developer" \
        --arg spec "$spec_id" \
        --arg path "$path" \
        --arg branch "$branch" \
        '.worktrees += [{developer: $dev, spec_id: $spec, path: $path, branch: $branch}] | .updated_at = $now' \
        "$STATE_FILE" > "$STATE_FILE_TMP"

    local result=$?
    pm_lock_release
    return $result
}

# 从 state.json 移除 worktree
_remove_worktree_from_state() {
    local developer="$1"
    local spec_id="$2"

    if ! pm_lock_acquire; then
        return 1
    fi

    local now
    now=$(pm_now_iso)
    jq --arg now "$now" \
        --arg dev "$developer" \
        --arg spec "$spec_id" \
        '.worktrees |= map(select(.developer != $dev or .spec_id != $spec)) | .updated_at = $now' \
        "$STATE_FILE" > "$STATE_FILE_TMP"

    local result=$?
    pm_lock_release
    return $result
}

# ============================================
# create 命令 - 创建 worktree
# ============================================
cmd_create() {
    local developer_id="$1"
    local spec_id="$2"

    # 验证输入
    pm_validate_developer_id "$developer_id" "worktree" || return 1
    pm_validate_spec_id "$spec_id" "worktree" || return 1
    pm_validate_safe_spec_id "$spec_id" "worktree" || return 1
    pm_check_developer_exists "$developer_id" "worktree" || return 1

    # 读取 state.json 获取开发者命名空间
    local namespace
    namespace=$(jq -r ".developers.\"$developer_id\".namespace[]" "$STATE_FILE" 2>/dev/null | tr '\n' ',' | sed 's/,$//')

    # 生成分支名和路径
    local dev_short="${developer_id#dev-}"
    local branch_name="pr-${dev_short}-${spec_id}"
    local worktree_path="$WORKTREE_BASE/$branch_name"

    # 检查是否已存在
    if [[ -d "$worktree_path" ]]; then
        pm_json_output "worktree" "success" "{\"path\": \"$worktree_path\", \"branch\": \"$branch_name\", \"exists\": true, \"namespace\": \"$namespace\"}"
        return 0
    fi

    # 创建 worktree
    cd "$REPO_ROOT" || {
        pm_json_output "worktree" "error" '{"error": "无法进入项目根目录"}'
        return 1
    }

    if ! git worktree add "$worktree_path" -b "$branch_name" 2>/dev/null; then
        pm_json_output "worktree" "error" "{\"error\": \"创建 worktree 失败: $worktree_path\"}"
        return 1
    fi

    # 更新 state.json
    _add_worktree_to_state "$developer_id" "$spec_id" "$worktree_path" "$branch_name"

    pm_json_output "worktree" "success" "{\"path\": \"$worktree_path\", \"branch\": \"$branch_name\", \"namespace\": \"$namespace\"}"
}

# ============================================
# remove 命令 - 删除 worktree
# ============================================
cmd_remove() {
    local developer_id="$1"
    local spec_id="$2"

    # 验证输入
    pm_validate_developer_id "$developer_id" "worktree" || return 1
    pm_validate_spec_id "$spec_id" "worktree" || return 1
    pm_validate_safe_spec_id "$spec_id" "worktree" || return 1

    local dev_short="${developer_id#dev-}"
    local branch_name="pr-${dev_short}-${spec_id}"
    local worktree_path="$WORKTREE_BASE/$branch_name"

    cd "$REPO_ROOT" || {
        pm_json_output "worktree" "error" '{"error": "无法进入项目根目录"}'
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
    _remove_worktree_from_state "$developer_id" "$spec_id"

    if [[ "$removed" == "true" ]]; then
        pm_json_output "worktree" "success" "{\"removed\": \"$worktree_path\"}"
    else
        pm_json_output "worktree" "success" "{\"removed\": null, \"note\": \"worktree 不存在，已从 state.json 中移除记录\"}"
    fi
}

# ============================================
# list 命令 - 列出所有 worktree
# ============================================
cmd_list() {
    cd "$REPO_ROOT" || {
        pm_json_output "worktree" "error" '{"error": "无法进入项目根目录"}'
        return 1
    }

    # 使用 jq 构建 JSON 数组
    local worktrees_json="[]"

    while IFS= read -r line; do
        if [[ "$line" =~ ^(/.+)\ ([a-f0-9]+)\ \[(.+)\]$ ]]; then
            local path="${BASH_REMATCH[1]}"
            local commit="${BASH_REMATCH[2]}"
            local branch="${BASH_REMATCH[3]}"
            # 使用 jq 添加到数组
            worktrees_json=$(echo "$worktrees_json" | jq \
                --arg p "$path" \
                --arg c "$commit" \
                --arg b "$branch" \
                '. += [{path: $p, commit: $c, branch: $b}]')
        fi
    done < <(git worktree list)

    pm_json_output "worktree" "success" "{\"worktrees\": $worktrees_json}"
}

# ============================================
# sync 命令 - 同步 state.json 与实际 Git worktrees
# ============================================
cmd_sync() {
    cd "$REPO_ROOT" || {
        pm_json_output "worktree" "error" '{"error": "无法进入项目根目录"}'
        return 1
    fi

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
    local new_worktrees="[]"

    if ! pm_lock_acquire; then
        pm_json_output "worktree" "error" '{"error": "无法获取文件锁"}'
        return 1
    fi

    for wt_path in $state_worktrees; do
        if [[ " ${actual_worktrees[*]} " =~ " ${wt_path} " ]]; then
            # worktree 仍然存在，保留
            local wt_entry
            wt_entry=$(jq -c --arg p "$wt_path" '.worktrees[] | select(.path == $p)' "$STATE_FILE")
            new_worktrees=$(echo "$new_worktrees" | jq --arg e "$wt_entry" '. += [$e | fromjson]')
        fi
    done

    # 更新 state.json
    local now
    now=$(pm_now_iso)
    echo "$new_worktrees" | jq --arg now "$now" '.worktrees = $in | .updated_at = $now' > "$STATE_FILE_TMP"

    local result=$?
    pm_lock_release

    if [[ $result -ne 0 ]]; then
        pm_json_output "worktree" "error" '{"error": "更新 state.json 失败"}'
        return 1
    fi

    mv "$STATE_FILE_TMP" "$STATE_FILE"

    local actual_count=${#actual_worktrees[@]}
    local state_count=$(echo "$state_worktrees" | wc -w | xargs)
    pm_json_output "worktree" "success" "{\"actual_count\": $actual_count, \"state_count\": $state_count, \"synced\": true}"
}

# ============================================
# 操作分发
# ============================================
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

环境变量:
  WORKTREE_BASE  worktree 基础目录 (默认: ~/.worktree)

示例:
  $(basename "$0") create dev-a CORE-01
  $(basename "$0") remove dev-a CORE-01
  $(basename "$0") list
  $(basename "$0") sync

EOF
        exit 1
        ;;
esac
