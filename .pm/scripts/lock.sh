#!/bin/bash
# lock.sh - 文件锁操作脚本
# 只处理锁的获取、释放、列出
# 输入/输出均为 JSON 格式

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
    echo '{"action": "lock", "status": "error", "error": "需要 jq: brew install jq"}' >&2
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

# 验证 lock_name 格式 (只允许字母数字下划线)
validate_lock_name() {
    local lock_name="$1"
    if [[ ! "$lock_name" =~ ^[a-z_]+$ ]]; then
        json_output "lock" "error" "{\"error\": \"无效的 lock_name 格式: $lock_name (只允许小写字母和下划线)\"}"
        return 1
    fi
    return 0
}

# 验证 developer_id 格式
validate_developer_id() {
    local dev_id="$1"
    if [[ ! "$dev_id" =~ ^dev-[a-z]$ ]]; then
        json_output "lock" "error" "{\"error\": \"无效的 developer_id 格式: $dev_id (应为 dev-a 格式)\"}"
        return 1
    fi
    return 0
}

# 验证 spec_id 格式
validate_spec_id() {
    local spec_id="$1"
    if [[ ! "$spec_id" =~ ^[A-Z]+-[0-9]+$ ]]; then
        json_output "lock" "error" "{\"error\": \"无效的 spec_id 格式: $spec_id (应为 CORE-01 格式)\"}"
        return 1
    fi
    return 0
}

# 获取当前 UTC 时间戳（秒）
timestamp_now() {
    date +%s
}

# 将 ISO 时间转换为 Unix 时间戳
timestamp_parse() {
    local iso_time="$1"
    if date -j -f "%Y-%m-%dT%H:%M:%SZ" "$iso_time" +%s 2>/dev/null; then
        :
    elif date -d "$iso_time" +%s 2>/dev/null; then
        :
    else
        echo "0"
    fi
}

# 获取偏移后的时间
date_offset() {
    local offset="$1"
    if date -v+1H >/dev/null 2>&1; then
        # macOS
        case "$offset" in
            *hour*|*hours*)
                local num="${offset%% *}"
                date -u -v+${num}H +"%Y-%m-%dT%H:%M:%SZ"
                ;;
            *day*|*days*)
                local num="${offset%% *}"
                date -u -v+${num}d +"%Y-%m-%dT%H:%M:%SZ"
                ;;
            *)
                date -u +"%Y-%m-%dT%H:%M:%SZ"
                ;;
        esac
    else
        # Linux
        date -u -d "$offset" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ"
    fi
}

# acquire 命令
cmd_acquire() {
    local lock_name="$1"
    local developer_id="$2"
    local spec_id="$3"
    local reason="${4:-}"
    local duration="${5:-+6 hours}"

    # 验证输入
    validate_lock_name "$lock_name" || return 1
    validate_developer_id "$developer_id" || return 1
    validate_spec_id "$spec_id" || return 1

    # 检查锁是否已存在
    if jq -e ".locks.\"$lock_name\"" "$STATE_FILE" >/dev/null 2>&1; then
        local lock_info
        lock_info=$(jq -c ".locks.\"$lock_name\"" "$STATE_FILE")
        json_output "lock" "error" "{\"error\": \"锁已被持有\", \"lock\": $lock_info}"
        return 1
    fi

    # 计算过期时间
    local now expires_at
    now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    expires_at=$(date_offset "$duration")

    # 更新 state.json
    jq --arg now "$now" \
        --arg expires "$expires_at" \
        --arg lock "$lock_name" \
        --arg owner "$developer_id" \
        --arg spec "$spec_id" \
        --arg reason "$reason" \
        '
        .locks[$lock] = {
            locked_by: $owner,
            locked_at: $now,
            spec_id: $spec,
            reason: $reason,
            expires_at: $expires
        } | .updated_at = $now
        ' "$STATE_FILE" > "$STATE_FILE_TMP"

    if mv "$STATE_FILE_TMP" "$STATE_FILE"; then
        local new_lock
        new_lock=$(jq -c ".locks.\"$lock_name\"" "$STATE_FILE")
        json_output "lock" "success" "{\"lock_name\": \"$lock_name\", \"lock\": $new_lock}"
    else
        json_output "lock" "error" '{"error": "更新 state.json 失败"}'
        return 1
    fi
}

