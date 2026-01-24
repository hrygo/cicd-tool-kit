#!/bin/bash
# state.sh - 结构化状态操作脚本
# 只读写 JSON，由 AI Agent 调用

set -euo pipefail

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 加载公共库
# shellcheck source=lib/lib.sh
# shellcheck source=lib/validate.sh
source "$SCRIPT_DIR/lib/lib.sh"
source "$SCRIPT_DIR/lib/validate.sh"

# ============================================
# 确保依赖
# ============================================
if ! command -v jq >/dev/null 2>&1; then
    pm_json_output "state" "error" '{"error": "需要 jq: brew install jq"}' >&2
    exit 1
fi

# 确保 state.json 存在
if [[ ! -f "$STATE_FILE" ]]; then
    pm_json_output "state" "error" '{"error": "状态文件不存在"}' >&2
    exit 1
fi

# ============================================
# read 命令 - 读取状态
# ============================================
cmd_read() {
    local path="${1:-.}"
    jq -c "$path" "$STATE_FILE"
}

# ============================================
# update 命令 - 更新指定路径
# ============================================
cmd_update() {
    local path="$1"
    local value="$2"

    if ! pm_lock_acquire; then
        pm_json_output "state" "error" '{"error": "无法获取文件锁"}'
        return 1
    fi

    local now
    now=$(pm_now_iso)
    jq --arg now "$now" "$path = $value | .updated_at = \$now" "$STATE_FILE" > "$STATE_FILE_TMP"

    if ! mv "$STATE_FILE_TMP" "$STATE_FILE"; then
        pm_lock_release
        pm_json_output "state" "error" '{"error": "更新 state.json 失败"}'
        return 1
    fi

    pm_lock_release
    jq -c "$path" "$STATE_FILE"
}

# ============================================
# set 命令 - 批量设置 (从 stdin)
# ============================================
cmd_set() {
    local input
    input=$(cat)

    if ! pm_lock_acquire; then
        pm_json_output "state" "error" '{"error": "无法获取文件锁"}'
        return 1
    fi

    local now
    now=$(pm_now_iso)
    echo "$input" | jq --arg now "$now" '. * $in | .updated_at = $now' > "$STATE_FILE_TMP"

    if ! mv "$STATE_FILE_TMP" "$STATE_FILE"; then
        pm_lock_release
        pm_json_output "state" "error" '{"error": "更新 state.json 失败"}'
        return 1
    fi

    pm_lock_release
    cat "$STATE_FILE"
}

# ============================================
# assign 命令 - 分配任务给开发者
# ============================================
cmd_assign() {
    local spec_id="$1"
    local dev_id="$2"

    # 验证输入
    pm_validate_spec_id "$spec_id" "state" || return 1
    pm_validate_developer_id "$dev_id" "state" || return 1
    pm_check_developer_exists "$dev_id" "state" || return 1
    pm_check_spec_exists "$spec_id" "state" || return 1

    # 获取锁
    if ! pm_lock_acquire; then
        pm_json_output "state" "error" '{"error": "无法获取文件锁"}'
        return 1
    fi

    local now
    now=$(pm_now_iso)

    # 1. 检查开发者是否有进行中任务
    local current_task
    current_task=$(jq -r ".developers.\"$dev_id\".current_task" "$STATE_FILE")
    if [[ "$current_task" != "null" && -n "$current_task" ]]; then
        pm_lock_release
        pm_json_output "assign" "error" "{\"error\": \"开发者有进行中任务: $current_task\"}"
        return 1
    fi

    # 2. 检查 Spec 依赖
    local deps
    deps=$(jq -r ".specs.\"$spec_id\".dependencies[]" "$STATE_FILE" 2>/dev/null || echo "")
    for dep in $deps; do
        local dep_status
        dep_status=$(jq -r ".specs.\"$dep\".status" "$STATE_FILE" 2>/dev/null || echo "unknown")
        if [[ "$dep_status" != "completed" ]]; then
            pm_lock_release
            pm_json_output "assign" "error" "{\"error\": \"依赖未满足: $dep ($dep_status)\"}"
            return 1
        fi
    done

    # 3. 更新状态
    jq --arg now "$now" \
        --arg spec "$spec_id" \
        --arg dev "$dev_id" \
        '
        .specs[$spec].status = "in_progress" |
        .specs[$spec].assignee = $dev |
        .developers[$dev].current_task = $spec |
        .updated_at = $now
        ' "$STATE_FILE" > "$STATE_FILE_TMP"

    if ! mv "$STATE_FILE_TMP" "$STATE_FILE"; then
        pm_lock_release
        pm_json_output "state" "error" '{"error": "更新 state.json 失败"}'
        return 1
    fi

    pm_lock_release

    # 4. 输出结果
    local spec_info dev_info
    spec_info=$(jq -c ".specs.\"$spec_id\"" "$STATE_FILE")
    dev_info=$(jq -c ".developers.\"$dev_id\"" "$STATE_FILE")
    pm_json_output "assign" "success" "{\"spec\": $spec_info, \"developer\": $dev_info}"
}

