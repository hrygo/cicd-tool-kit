#!/bin/bash
# error.sh - 增强错误处理库
# 提供结构化错误、备份恢复、事务支持

# 确保只加载一次
if [[ "${__PM_ERROR_LOADED__:-false}" == "true" ]]; then
    return 0
fi
__PM_ERROR_LOADED__=true

# 加载公共库
# shellcheck source=../lib/lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

# ============================================
# 错误代码定义 (使用 case 语句支持 bash 3.2+)
# ============================================

# 获取错误代码对应的错误名称
pm_error_code_to_name() {
    local code="$1"
    case "$code" in
        E1001) echo "INVALID_INPUT" ;;
        E1002) echo "INVALID_SPEC_ID" ;;
        E1003) echo "INVALID_DEVELOPER_ID" ;;
        E1004) echo "INVALID_LOCK_NAME" ;;
        E1005) echo "PATH_TRAVERSAL_DETECTED" ;;
        E2001) echo "SPEC_NOT_FOUND" ;;
        E2002) echo "DEVELOPER_NOT_FOUND" ;;
        E2003) echo "LOCK_NOT_FOUND" ;;
        E2004) echo "SPEC_ALREADY_ASSIGNED" ;;
        E2005) echo "DEVELOPER_BUSY" ;;
        E2006) echo "DEPENDENCY_NOT_MET" ;;
        E3001) echo "LOCK_CONFLICT" ;;
        E3002) echo "LOCK_ACQUIRE_FAILED" ;;
        E3003) echo "LOCK_EXPIRED" ;;
        E4001) echo "FILE_NOT_FOUND" ;;
        E4002) echo "FILE_WRITE_FAILED" ;;
        E4003) echo "STATE_CORRUPTED" ;;
        E5001) echo "WORKTREE_CREATE_FAILED" ;;
        E5002) echo "WORKTREE_REMOVE_FAILED" ;;
        E5003) echo "GIT_COMMAND_FAILED" ;;
        E6001) echo "FLOCK_UNAVAILABLE" ;;
        E6002) echo "JQ_NOT_FOUND" ;;
        E6003) echo "GIT_NOT_FOUND" ;;
        E7001) echo "BACKUP_FAILED" ;;
        E7002) echo "ROLLBACK_FAILED" ;;
        E7003) echo "TRANSACTION_ABORTED" ;;
        *) echo "UNKNOWN_ERROR" ;;
    esac
}

# 获取错误代码的建议
pm_error_suggestion() {
    local code="$1"
    local suggestion=""

    case "$code" in
        E1002|E1003|E1004)
            suggestion="请检查输入格式是否符合要求"
            ;;
        E2005)
            suggestion="请先完成当前任务，或使用 .pm/scripts/state.sh complete 完成任务"
            ;;
        E2006)
            suggestion="请等待依赖任务完成，或检查依赖配置是否正确"
            ;;
        E3001)
            suggestion="等待锁释放，或使用 .pm/scripts/lock.sh prune 清理过期锁"
            ;;
        E4003)
            suggestion="请使用 .pm/scripts/state.sh restore 恢复备份"
            ;;
        E6001)
            suggestion="建议安装 flock: brew install flock (Linux 通常自带)"
            ;;
        E6002)
            suggestion="请安装 jq: brew install jq"
            ;;
        E7001)
            suggestion="检查磁盘空间和文件权限"
            ;;
        *)
            suggestion="请查看错误上下文获取更多信息"
            ;;
    esac

    echo "$suggestion"
}

# ============================================
# 结构化错误输出
# ============================================

# 增强的 JSON 错误输出
# 用法: pm_error_output <action> <error_code> <error_message> [context_json]
pm_error_output() {
    local action="$1"
    local error_code="$2"
    local error_message="$3"
    local context="${4:-{}}"

    local error_name
    error_name=$(pm_error_code_to_name "$error_code")

    local suggestion
    suggestion=$(pm_error_suggestion "$error_code")

    local timestamp
    timestamp=$(pm_now_iso)

    jq -n \
        --arg a "$action" \
        --arg ec "$error_code" \
        --arg en "$error_name" \
        --arg em "$error_message" \
        --arg ctx "$context" \
        --arg sug "$suggestion" \
        --arg ts "$timestamp" \
        '{
            action: $a,
            status: "error",
            error_code: $ec,
            error_name: $en,
            error_message: $em,
            context: ($ctx | fromjson),
            suggestion: $sug,
            timestamp: $ts
        }'
}