# release 命令
cmd_release() {
    local lock_name="$1"

    validate_lock_name "$lock_name" || return 1

    # 检查锁是否存在
    if ! jq -e ".locks.\"$lock_name\"" "$STATE_FILE" >/dev/null 2>&1; then
        json_output "lock" "error" "{\"error\": \"锁不存在: $lock_name\"}"
        return 1
    fi

    # 更新 state.json 删除锁
    local now
    now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    jq --arg now "$now" \
        --arg lock "$lock_name" \
        'del(.locks[$lock]) | .updated_at = $now' \
        "$STATE_FILE" > "$STATE_FILE_TMP"

    if mv "$STATE_FILE_TMP" "$STATE_FILE"; then
        json_output "lock" "success" "{\"lock_name\": \"$lock_name\", \"released\": true}"
    else
        json_output "lock" "error" '{"error": "更新 state.json 失败"}'
        return 1
    fi
}

# list 命令
cmd_list() {
    local locks
    locks=$(jq -c '.locks' "$STATE_FILE")
    json_output "lock" "success" "{\"locks\": $locks}"
}

# check 命令
cmd_check() {
    local lock_name="$1"

    validate_lock_name "$lock_name" || return 1

    if jq -e ".locks.\"$lock_name\"" "$STATE_FILE" >/dev/null 2>&1; then
        local lock_info
        lock_info=$(jq -c ".locks.\"$lock_name\"" "$STATE_FILE")
        json_output "lock" "success" "{\"lock_name\": \"$lock_name\", \"locked\": true, \"lock\": $lock_info}"
    else
        json_output "lock" "success" "{\"lock_name\": \"$lock_name\", \"locked\": false}"
    fi
}

# prune 命令 - 清理过期锁
cmd_prune() {
    local now_ts
    now_ts=$(timestamp_now)

    # 获取所有锁
    local locks
    locks=$(jq -r '.locks | keys[]' "$STATE_FILE" 2>/dev/null || echo "")

    local pruned=0
    local pruned_list="[]"

    for lock_name in $locks; do
        if [[ -z "$lock_name" ]]; then
            continue
        fi

        local expires_at
        expires_at=$(jq -r ".locks.\"$lock_name\".expires_at" "$STATE_FILE" 2>/dev/null || echo "")
        if [[ -z "$expires_at" || "$expires_at" == "null" ]]; then
            continue
        fi

        local expiry_ts
        expiry_ts=$(timestamp_parse "$expires_at")

        # 如果锁已过期
        if [[ $expiry_ts -gt 0 ]] && [[ $now_ts -gt $expiry_ts ]]; then
            # 删除锁
            jq --arg lock "$lock_name" 'del(.locks[$lock])' "$STATE_FILE" > "$STATE_FILE_TMP"
            if mv "$STATE_FILE_TMP" "$STATE_FILE"; then
                ((pruned++))
                pruned_list=$(echo "$pruned_list" | jq --arg l "$lock_name" '. + [$l]')
            fi
        fi
    done

    # 更新 updated_at
    local now
    now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    jq --arg now "$now" '.updated_at = $now' "$STATE_FILE" > "$STATE_FILE_TMP"
    mv "$STATE_FILE_TMP" "$STATE_FILE"

    json_output "lock" "success" "{\"pruned\": $pruned, \"locks\": $pruned_list}"
}

# 操作分发
ACTION="${1:-}"

case "$ACTION" in
    acquire)
        cmd_acquire "$2" "$3" "$4" "${5:-}" "${6:-+6 hours}"
        ;;
    release)
        cmd_release "$2"
        ;;
    list)
        cmd_list
        ;;
    check)
        cmd_check "$2"
        ;;
    prune)
        cmd_prune
        ;;
    *)
        cat <<EOF
用法: $(basename "$0") <command> [args]

命令:
  acquire <lock> <dev> <spec> [reason] [duration]  获取锁
  release <lock>                                释放锁
  list                                         列出所有锁
  check <lock>                                 检查锁状态
  prune                                        清理过期锁

示例:
  $(basename "$0") acquire runner dev-a CORE-01 "实现 Runner 生命周期"
  $(basename "$0") release runner
  $(basename "$0") list
  $(basename "$0") check runner
  $(basename "$0") prune

EOF
        exit 1
        ;;
esac
