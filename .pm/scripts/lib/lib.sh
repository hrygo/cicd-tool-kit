#!/bin/bash
# lib.sh - 公共函数库
# 提供 JSON 写入、输出等通用功能

# 确保只加载一次
if [[ "${__PM_LIB_LOADED__:-false}" == "true" ]]; then
    return 0
fi
__PM_LIB_LOADED__=true

# ============================================
# 路径配置
# ============================================
pm_init_paths() {
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
    REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
    STATE_FILE="${STATE_FILE:-$REPO_ROOT/.pm/state.json}"
    STATE_FILE_TMP="${STATE_FILE}.tmp"
    STATE_FILE_LOCK="${STATE_FILE}.lock"

    export SCRIPT_DIR REPO_ROOT STATE_FILE STATE_FILE_TMP STATE_FILE_LOCK
}

# 初始化路径
pm_init_paths

# ============================================
# 文件锁保护 (解决竞态条件)
# ============================================
# 使用 fd 9 作为锁文件描述符，避免与标准流冲突
# 使用超时机制避免死锁

# 检测是否有 flock 命令
PM_HAS_FLOCK=false
if command -v flock >/dev/null 2>&1; then
    PM_HAS_FLOCK=true
fi

# 获取排他锁
# 用法: pm_lock_acquire || exit 1
pm_lock_acquire() {
    local timeout="${1:-30}"  # 默认 30 秒超时

    # 确保锁文件存在
    touch "$STATE_FILE_LOCK"

    if [[ "$PM_HAS_FLOCK" == "true" ]]; then
        # 尝试获取锁 (使用 flock 的超时机制)
        flock -w "$timeout" 9 || {
            echo '{"status": "error", "error": "无法获取文件锁，可能被其他进程持有"}' >&2
            return 1
        }
    fi
    # 如果没有 flock，跳过锁（单用户本地开发环境可接受）

    return 0
}

# 释放锁
pm_lock_release() {
    if [[ "$PM_HAS_FLOCK" == "true" ]]; then
        flock -u 9 2>/dev/null || true
    fi
}

# ============================================
# 原子 JSON 写入 (带锁保护)
# ============================================
# 用法: pm_json_write '.locks[$key] = value' "jq 过滤器"
pm_json_write() {
    local jq_filter="$1"
    local now
    now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # 获取锁
    pm_lock_acquire || return 1

    # 执行 jq 操作
    local result
    result=$(jq --arg now "$now" "$jq_filter | .updated_at = \$now" "$STATE_FILE" 2>&1)
    if [[ $? -ne 0 ]]; then
        pm_lock_release
        echo "{\"status\": \"error\", \"error\": \"jq 操作失败: $result\"}" >&2
        return 1
    fi

    # 原子写入
    echo "$result" > "$STATE_FILE_TMP"
    if ! mv "$STATE_FILE_TMP" "$STATE_FILE"; then
        pm_lock_release
        echo '{"status": "error", "error": "写入 state.json 失败"}' >&2
        return 1
    fi

    pm_lock_release
    return 0
}

# ============================================
# JSON 输出 (使用 jq 生成，避免硬编码)
# ============================================
# 用法: pm_json_output "action" "status" '{"key": "value"}'
pm_json_output() {
    local action="$1"
    local status="$2"
    local data="${3:-{}}"
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    jq -n \
        --arg a "$action" \
        --arg s "$status" \
        --arg ts "$timestamp" \
        --argjson d "$data" \
        '{action: $a, status: $s, data: $d, timestamp: $ts}'
}

# ============================================
# 时间戳处理
# ============================================
# 获取当前 UTC 时间戳（秒）
pm_timestamp_now() {
    date +%s
}

# 将 ISO 时间转换为 Unix 时间戳
# 如果解析失败，返回 -1（而不是 0，避免与 1970-01-01 混淆）
pm_timestamp_parse() {
    local iso_time="$1"

    local ts
    if ts=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$iso_time" +%s 2>/dev/null); then
        echo "$ts"
    elif ts=$(date -d "$iso_time" +%s 2>/dev/null); then
        echo "$ts"
    else
        echo "-1"
    fi
}

# 获取偏移后的时间 (ISO 格式)
# 用法: pm_date_offset "+6 hours" 或 "+2 days"
pm_date_offset() {
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

# 获取当前 UTC 时间 (ISO 格式)
pm_now_iso() {
    date -u +"%Y-%m-%dT%H:%M:%SZ"
}

# ============================================
# 临时文件清理
# ============================================
pm_cleanup() {
    rm -f "$STATE_FILE_TMP" 2>/dev/null || true
}

# 设置清理陷阱
trap pm_cleanup EXIT

# 在脚本结束时使用 flock 保持锁
# 将锁文件描述符重定向到锁文件
exec 9>"$STATE_FILE_LOCK"
