#!/bin/bash
# lock.sh - 文件锁操作脚本
# 只处理锁的获取、释放、列出
# 输出/输出均为 JSON 格式

set -euo pipefail

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 加载公共库
# shellcheck source=lib/lib.sh
# shellcheck source=lib/validate.sh
source "$SCRIPT_DIR/lib/lib.sh"
source "$SCRIPT_DIR/lib/validate.sh"

# ============================================
# 配置常量
# ============================================
# 默认锁超时时间（可通过环境变量 LOCK_DEFAULT_DURATION 覆盖）
readonly LOCK_DEFAULT_DURATION="${LOCK_DEFAULT_DURATION:-+6 hours}"

# ============================================
# acquire 命令 - 获取锁
# ============================================
cmd_acquire() {
    local lock_name="$1"
    local developer_id="$2"
    local spec_id="$3"
    local reason="${4:-}"
    local duration="${5:-$LOCK_DEFAULT_DURATION}"

    # 验证输入
    pm_validate_lock_name "$lock_name" "lock" || return 1
    pm_validate_developer_id "$developer_id" "lock" || return 1
    pm_validate_spec_id "$spec_id" "lock" || return 1

    # 检查锁是否已存在
    if pm_check_lock_exists "$lock_name"; then
        local lock_info
        lock_info=$(jq -c ".locks.\"$lock_name\"" "$STATE_FILE")
        pm_json_output "lock" "error" "{\"error\": \"锁已被持有\", \"lock\": $lock_info}"
        return 1
    fi

    # 计算过期时间
    local now expires_at
    now=$(pm_now_iso)
    expires_at=$(pm_date_offset "$duration")

    # 获取锁并写入 state.json
    if ! pm_lock_acquire; then
        pm_json_output "lock" "error" '{"error": "无法获取文件锁"}'
        return 1
    fi

    # 执行 jq 操作并写入
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

    if ! mv "$STATE_FILE_TMP" "$STATE_FILE"; then
        pm_lock_release
        pm_json_output "lock" "error" '{"error": "更新 state.json 失败"}'
        return 1
    fi

    pm_lock_release

    local new_lock
    new_lock=$(jq -c ".locks.\"$lock_name\"" "$STATE_FILE")
    pm_json_output "lock" "success" "{\"lock_name\": \"$lock_name\", \"lock\": $new_lock}"
}

# ============================================
# release 命令 - 释放锁
# ============================================
cmd_release() {
    local lock_name="$1"

    pm_validate_lock_name "$lock_name" "lock" || return 1

    # 检查锁是否存在
    if ! pm_check_lock_exists "$lock_name"; then
        pm_json_output "lock" "error" "{\"error\": \"锁不存在: $lock_name\"}"
        return 1
    fi

    # 获取锁并写入
    if ! pm_lock_acquire; then
        pm_json_output "lock" "error" '{"error": "无法获取文件锁"}'
        return 1
    fi

    local now
    now=$(pm_now_iso)
    jq --arg now "$now" \
        --arg lock "$lock_name" \
        'del(.locks[$lock]) | .updated_at = $now' \
        "$STATE_FILE" > "$STATE_FILE_TMP"

    if ! mv "$STATE_FILE_TMP" "$STATE_FILE"; then
        pm_lock_release
        pm_json_output "lock" "error" '{"error": "更新 state.json 失败"}'
        return 1
    fi

    pm_lock_release
    pm_json_output "lock" "success" "{\"lock_name\": \"$lock_name\", \"released\": true}"
}

# ============================================
# list 命令 - 列出所有锁
# ============================================
cmd_list() {
    local locks
    locks=$(jq -c '.locks' "$STATE_FILE")
    # 直接传递 jq 输出给 pm_json_output
    pm_json_output "lock" "success" "$locks"
}

