#!/bin/bash
# state.sh - 结构化状态操作脚本
# 只读写 JSON，由 AI Agent 调用

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
STATE_FILE="${STATE_FILE:-$REPO_ROOT/.pm/state.json}"
STATE_FILE_TMP="${STATE_FILE}.tmp"

# 清理临时文件
cleanup() {
    rm -f "$STATE_FILE_TMP" 2>/dev/null || true
}
trap cleanup EXIT

# 确保 jq 可用
if ! command -v jq >/dev/null 2>&1; then
    echo '{"action": "state", "status": "error", "error": "需要 jq: brew install jq"}' >&2
    exit 1
fi

# 确保 state.json 存在
if [[ ! -f "$STATE_FILE" ]]; then
    echo '{"action": "state", "status": "error", "error": "状态文件不存在"}' >&2
    exit 1
fi

# 验证 spec_id 格式 ([A-Z]+-[0-9]+)
validate_spec_id() {
    local spec_id="$1"
    if [[ ! "$spec_id" =~ ^[A-Z]+-[0-9]+$ ]]; then
        echo "{\"action\": \"state\", \"status\": \"error\", \"error\": \"无效的 spec_id 格式: $spec_id (应为 CORE-01 格式)\"}"
        return 1
    fi
    return 0
}

# 验证 developer_id 格式 (dev-[a-z])
validate_developer_id() {
    local dev_id="$1"
    if [[ ! "$dev_id" =~ ^dev-[a-z]$ ]]; then
        echo "{\"action\": \"state\", \"status\": \"error\", \"error\": \"无效的 developer_id 格式: $dev_id (应为 dev-a 格式)\"}"
        return 1
    fi
    return 0
}

# 验证开发者是否存在
check_developer_exists() {
    local dev_id="$1"
    if ! jq -e ".developers.\"$dev_id\"" "$STATE_FILE" >/dev/null 2>&1; then
        echo "{\"action\": \"state\", \"status\": \"error\", \"error\": \"开发者不存在: $dev_id\"}"
        return 1
    fi
    return 0
}

# 验证 Spec 是否存在
check_spec_exists() {
    local spec_id="$1"
    if ! jq -e ".specs.\"$spec_id\"" "$STATE_FILE" >/dev/null 2>&1; then
        echo "{\"action\": \"state\", \"status\": \"error\", \"error\": \"Spec 不存在: $spec_id\"}"
        return 1
    fi
    return 0
}

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

# read 命令
cmd_read() {
    local path="${1:-.}"
    jq -c "$path" "$STATE_FILE"
}

# update 命令
cmd_update() {
    local path="$1"
    local value="$2"
    local now
    now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    jq --arg now "$now" "$path = $value | .updated_at = \$now" "$STATE_FILE" > "$STATE_FILE_TMP"
    mv "$STATE_FILE_TMP" "$STATE_FILE"
    jq -c "$path" "$STATE_FILE"
}

# set 命令
cmd_set() {
    local now
    now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    jq --arg now "$now" '.updated_at = $now' "$STATE_FILE" \
        | jq --arg now "$now" '. * $in | .updated_at = $now' \
        > "$STATE_FILE_TMP"
    mv "$STATE_FILE_TMP" "$STATE_FILE"
    cat "$STATE_FILE"
}