# 简化的 JSON 成功输出
# 用法: pm_success_output <action> <data_json>
pm_success_output() {
    local action="$1"
    local data="$2"
    local timestamp
    timestamp=$(pm_now_iso)

    jq -n \
        --arg a "$action" \
        --arg d "$data" \
        --arg ts "$timestamp" \
        '{
            action: $a,
            status: "success",
            data: ($d | fromjson),
            timestamp: $ts
        }'
}

# ============================================
# 备份和恢复机制
# ============================================

# 备份目录
readonly PM_BACKUP_DIR="${STATE_FILE}.backup.d"

# 初始化备份目录
pm_backup_init() {
    if [[ ! -d "$PM_BACKUP_DIR" ]]; then
        mkdir -p "$PM_BACKUP_DIR" || {
            pm_error_output "backup_init" "E4002" "无法创建备份目录" "{}" >&2
            return 1
        }
    fi
}

# 创建状态备份
# 返回: {backup_id, path, size, created_at}
pm_state_backup() {
    pm_backup_init

    local backup_id
    backup_id="$(date +%Y%m%d-%H%M%S)-$$"
    local backup_path="$PM_BACKUP_DIR/$backup_id.json"

    # 原子复制
    if ! cp "$STATE_FILE" "$backup_path" 2>/dev/null; then
        pm_error_output "backup" "E7001" "备份创建失败" "{\"backup_id\": \"$backup_id\"}" >&2
        return 1
    fi

    # 获取备份大小
    local backup_size
    backup_size=$(wc -c < "$backup_path" 2>/dev/null || echo "0")

    # 清理旧备份（保留最近 50 个）
    pm_backup_prune 50

    jq -n \
        --arg id "$backup_id" \
        --arg path "$backup_path" \
        --arg size "$backup_size" \
        --arg ts "$(pm_now_iso)" \
        '{backup_id: $id, path: $path, size: ($size | tonumber), created_at: $ts}'
}

# 恢复状态备份
# 用法: pm_state_restore <backup_id>
pm_state_restore() {
    local backup_id="$1"
    local backup_path="$PM_BACKUP_DIR/$backup_id.json"

    if [[ ! -f "$backup_path" ]]; then
        pm_error_output "restore" "E4001" "备份文件不存在" "{\"backup_id\": \"$backup_id\"}" >&2
        return 1
    fi

    # 验证备份文件
    if ! jq '.' "$backup_path" >/dev/null 2>&1; then
        pm_error_output "restore" "E4003" "备份文件损坏" "{\"backup_id\": \"$backup_id\"}" >&2
        return 1
    fi

    # 原子恢复
    if ! pm_lock_acquire; then
        pm_error_output "restore" "E3002" "无法获取文件锁" '{}' >&2
        return 1
    fi

    if ! cp "$backup_path" "$STATE_FILE"; then
        pm_lock_release
        pm_error_output "restore" "E4002" "恢复失败" '{}' >&2
        return 1
    fi

    pm_lock_release

    jq -n \
        --arg id "$backup_id" \
        --arg ts "$(pm_now_iso)" \
        '{restored: true, backup_id: $id, timestamp: $ts}'
}

# 列出所有备份
pm_state_backup_list() {
    pm_backup_init

    if [[ ! -d "$PM_BACKUP_DIR" ]]; then
        echo '{"backups": []}'
        return 0
    fi

    local backups_json="[]"

    while IFS= read -r -d '' backup_file; do
        local backup_name
        backup_name=$(basename "$backup_file" .json)

        local backup_size
        backup_size=$(wc -c < "$backup_file" 2>/dev/null || echo "0")

        local backup_time
        # 从文件名解析时间: YYYYMMDD-HHMMSS-PID
        if [[ "$backup_name" =~ ^([0-9]{8}-[0-9]{6}) ]]; then
            backup_time="${BASH_REMATCH[1]}"
        else
            backup_time=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M:%S" "$backup_file" 2>/dev/null || stat -c "%y" "$backup_file" 2>/dev/null || echo "unknown")
        fi

        backups_json=$(echo "$backups_json" | jq \
            --arg id "$backup_name" \
            --arg size "$backup_size" \
            --arg time "$backup_time" \
            '. += [{backup_id: $id, size: ($size | tonumber), created_at: $time}]')
    done < <(find "$PM_BACKUP_DIR" -name "*.json" -print0 2>/dev/null | sort -rz)

    jq -n --argjson backups "$backups_json" '{backups: $backups}'
}