# ============================================
# check 命令 - 检查锁状态
# ============================================
cmd_check() {
    local lock_name="$1"

    pm_validate_lock_name "$lock_name" "lock" || return 1

    if pm_check_lock_exists "$lock_name"; then
        local lock_info
        lock_info=$(jq -c ".locks.\"$lock_name\"" "$STATE_FILE")
        local output
        output=$(jq -c -n --arg ln "$lock_name" --argjson li "$lock_info" '{lock_name: $ln, locked: true, lock: $li}')
        pm_json_output "lock" "success" "$output"
    else
        local output
        output=$(jq -c -n --arg ln "$lock_name" '{lock_name: $ln, locked: false}')
        pm_json_output "lock" "success" "$output"
    fi
}

# ============================================
# prune 命令 - 清理过期锁
# ============================================
cmd_prune() {
    local now_ts
    now_ts=$(pm_timestamp_now)

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
        expiry_ts=$(pm_timestamp_parse "$expires_at")

        # 解析失败时，expiry_ts 为 -1
        if [[ $expiry_ts -lt 0 ]]; then
            # 解析失败，跳过此锁
            continue
        fi

        # 锁已过期
        if [[ $now_ts -gt $expiry_ts ]]; then
            # 获取锁并删除
            if pm_lock_acquire; then
                local now
                now=$(pm_now_iso)
                jq --arg now "$now" \
                    --arg lock "$lock_name" \
                    'del(.locks[$lock]) | .updated_at = $now' \
                    "$STATE_FILE" > "$STATE_FILE_TMP"

                if mv "$STATE_FILE_TMP" "$STATE_FILE"; then
                    ((pruned++))
                    pruned_list=$(echo "$pruned_list" | jq --arg l "$lock_name" '. + [$l]')
                fi
                pm_lock_release
            fi
        fi
    done

    pm_json_output "lock" "success" "{\"pruned\": $pruned, \"locks\": $pruned_list}"
}

# ============================================
# force-release 命令 - 强制释放锁 (危险操作)
# ============================================
cmd_force_release() {
    local lock_name="$1"

    pm_validate_lock_name "$lock_name" "lock" || return 1

    # 检查锁是否存在
    if ! pm_check_lock_exists "$lock_name"; then
        pm_json_output "lock" "error" "{\"error\": \"锁不存在: $lock_name\"}"
        return 1
    fi

    local lock_info
    lock_info=$(jq -c ".locks.\"$lock_name\"" "$STATE_FILE")

    # 获取文件锁
    if ! pm_lock_acquire; then
        pm_json_output "lock" "error" '{"error": "无法获取文件锁"}'
        return 1
    fi

    local now
    now=$(pm_now_iso)
    jq --arg now "$now" \
        --arg lock "$lock_name" \
        'del(.locks[$lock]) | .updated_at = $now' \
        "$STATE_FILE" > "$STATE_FILE_TMP"

    if ! mv "$STATE_FILE_TMP" "$STATE_FILE"; then
        pm_lock_release
        pm_json_output "lock" "error" '{"error": "强制释放失败"}'
        return 1
    fi

    pm_lock_release

    pm_log_warn "强制释放锁: $lock_name" >&2

    pm_json_output "lock" "success" "{\"lock_name\": \"$lock_name\", \"force_released\": true, \"previous_lock\": $lock_info}"
}

# ============================================
# 操作分发
# ============================================
ACTION="${1:-}"

case "$ACTION" in
    acquire)
        cmd_acquire "$2" "$3" "$4" "${5:-}" "${6:-$LOCK_DEFAULT_DURATION}"
        ;;
    release)
        cmd_release "$2"
        ;;
    force-release)
        cmd_force_release "$2"
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
  force-release <lock>                           强制释放锁
  list                                         列出所有锁
  check <lock>                                 检查锁状态
  prune                                        清理过期锁

环境变量:
  LOCK_DEFAULT_DURATION  默认锁超时 (默认: +6 hours)

示例:
  $(basename "$0") acquire runner dev-a CORE-01 "实现 Runner 生命周期"
  $(basename "$0") release runner
  $(basename "$0") force-release runner  # 危险操作
  $(basename "$0") list
  $(basename "$0") check runner
  $(basename "$0") prune

EOF
        exit 1
        ;;
esac