# ============================================
# complete 命令 - 完成任务
# ============================================
cmd_complete() {
    local spec_id="$1"
    local dev_id="$2"

    # 验证输入
    pm_validate_spec_id "$spec_id" "state" || return 1
    pm_validate_developer_id "$dev_id" "state" || return 1
    pm_check_developer_exists "$dev_id" "state" || return 1
    pm_check_spec_exists "$spec_id" "state" || return 1

    # 获取锁
    if ! pm_lock_acquire; then
        pm_json_output "state" "error" '{"error": "无法获取文件锁"}'
        return 1
    fi

    local now
    now=$(pm_now_iso)

    jq --arg now "$now" \
        --arg spec "$spec_id" \
        --arg dev "$dev_id" \
        '
        .specs[$spec].status = "completed" |
        .specs[$spec].completed_at = $now |
        .developers[$dev].current_task = null |
        .developers[$dev].completed_specs += [$spec] |
        .updated_at = $now
        ' "$STATE_FILE" > "$STATE_FILE_TMP"

    if ! mv "$STATE_FILE_TMP" "$STATE_FILE"; then
        pm_lock_release
        pm_json_output "state" "error" '{"error": "更新 state.json 失败"}'
        return 1
    fi

    pm_lock_release

    local spec_info dev_info
    spec_info=$(jq -c ".specs.\"$spec_id\"" "$STATE_FILE")
    dev_info=$(jq -c ".developers.\"$dev_id\"" "$STATE_FILE")
    pm_json_output "complete" "success" "{\"spec\": $spec_info, \"developer\": $dev_info}"
}

# ============================================
# progress 命令 - 计算进度统计
# ============================================
cmd_progress() {
    local total completed in_progress ready blocked
    total=$(jq '[.specs | to_entries[] | select(.value.status != "completed")] | length' "$STATE_FILE")
    completed=$(jq '[.specs | to_entries[] | select(.value.status == "completed")] | length' "$STATE_FILE")
    in_progress=$(jq '[.specs | to_entries[] | select(.value.status == "in_progress")] | length' "$STATE_FILE")
    ready=$(jq '[.specs | to_entries[] | select(.value.status == "ready")] | length' "$STATE_FILE")
    blocked=$(jq '[.specs | to_entries[] | select(.value.status == "blocked")] | length' "$STATE_FILE")

    local total_specs percent
    total_specs=$(jq '.specs | length' "$STATE_FILE")
    if [[ $total_specs -gt 0 ]]; then
        percent=$((completed * 100 / total_specs))
    else
        percent=0
    fi

    # 使用 jq 生成 JSON (避免 heredoc)
    jq -n \
        --arg a "progress" \
        --arg s "success" \
        --arg ts "$(pm_now_iso)" \
        --arg p "$percent%" \
        --arg tot "$total_specs" \
        --arg comp "$completed" \
        --arg prog "$in_progress" \
        --arg rd "$ready" \
        --arg blk "$blocked" \
        '{
            action: $a,
            status: $s,
            data: {
                summary: {
                    total_progress: $p,
                    total: ($tot | tonumber),
                    completed: ($comp | tonumber),
                    in_progress: ($prog | tonumber),
                    ready: ($rd | tonumber),
                    blocked: ($blk | tonumber)
                }
            },
            timestamp: $ts
        }' | jq '
            .data.developers = {
                "dev-a": (.input.developers["dev-a"] // {current_task: null, completed_specs: 0}),
                "dev-b": (.input.developers["dev-b"] // {current_task: null, completed_specs: 0}),
                "dev-c": (.input.developers["dev-c"] // {current_task: null, completed_specs: 0})
            } |
            .data.active_locks = (.input.locks | keys) |
            .data.active_worktrees = (.input.worktrees | length)
        ' "$STATE_FILE"
}

# ============================================
# validate 命令 - 验证状态文件
# ============================================
cmd_validate() {
    local schema="$REPO_ROOT/.pm/state.schema.json"

    if command -v ajv >/dev/null 2>&1; then
        if ajv test -s "$schema" -d "$STATE_FILE" --valid 2>&1 | grep -q "true"; then
            pm_json_output "validate" "success" '{"valid": true}'
        else
            pm_json_output "validate" "error" '{"valid": false}'
        fi
    elif command -v check-jsonschema >/dev/null 2>&1; then
        if check-jsonschema "$STATE_FILE" "$schema" 2>&1 | grep -q "PASS"; then
            pm_json_output "validate" "success" '{"valid": true}'
        else
            pm_json_output "validate" "error" '{"valid": false}'
        fi
    else
        # 基础验证
        if jq '.' "$STATE_FILE" >/dev/null 2>&1; then
            pm_json_output "validate" "success" '{"valid": true, "note": "安装 ajv 或 check-jsonschema 进行完整验证"}'
        else
            pm_json_output "validate" "error" '{"valid": false}'
        fi
    fi
}

# ============================================
# 操作分发
# ============================================
ACTION="${1:-}"

case "$ACTION" in
    read)
        cmd_read "$2"
        ;;
    update)
        cmd_update "$2" "$3"
        ;;
    set)
        cmd_set
        ;;
    assign)
        cmd_assign "$2" "$3"
        ;;
    complete)
        cmd_complete "$2" "$3"
        ;;
    progress)
        cmd_progress
        ;;
    validate)
        cmd_validate
        ;;
    *)
        cat <<EOF
用法: $(basename "$0") <command> [args]

命令:
  read [path]           读取状态 (jq 路径)
  update <path> <value> 更新指定路径
  set                   批量设置 (从 stdin)
  assign <spec> <dev>   分配任务
  complete <spec> <dev> 完成任务
  progress              计算进度统计
  validate              验证状态文件

示例:
  $(basename "$0") read .specs.CORE-01
  $(basename "$0") update .specs.CORE-01.status 'in_progress'
  $(basename "$0") assign CORE-01 dev-a
  $(basename "$0") complete CORE-01 dev-a
  $(basename "$0") progress

EOF
        exit 1
        ;;
esac
