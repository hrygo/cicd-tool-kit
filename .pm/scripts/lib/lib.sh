#!/bin/bash
# lib.sh - 公共函数库 v3.1
# 提供 JSON 写入、输出、事务等通用功能

# 确保只加载一次
if [[ "${__PM_LIB_LOADED__:-false}" == "true" ]]; then
    return 0
fi
__PM_LIB_LOADED__=true

# ============================================
# 版本信息
# ============================================
readonly PM_LIB_VERSION="3.1.0"

# ============================================
# 路径配置
# ============================================
pm_init_paths() {
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
    REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
    STATE_FILE="${STATE_FILE:-$REPO_ROOT/.pm/state.json}"
    STATE_FILE_TMP="${STATE_FILE}.tmp"
    STATE_FILE_LOCK="${STATE_FILE}.lock"
    PM_BACKUP_DIR="${STATE_FILE}.backup.d"

    export SCRIPT_DIR REPO_ROOT STATE_FILE STATE_FILE_TMP STATE_FILE_LOCK PM_BACKUP_DIR
}

# 初始化路径
pm_init_paths

# ============================================
# 日志功能
# ============================================
# 日志级别: DEBUG=0, INFO=1, WARN=2, ERROR=3
PM_LOG_LEVEL="${PM_LOG_LEVEL:-1}"

pm_log() {
    local level="$1"
    local level_num="$2"
    shift 2

    if [[ $level_num -ge $PM_LOG_LEVEL ]]; then
        echo "[$(date +'%Y-%m-%d %H:%M:%S')] [$level]" "$*" >&2
    fi
}

pm_log_debug() { pm_log "DEBUG" 0 "$@"; }
pm_log_info() { pm_log "INFO" 1 "$@"; }
pm_log_warn() { pm_log "WARN" 2 "$@"; }
pm_log_error() { pm_log "ERROR" 3 "$@"; }

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
# 用法: pm_lock_acquire [timeout]
pm_lock_acquire() {
    local timeout="${1:-30}"  # 默认 30 秒超时
    local warning_shown=false

    # 确保锁文件存在
    if [[ ! -f "$STATE_FILE_LOCK" ]]; then
        touch "$STATE_FILE_LOCK" 2>/dev/null || {
            pm_log_error "无法创建锁文件: $STATE_FILE_LOCK"
            return 1
        }
    fi

    if [[ "$PM_HAS_FLOCK" == "true" ]]; then
        # 尝试获取锁 (使用 flock 的超时机制)
        local start_time
        start_time=$(date +%s)

        while true; do
            if flock -w 1 -x 9 2>/dev/null; then
                # 成功获取锁
                return 0
            fi

            local current_time
            current_time=$(date +%s)
            local elapsed=$((current_time - start_time))

            if [[ $elapsed -ge $timeout ]]; then
                pm_log_error "获取文件锁超时 (${timeout}s)"
                pm_log_error "可能有其他进程正在操作，或锁文件残留"
                pm_log_error "尝试清理: flock -u 9 或删除 $STATE_FILE_LOCK"
                return 1
            fi

            # 每 5 秒显示一次警告
            if [[ $warning_shown == false && $elapsed -ge 5 ]]; then
                pm_log_warn "等待文件锁中... (${elapsed}s/${timeout}s)"
                warning_shown=true
            fi

            sleep 0.5
        done
    else
        # 没有 flock，发出警告但仍继续
        if [[ ! "$PM_FLOCK_WARN_SHOWN" == "true" ]]; then
            pm_log_warn "flock 未安装，文件锁保护被禁用"
            pm_log_warn "建议安装: brew install flock"
            export PM_FLOCK_WARN_SHOWN=true
        fi
    fi

    return 0
}

# 释放锁
pm_lock_release() {
    if [[ "$PM_HAS_FLOCK" == "true" ]]; then
        flock -u 9 2>/dev/null || true
    fi
}

# 强制释放锁 (危险操作，仅用于故障恢复)
pm_lock_force_release() {
    pm_log_warn "强制释放文件锁"
    rm -f "$STATE_FILE_LOCK"
    touch "$STATE_FILE_LOCK"
}

# ============================================
# 原子 JSON 写入 (带锁保护)
# ============================================
# 用法: pm_json_write '.locks[$key] = value'
pm_json_write() {
    local jq_filter="$1"
    local now
    now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # 获取锁
    if ! pm_lock_acquire; then
        echo '{"status": "error", "error": "无法获取文件锁"}' >&2
        return 1
    fi

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
# 增强的 JSON 输出 (带错误码和建议)
# ============================================
# 用法: pm_json_error "action" "E3001" "锁已被占用" '{"lock": "runner"}'
pm_json_error() {
    local action="$1"
    local error_code="$2"
    local error_message="$3"
    local context="${4:-{}}"

    # 如果已加载 error.sh，使用增强版本
    if declare -f pm_error_output >/dev/null 2>&1; then
        pm_error_output "$action" "$error_code" "$error_message" "$context"
    else
        # 简化版本
        pm_json_output "$action" "error" "{\"error_code\": $error_code, \"error_message\": \"$error_message\", \"context\": $context}"
    fi
}

# ============================================
# 时间戳处理
# ============================================
# 获取当前 Unix 时间戳（秒）
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
    elif ts=$(date -j -f "%Y-%m-%dT%H:%M:%S%z" "$iso_time" +%s 2>/dev/null); then
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
        # macOS (BSD date)
        case "$offset" in
            *hour*|*hours*)
                local num="${offset%% *}"
                date -u -v+${num}H +"%Y-%m-%dT%H:%M:%SZ"
                ;;
            *day*|*days*)
                local num="${offset%% *}"
                date -u -v+${num}d +"%Y-%m-%dT%H:%M:%SZ"
                ;;
            *week*|*weeks*)
                local num="${offset%% *}"
                date -u -v+${num}w +"%Y-%m-%dT%H:%M:%SZ"
                ;;
            *)
                date -u +"%Y-%m-%dT%H:%M:%SZ"
                ;;
        esac
    else
        # Linux (GNU date)
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
if [[ -f "$STATE_FILE_LOCK" ]] || touch "$STATE_FILE_LOCK" 2>/dev/null; then
    exec 9>"$STATE_FILE_LOCK"
