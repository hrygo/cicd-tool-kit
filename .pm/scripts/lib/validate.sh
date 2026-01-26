#!/bin/bash
# validate.sh - 输入验证函数库
# 提供统一的输入验证功能

# 确保只加载一次
if [[ "${__PM_VALIDATE_LOADED__:-false}" == "true" ]]; then
    return 0
fi
__PM_VALIDATE_LOADED__=true

# 加载公共库
# shellcheck source=../lib/lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

# ============================================
# 错误输出辅助函数
# ============================================
_validate_error() {
    local action="$1"
    local message="$2"
    local output
    output=$(jq -n --arg msg "$message" '{error: $msg}')
    pm_json_output "$action" "error" "$output"
    return 1
}

# ============================================
# 格式验证正则表达式
# ============================================
readonly PM_REGEX_LOCK_NAME='^[a-z_]+$'
readonly PM_REGEX_DEVELOPER_ID='^dev-[a-z]$'
readonly PM_REGEX_SPEC_ID='^[A-Z]+-[0-9]+$'

# ============================================
# 验证函数
# ============================================

# 验证 lock_name 格式
# 只允许小写字母和下划线
pm_validate_lock_name() {
    local lock_name="$1"
    local action="${2:-lock}"

    if [[ -z "$lock_name" ]]; then
        _validate_error "$action" 'lock_name 不能为空'
        return 1
    fi

    if [[ ! "$lock_name" =~ $PM_REGEX_LOCK_NAME ]]; then
        _validate_error "$action" "无效的 lock_name 格式: $lock_name (只允许小写字母和下划线)"
        return 1
    fi

    return 0
}

# 验证 developer_id 格式
# 格式: dev-a, dev-b, dev-c
pm_validate_developer_id() {
    local dev_id="$1"
    local action="${2:-unknown}"

    if [[ -z "$dev_id" ]]; then
        _validate_error "$action" 'developer_id 不能为空'
        return 1
    fi

    if [[ ! "$dev_id" =~ $PM_REGEX_DEVELOPER_ID ]]; then
        _validate_error "$action" "无效的 developer_id 格式: $dev_id (应为 dev-a 格式)"
        return 1
    fi

    return 0
}

# 验证 spec_id 格式
# 格式: CORE-01, SKILL-01, etc.
pm_validate_spec_id() {
    local spec_id="$1"
    local action="${2:-unknown}"

    if [[ -z "$spec_id" ]]; then
        _validate_error "$action" 'spec_id 不能为空'
        return 1
    fi

    if [[ ! "$spec_id" =~ $PM_REGEX_SPEC_ID ]]; then
        _validate_error "$action" "无效的 spec_id 格式: $spec_id (应为 CORE-01 格式)"
        return 1
    fi

    return 0
}

# 验证 spec_id 不包含路径穿越字符
# 防止 ../../ 等攻击
pm_validate_safe_spec_id() {
    local spec_id="$1"
    local action="${2:-unknown}"

    if [[ "$spec_id" =~ \.\. ]] || [[ "$spec_id" =~ / ]] || [[ "$spec_id" =~ \\ ]]; then
        _validate_error "$action" "spec_id 包含非法字符: $spec_id"
        return 1
    fi

    return 0
}

# 验证状态值是否合法
pm_validate_status() {
    local status="$1"
    local action="${2:-unknown}"

    local valid_statuses=("pending" "ready" "assigned" "in_progress" "completed" "blocked")
    local valid=false

    for s in "${valid_statuses[@]}"; do
        if [[ "$status" == "$s" ]]; then
            valid=true
            break
        fi
    done

    if [[ "$valid" == "false" ]]; then
        _validate_error "$action" "无效的 status: $status (允许值: ${valid_statuses[*]})"
        return 1
    fi

    return 0
}

# 验证优先级值是否合法
pm_validate_priority() {
    local priority="$1"
    local action="${2:-unknown}"

    local valid_priorities=("p0" "p1" "p2")
    local valid=false

    for p in "${valid_priorities[@]}"; do
        if [[ "$priority" == "$p" ]]; then
            valid=true
            break
        fi
    done

    if [[ "$valid" == "false" ]]; then
        _validate_error "$action" "无效的 priority: $priority (允许值: ${valid_priorities[*]})"
        return 1
    fi

    return 0
}

# ============================================
# 存在性检查函数
# ============================================

# 检查开发者是否存在于 state.json
pm_check_developer_exists() {
    local dev_id="$1"
    local action="${2:-unknown}"

    if ! jq -e ".developers.\"$dev_id\"" "$STATE_FILE" >/dev/null 2>&1; then
        _validate_error "$action" "开发者不存在: $dev_id"
        return 1
    fi

    return 0
}

# 检查 Spec 是否存在于 state.json
pm_check_spec_exists() {
    local spec_id="$1"
    local action="${2:-unknown}"

    if ! jq -e ".specs.\"$spec_id\"" "$STATE_FILE" >/dev/null 2>&1; then
        _validate_error "$action" "Spec 不存在: $spec_id"
        return 1
    fi

    return 0
}

# 检查锁是否已存在
pm_check_lock_exists() {
    local lock_name="$1"

    jq -e ".locks.\"$lock_name\"" "$STATE_FILE" >/dev/null 2>&1
    return $?
}

# ============================================
# 导出所有验证函数供其他脚本使用
# ============================================
export PM_REGEX_LOCK_NAME PM_REGEX_DEVELOPER_ID PM_REGEX_SPEC_ID