# 清理旧备份，保留最近的 N 个
pm_backup_prune() {
    local keep_count="${1:-50}"

    if [[ ! -d "$PM_BACKUP_DIR" ]]; then
        return 0
    fi

    local count=0
    find "$PM_BACKUP_DIR" -name "*.json" -type f | sort -r | while read -r backup_file; do
        ((count++))
        if [[ $count -gt $keep_count ]]; then
            rm -f "$backup_file"
        fi
    done
}

# ============================================
# 事务支持
# ============================================

# 事务上下文
_PM_TRANSACTION_ACTIVE=false
_PM_TRANSACTION_BACKUP_ID=""

# 开始事务
# 用法: pm_transaction_begin
pm_transaction_begin() {
    if [[ "$_PM_TRANSACTION_ACTIVE" == "true" ]]; then
        # 嵌套事务不支持
        return 1
    fi

    local backup_result
    backup_result=$(pm_state_backup)

    if [[ "$(echo "$backup_result" | jq -r '.status')" != "success" ]]; then
        return 1
    fi

    _PM_TRANSACTION_ACTIVE=true
    _PM_TRANSACTION_BACKUP_ID=$(echo "$backup_result" | jq -r '.backup_id')

    # 导出事务状态供子 shell 使用
    export _PM_TRANSACTION_ACTIVE _PM_TRANSACTION_BACKUP_ID
}

# 提交事务
# 用法: pm_transaction_commit
pm_transaction_commit() {
    if [[ "$_PM_TRANSACTION_ACTIVE" != "true" ]]; then
        return 1
    fi

    _PM_TRANSACTION_ACTIVE=false
    _PM_TRANSACTION_BACKUP_ID=""

    export _PM_TRANSACTION_ACTIVE _PM_TRANSACTION_BACKUP_ID
}

# 回滚事务
# 用法: pm_transaction_rollback
pm_transaction_rollback() {
    if [[ "$_PM_TRANSACTION_ACTIVE" != "true" ]]; then
        return 1
    fi

    if [[ -n "$_PM_TRANSACTION_BACKUP_ID" ]]; then
        pm_state_restore "$_PM_TRANSACTION_BACKUP_ID" >/dev/null 2>&1
    fi

    _PM_TRANSACTION_ACTIVE=false
    _PM_TRANSACTION_BACKUP_ID=""

    export _PM_TRANSACTION_ACTIVE _PM_TRANSACTION_BACKUP_ID
}

# 检查事务是否活跃
pm_transaction_active() {
    [[ "$_PM_TRANSACTION_ACTIVE" == "true" ]]
}

# 事务包装器
# 用法: pm_transaction <command> [args...]
pm_transaction() {
    pm_transaction_begin || return 1

    local exit_code=0

    # 执行命令
    "$@" || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        pm_transaction_commit
    else
        pm_transaction_rollback
    fi

    return $exit_code
}

# ============================================
# 事务安全的状态更新
# ============================================

# 带事务的 JSON 更新
# 用法: pm_safe_update <jq_filter> <error_code>
pm_safe_update() {
    local jq_filter="$1"
    local error_code="${2:-E4002}"

    if ! pm_lock_acquire; then
        pm_error_output "update" "E3002" "无法获取文件锁" '{}' >&2
        return 1
    fi

    local now
    now=$(pm_now_iso)

    # 执行更新
    local result
    result=$(jq --arg now "$now" "$jq_filter | .updated_at = \$now" "$STATE_FILE" 2>&1)
    if [[ $? -ne 0 ]]; then
        pm_lock_release
        pm_error_output "update" "$error_code" "jq 操作失败: $result" '{}' >&2
        return 1
    fi

    # 写入临时文件
    echo "$result" > "$STATE_FILE_TMP"

    # 原子移动
    if ! mv "$STATE_FILE_TMP" "$STATE_FILE"; then
        pm_lock_release
        pm_error_output "update" "$error_code" "写入 state.json 失败" '{}' >&2
        return 1
    fi

    pm_lock_release
    return 0
}

# ============================================
# 导出变量和函数
# ============================================
export PM_BACKUP_DIR
export _PM_TRANSACTION_ACTIVE _PM_TRANSACTION_BACKUP_ID