fi

# ============================================
# 备份支持 (轻量级版本)
# ============================================
pm_create_backup() {
    mkdir -p "$PM_BACKUP_DIR" 2>/dev/null || return 1

    local backup_id
    backup_id="$(date +%Y%m%d-%H%M%S)-$$"
    local backup_path="$PM_BACKUP_DIR/$backup_id.json"

    cp "$STATE_FILE" "$backup_path" 2>/dev/null || return 1

    # 清理旧备份（保留最近 20 个）
    ls -t "$PM_BACKUP_DIR"/*.json 2>/dev/null | tail -n +21 | xargs rm -f 2>/dev/null || true

    echo "$backup_id"
}

pm_restore_backup() {
    local backup_id="$1"
    local backup_path="$PM_BACKUP_DIR/$backup_id.json"

    [[ -f "$backup_path" ]] || return 1
    jq '.' "$backup_path" >/dev/null 2>&1 || return 1

    cp "$backup_path" "$STATE_FILE" || return 1
    return 0
}

pm_list_backups() {
    if [[ ! -d "$PM_BACKUP_DIR" ]]; then
        echo "[]"
        return 0
    fi

    local backups_json="[]"

    for backup_file in "$PM_BACKUP_DIR"/*.json; do
        [[ -f "$backup_file" ]] || continue

        local backup_name
        backup_name=$(basename "$backup_file" .json)

        local backup_size
        backup_size=$(wc -c < "$backup_file" 2>/dev/null || echo "0")

        backups_json=$(echo "$backups_json" | jq \
            --arg id "$backup_name" \
            --arg size "$backup_size" \
            '. += [{backup_id: $id, size: ($size | tonumber)}]')
    done

    jq -n --argjson backups "$backups_json" '{backups: $backups}'
}

# ============================================
# 动态获取开发者列表 (避免硬编码)
# ============================================
pm_get_developers() {
    jq -r '.developers | keys[]' "$STATE_FILE" 2>/dev/null || echo ""
}

pm_get_spec_ids() {
    jq -r '.specs | keys[]' "$STATE_FILE" 2>/dev/null || echo ""
}

pm_get_lock_names() {
    jq -r '.locks | keys[]' "$STATE_FILE" 2>/dev/null || echo ""
}

# ============================================
# 状态查询辅助函数
# ============================================
# 检查 Spec 是否存在
pm_spec_exists() {
    local spec_id="$1"
    jq -e ".specs.\"$spec_id\"" "$STATE_FILE" >/dev/null 2>&1
}

# 检查开发者是否存在
pm_developer_exists() {
    local dev_id="$1"
    jq -e ".developers.\"$dev_id\"" "$STATE_FILE" >/dev/null 2>&1
}

# 检查锁是否存在
pm_lock_exists() {
    local lock_name="$1"
    jq -e ".locks.\"$lock_name\"" "$STATE_FILE" >/dev/null 2>&1
}

# 获取 Spec 状态
pm_get_spec_status() {
    local spec_id="$1"
    jq -r ".specs.\"$spec_id\".status // \"unknown\"" "$STATE_FILE" 2>/dev/null
}

# 获取开发者当前任务
pm_get_developer_task() {
    local dev_id="$1"
    jq -r ".developers.\"$dev_id\".current_task // \"null\"" "$STATE_FILE" 2>/dev/null
}

# 获取 Spec 的依赖列表
pm_get_spec_dependencies() {
    local spec_id="$1"
    jq -r ".specs.\"$spec_id\".dependencies[] // empty" "$STATE_FILE" 2>/dev/null
}

# 检查 Spec 的依赖是否都已完成
pm_check_dependencies() {
    local spec_id="$1"
    local dep

    while read -r dep; do
        [[ -z "$dep" ]] && continue
        local dep_status
        dep_status=$(pm_get_spec_status "$dep")
        if [[ "$dep_status" != "completed" ]]; then
            echo "$dep"
            return 1
        fi
    done < <(pm_get_spec_dependencies "$spec_id")

    return 0
}

# ============================================
# 环境检查
# ============================================
pm_check_dependencies() {
    local missing=()

    command -v jq >/dev/null 2>&1 || missing+=("jq")
    command -v git >/dev/null 2>&1 || missing+=("git")

    if [[ ${#missing[@]} -gt 0 ]]; then
        echo "缺少依赖: ${missing[*]}"
        return 1
    fi

    return 0
}

# ============================================
# 导出
# ============================================
export PM_LIB_VERSION PM_LOG_LEVEL
export PM_HAS_FLOCK PM_FLOCK_WARN_SHOWN