# assign 命令
cmd_assign() {
    local spec_id="$1"
    local dev_id="$2"

    # 验证输入
    validate_spec_id "$spec_id" || return 1
    validate_developer_id "$dev_id" || return 1
    check_developer_exists "$dev_id" || return 1
    check_spec_exists "$spec_id" || return 1

    local now
    now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # 1. 检查开发者是否有进行中任务
    local current_task
    current_task=$(jq -r ".developers.\"$dev_id\".current_task" "$STATE_FILE")
    if [[ "$current_task" != "null" && -n "$current_task" ]]; then
        echo "{\"action\": \"assign\", \"status\": \"error\", \"error\": \"开发者有进行中任务: $current_task\"}"
        return 1
    fi

    # 2. 检查 Spec 依赖
    local deps
    deps=$(jq -r ".specs.\"$spec_id\".dependencies[]" "$STATE_FILE" 2>/dev/null || echo "")
    for dep in $deps; do
        local dep_status
        dep_status=$(jq -r ".specs.\"$dep\".status" "$STATE_FILE" 2>/dev/null || echo "unknown")
        if [[ "$dep_status" != "completed" ]]; then
            echo "{\"action\": \"assign\", \"status\": \"error\", \"error\": \"依赖未满足: $dep ($dep_status)\"}"
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
    mv "$STATE_FILE_TMP" "$STATE_FILE"

    # 4. 输出结果
    local spec_info dev_info
    spec_info=$(jq -c ".specs.\"$spec_id\"" "$STATE_FILE")
    dev_info=$(jq -c ".developers.\"$dev_id\"" "$STATE_FILE")
    echo "{\"action\": \"assign\", \"status\": \"success\", \"data\": {\"spec\": $spec_info, \"developer\": $dev_info}}"
}

# complete 命令
cmd_complete() {
    local spec_id="$1"
    local dev_id="$2"

    # 验证输入
    validate_spec_id "$spec_id" || return 1
    validate_developer_id "$dev_id" || return 1
    check_developer_exists "$dev_id" || return 1
    check_spec_exists "$spec_id" || return 1

    local now
    now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

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
    mv "$STATE_FILE_TMP" "$STATE_FILE"

    local spec_info dev_info
    spec_info=$(jq -c ".specs.\"$spec_id\"" "$STATE_FILE")
    dev_info=$(jq -c ".developers.\"$dev_id\"" "$STATE_FILE")
    echo "{\"action\": \"complete\", \"status\": \"success\", \"data\": {\"spec\": $spec_info, \"developer\": $dev_info}}"
}

# progress 命令
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

    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    cat <<EOF
{
  "action": "progress",
  "status": "success",
  "data": {
    "summary": {
      "total_progress": "$percent%",
      "total": $total_specs,
      "completed": $completed,
      "in_progress": $in_progress,
      "ready": $ready,
      "blocked": $blocked
    },
    "developers": {
      "dev-a": $(jq -c '.developers["dev-a"] | {current_task, completed_specs: (.completed_specs | length)}' "$STATE_FILE"),
      "dev-b": $(jq -c '.developers["dev-b"] | {current_task, completed_specs: (.completed_specs | length)}' "$STATE_FILE"),
      "dev-c": $(jq -c '.developers["dev-c"] | {current_task, completed_specs: (.completed_specs | length)}' "$STATE_FILE")
    },
    "active_locks": $(jq -c '.locks | keys' "$STATE_FILE"),
    "active_worktrees": $(jq -c '.worktrees | length' "$STATE_FILE")
  },
  "timestamp": "$timestamp"
}
EOF
}

# validate 命令
cmd_validate() {
    local schema="$REPO_ROOT/.pm/state.schema.json"
    if command -v ajv >/dev/null 2>&1; then
        ajv test -s "$schema" -d "$STATE_FILE" --valid 2>&1 | head -1
    elif command -v check-jsonschema >/dev/null 2>&1; then
        if check-jsonschema "$STATE_FILE" "$schema" 2>&1 | grep -q "PASS"; then
            echo '{"action": "validate", "status": "success", "valid": true}'
        else
            echo '{"action": "validate", "status": "error", "valid": false}'
        fi
    else
        # 基础验证
        if jq '.' "$STATE_FILE" >/dev/null 2>&1; then
            echo '{"action": "validate", "status": "success", "valid": true, "note": "安装 ajv 或 check-jsonschema 进行完整验证"}'
        else
            echo '{"action": "validate", "status": "error", "valid": false}'
        fi
    fi
}

# 操作分发
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
  $(basename "$0") update .specs.CORE-01.status '"'"'in_progress'"'"
  $(basename "$0") assign CORE-01 dev-a
  $(basename "$0") complete CORE-01 dev-a
  $(basename "$0") progress

EOF
        exit 1
        ;;
esac
