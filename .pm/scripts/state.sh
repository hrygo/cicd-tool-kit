#!/bin/bash
# state.sh - 结构化状态操作脚本 v3.1
# 只读写 JSON，由 AI Agent 调用

set -euo pipefail

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 加载公共库
# shellcheck source=lib/lib.sh
# shellcheck source=lib/validate.sh
# shellcheck source=lib/error.sh
source "$SCRIPT_DIR/lib/lib.sh"
source "$SCRIPT_DIR/lib/validate.sh"
source "$SCRIPT_DIR/lib/error.sh" 2>/dev/null || true

# ============================================
# 确保依赖
# ============================================
if ! command -v jq >/dev/null 2>&1; then
    pm_json_error "state" "E6002" "需要 jq" '{"install": "brew install jq"}' >&2
    exit 1
fi

# 确保 state.json 存在
if [[ ! -f "$STATE_FILE" ]]; then
    pm_json_error "state" "E4001" "状态文件不存在" "{\"path\": \"$STATE_FILE\"}" >&2
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
        pm_json_error "state" "E3002" "无法获取文件锁" '{}' >&2
        return 1
    fi

    local now
    now=$(pm_now_iso)
    jq --arg now "$now" "$path = $value | .updated_at = \$now" "$STATE_FILE" > "$STATE_FILE_TMP"

    if ! mv "$STATE_FILE_TMP" "$STATE_FILE"; then
        pm_lock_release
        pm_json_error "state" "E4002" "更新 state.json 失败" '{}' >&2
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
        pm_json_error "state" "E3002" "无法获取文件锁" '{}' >&2
        return 1
    fi

    local now
    now=$(pm_now_iso)
    echo "$input" | jq --arg now "$now" '. * $in | .updated_at = $now' > "$STATE_FILE_TMP"

    if ! mv "$STATE_FILE_TMP" "$STATE_FILE"; then
        pm_lock_release
        pm_json_error "state" "E4002" "更新 state.json 失败" '{}' >&2
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
    local auto_backup="${3:-true}"  # 默认自动备份

    # 验证输入
    pm_validate_spec_id "$spec_id" "state" || return 1
    pm_validate_developer_id "$dev_id" "state" || return 1
    pm_check_developer_exists "$dev_id" "state" || return 1
    pm_check_spec_exists "$spec_id" "state" || return 1

    # 创建备份（用于回滚）
    local backup_id=""
    if [[ "$auto_backup" == "true" ]]; then
        backup_id=$(pm_create_backup 2>/dev/null || echo "")
    fi

    # 获取锁
    if ! pm_lock_acquire; then
        pm_json_error "assign" "E3002" "无法获取文件锁" '{}' >&2
        return 1
    fi

    local now
    now=$(pm_now_iso)

    # 1. 检查开发者是否有进行中任务
    local current_task
    current_task=$(pm_get_developer_task "$dev_id")
    if [[ "$current_task" != "null" && -n "$current_task" ]]; then
        pm_lock_release
        pm_json_error "assign" "E2005" "开发者有进行中任务" "{\"dev_id\": \"$dev_id\", \"current_task\": \"$current_task\"}" >&2
        return 1
    fi

    # 2. 检查 Spec 依赖（Scripts 只做简单检查，详细分析由 Agent 负责）
    local blocking_dep=""
    while read -r dep; do
        [[ -z "$dep" ]] && continue
        local dep_status
        dep_status=$(pm_get_spec_status "$dep")
        if [[ "$dep_status" != "completed" ]]; then
            blocking_dep="$dep"
            break
        fi
    done < <(pm_get_spec_dependencies "$spec_id")

    if [[ -n "$blocking_dep" ]]; then
        pm_lock_release
        pm_json_error "assign" "E2006" "依赖未满足" "{\"spec_id\": \"$spec_id\", \"blocking\": \"$blocking_dep\"}" >&2
        return 1
    fi

    # 3. 更新状态
    if ! jq --arg now "$now" \
        --arg spec "$spec_id" \
        --arg dev "$dev_id" \
        '
        .specs[$spec].status = "in_progress" |
        .specs[$spec].assignee = $dev |
        .specs[$spec].started_at = $now |
        .developers[$dev].current_task = $spec |
        .updated_at = $now
        ' "$STATE_FILE" > "$STATE_FILE_TMP"; then
        pm_lock_release
        pm_json_error "assign" "E4003" "状态更新失败 (jq 错误)" '{}' >&2
        # 尝试回滚
        [[ -n "$backup_id" ]] && pm_restore_backup "$backup_id" 2>/dev/null || true
        return 1
    fi

    if ! mv "$STATE_FILE_TMP" "$STATE_FILE"; then
        pm_lock_release
        pm_json_error "assign" "E4002" "写入 state.json 失败" "{\"backup_id\": \"$backup_id\"}" >&2
        # 尝试回滚
        [[ -n "$backup_id" ]] && pm_restore_backup "$backup_id" 2>/dev/null || true
        return 1
    fi

    pm_lock_release

    # 4. 输出结果
    local output
    output=$(jq -n \
        --arg a "assign" \
        --arg s "success" \
        --arg ts "$(pm_now_iso)" \
        --argjson spec "$(jq -c ".specs.\"$spec_id\"" "$STATE_FILE")" \
        --argjson dev "$(jq -c ".developers.\"$dev_id\"" "$STATE_FILE")" \
        --arg bid "$backup_id" \
        '{action: $a, status: $s, data: {spec: $spec, developer: $dev, backup_id: $bid}, timestamp: $ts}')
    echo "$output"
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

    # 创建备份
    local backup_id
    backup_id=$(pm_create_backup 2>/dev/null || echo "")

    # 获取锁
    if ! pm_lock_acquire; then
        pm_json_error "complete" "E3002" "无法获取文件锁" '{}' >&2
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
        .specs[$spec].assignee = null |
        .developers[$dev].current_task = null |
        .developers[$dev].completed_specs += [$spec] |
        .updated_at = $now
        ' "$STATE_FILE" > "$STATE_FILE_TMP"

    if ! mv "$STATE_FILE_TMP" "$STATE_FILE"; then
        pm_lock_release
        pm_json_error "complete" "E4002" "写入 state.json 失败" "{\"backup_id\": \"$backup_id\"}" >&2
        [[ -n "$backup_id" ]] && pm_restore_backup "$backup_id" 2>/dev/null || true
        return 1
    fi

    pm_lock_release

    local output
    output=$(jq -n \
        --arg a "complete" \
        --arg s "success" \
        --arg ts "$(pm_now_iso)" \
        --argjson spec "$(jq -c ".specs.\"$spec_id\"" "$STATE_FILE")" \
        --argjson dev "$(jq -c ".developers.\"$dev_id\"" "$STATE_FILE")" \
        --arg bid "$backup_id" \
        '{action: $a, status: $s, data: {spec: $spec, developer: $dev, backup_id: $bid}, timestamp: $ts}')
    echo "$output"
}

# ============================================
# unassign 命令 - 取消分配 (新增)
# ============================================
cmd_unassign() {
    local spec_id="$1"
    local dev_id="$2"

    pm_validate_spec_id "$spec_id" "state" || return 1
    pm_validate_developer_id "$dev_id" "state" || return 1

    local backup_id
    backup_id=$(pm_create_backup 2>/dev/null || echo "")

    if ! pm_lock_acquire; then
        pm_json_error "unassign" "E3002" "无法获取文件锁" '{}' >&2
        return 1
    fi

    local now
    now=$(pm_now_iso)

    jq --arg now "$now" \
        --arg spec "$spec_id" \
        --arg dev "$dev_id" \
        '
        .specs[$spec].status = "ready" |
        .specs[$spec].assignee = null |
        .specs[$spec].started_at = null |
        .developers[$dev].current_task = null |
        .updated_at = $now
        ' "$STATE_FILE" > "$STATE_FILE_TMP"

    if ! mv "$STATE_FILE_TMP" "$STATE_FILE"; then
        pm_lock_release
        pm_json_error "unassign" "E4002" "写入失败" "{\"backup_id\": \"$backup_id\"}" >&2
        [[ -n "$backup_id" ]] && pm_restore_backup "$backup_id" 2>/dev/null || true
        return 1
    fi

    pm_lock_release

    pm_success_output "unassign" "{\"spec_id\": \"$spec_id\", \"developer_id\": \"$dev_id\", \"backup_id\": \"$backup_id\"}"
}

# ============================================
# progress 命令 - 计算进度统计 (单次 jq 优化)
# ============================================
cmd_progress() {
    # 单次 jq 调用完成所有计算
    jq \
        --arg a "progress" \
        --arg s "success" \
        --arg ts "$(pm_now_iso)" \
        '
        {
            action: $a,
            status: $s,
            data: {
                summary: {
                    total_specs: (.specs | length),
                    completed: ([.specs | to_entries[] | select(.value.status == "completed")] | length),
                    in_progress: ([.specs | to_entries[] | select(.value.status == "in_progress")] | length),
                    ready: ([.specs | to_entries[] | select(.value.status == "ready")] | length),
                    blocked: ([.specs | to_entries[] | select(.value.status == "blocked")] | length),
                    pending: ([.specs | to_entries[] | select(.value.status == "pending")] | length),
                    total_progress: (
                        if (.specs | length) > 0
                        then "\((([.specs | to_entries[] | select(.value.status == "completed")] | length) * 100 / (.specs | length)) | tostring)%"
                        else "0%"
                        end
                    )
                },
                developers: [
                    .developers | to_entries[] | {
                        id: .key,
                        name: .value.name,
                        current_task: .value.current_task,
                        completed_count: (.value.completed_specs | length)
                    }
                ],
                active_locks: [
                    .locks | to_entries[] | {
                        name: .key,
                        locked_by: .value.locked_by,
                        spec_id: .value.spec_id
                    }
                ],
                active_worktrees_count: (.worktrees | length)
            },
            timestamp: $ts
        }
        ' "$STATE_FILE"
}

# ============================================
# validate 命令 - 验证状态文件
# ============================================
cmd_validate() {
    local schema="$REPO_ROOT/.pm/state.schema.json"
    local errors_json="[]"

    # 1. JSON 语法验证
    if ! jq '.' "$STATE_FILE" >/dev/null 2>&1; then
        errors_json=$(echo "$errors_json" | jq '. + [{code: "E4003", message: "state.json 不是有效的 JSON"}]')
    fi

    # 2. Schema 验证 (如果可用)
    if [[ -f "$schema" ]]; then
        if command -v ajv >/dev/null 2>&1; then
            if ! ajv test -s "$schema" -d "$STATE_FILE" --valid 2>&1 | grep -q "true"; then
                errors_json=$(echo "$errors_json" | jq '. + [{code: "E4003", message: "JSON Schema 验证失败"}]')
            fi
        elif command -v check-jsonschema >/dev/null 2>&1; then
            if ! check-jsonschema "$STATE_FILE" "$schema" 2>&1 | grep -q "PASS"; then
                errors_json=$(echo "$errors_json" | jq '. + [{code: "E4003", message: "JSON Schema 验证失败"}]')
            fi
        fi
    fi

    # 3. 业务逻辑验证
    # 检查：每个开发者是否有效
    while read -r dev_id; do
        [[ -z "$dev_id" ]] && continue
        if ! pm_validate_developer_id "$dev_id" "validate" 2>/dev/null; then
            errors_json=$(echo "$errors_json" | jq --arg id "$dev_id" '. + [{code: "E1003", message: "无效的开发者 ID", context: {developer_id: $id}}]')
        fi
    done < <(jq -r '.developers | keys[]' "$STATE_FILE" 2>/dev/null)

    # 检查：每个 Spec 是否有效
    while read -r spec_id; do
        [[ -z "$spec_id" ]] && continue
        if ! pm_validate_spec_id "$spec_id" "validate" 2>/dev/null; then
            errors_json=$(echo "$errors_json" | jq --arg id "$spec_id" '. + [{code: "E1002", message: "无效的 Spec ID", context: {spec_id: $id}}]')
        fi
    done < <(jq -r '.specs | keys[]' "$STATE_FILE" 2>/dev/null)

    # 输出结果
    local error_count
    error_count=$(echo "$errors_json" | jq 'length')

    if [[ $error_count -eq 0 ]]; then
        pm_success_output "validate" '{"valid": true, "errors": []}'
    else
        jq -n \
            --arg a "validate" \
            --arg s "error" \
            --arg ts "$(pm_now_iso)" \
            --argjson errors "$errors_json" \
            '{action: $a, status: $s, data: {valid: false, error_count: ($errors | length), errors: $errors}, timestamp: $ts}'
    fi
}

# ============================================
# backup 命令 - 创建备份
# ============================================
cmd_backup() {
    local backup_id
    backup_id=$(pm_create_backup)

    if [[ -z "$backup_id" ]]; then
        pm_json_error "backup" "E7001" "备份创建失败" '{}' >&2
        return 1
    fi

    local backup_path="$PM_BACKUP_DIR/$backup_id.json"
    local backup_size
    backup_size=$(wc -c < "$backup_path" 2>/dev/null || echo "0")

    pm_success_output "backup" "{\"backup_id\": \"$backup_id\", \"path\": \"$backup_path\", \"size\": $backup_size}"
}

# ============================================
# restore 命令 - 恢复备份
# ============================================
cmd_restore() {
    local backup_id="$1"

    if [[ -z "$backup_id" ]]; then
        pm_json_error "restore" "E1001" "需要指定 backup_id" '{}' >&2
        return 1
    fi

    if ! pm_restore_backup "$backup_id"; then
        pm_json_error "restore" "E4001" "恢复失败" "{\"backup_id\": \"$backup_id\"}" >&2
        return 1
    fi

    pm_success_output "restore" "{\"backup_id\": \"$backup_id\", \"restored\": true}"
}

# ============================================
# list-backups 命令 - 列出所有备份
# ============================================
cmd_list_backups() {
    pm_list_backups
}

# ============================================
# rollback 命令 - 回滚到指定备份
# ============================================
cmd_rollback() {
    local backup_id="$1"

    if [[ -z "$backup_id" ]]; then
        # 没有指定 ID，回滚到最新的备份
        local latest
        latest=$(ls -t "$PM_BACKUP_DIR"/*.json 2>/dev/null | head -1)
        if [[ -z "$latest" ]]; then
            pm_json_error "rollback" "E4001" "没有可用的备份" '{}' >&2
            return 1
        fi
        backup_id=$(basename "$latest" .json)
    fi

    cmd_restore "$backup_id"
}

# ============================================
# rollback 命令 - 回滚到指定备份
# ============================================
cmd_rollback() {
    local backup_id="$1"

    if [[ -z "$backup_id" ]]; then
        # 没有指定 ID，回滚到最新的备份
        local latest
        latest=$(ls -t "$PM_BACKUP_DIR"/*.json 2>/dev/null | head -1)
        if [[ -z "$latest" ]]; then
            pm_json_error "rollback" "E4001" "没有可用的备份" '{}' >&2
            return 1
        fi
        backup_id=$(basename "$latest" .json)
    fi

    cmd_restore "$backup_id"
}

# ============================================
# health 命令 - 健康检查
# ============================================
cmd_health() {
    local health_status="healthy"
    local checks="[]"

    # 1. 检查 state.json 文件
    if [[ ! -f "$STATE_FILE" ]]; then
        health_status="unhealthy"
        checks=$(echo "$checks" | jq --arg c "state_file" --arg s "critical" --arg m "状态文件不存在" '. += [{check: $c, status: $s, message: $m}]')
    else
        # 2. 验证 JSON 格式
        if ! jq '.' "$STATE_FILE" >/dev/null 2>&1; then
            health_status="unhealthy"
            checks=$(echo "$checks" | jq --arg c "json_valid" --arg s "critical" --arg m "JSON 格式损坏" '. += [{check: $c, status: $s, message: $m}]')
        fi
    fi

    # 3. 检查备份目录
    local backup_count=0
    if [[ -d "$PM_BACKUP_DIR" ]]; then
        backup_count=$(find "$PM_BACKUP_DIR" -name "*.json" 2>/dev/null | wc -l | xargs)
    fi

    # 4. 检查过期锁
    local now_ts
    now_ts=$(pm_timestamp_now)
    local expired_locks=0
    while read -r lock_name; do
        [[ -z "$lock_name" ]] && continue
        local expires_at
        expires_at=$(jq -r ".locks.\"$lock_name\".expires_at" "$STATE_FILE" 2>/dev/null || echo "")
        if [[ -n "$expires_at" && "$expires_at" != "null" ]]; then
            local expiry_ts
            expiry_ts=$(pm_timestamp_parse "$expires_at")
            if [[ $expiry_ts -gt 0 && $now_ts -gt $expiry_ts ]]; then
                ((expired_locks++))
            fi
        fi
    done < <(jq -r '.locks | keys[]' "$STATE_FILE" 2>/dev/null)

    if [[ $expired_locks -gt 0 ]]; then
        health_status="warning"
        checks=$(echo "$checks" | jq --arg c "expired_locks" --arg s "warning" --arg m "有 $expired_locks 个过期锁" '. += [{check: $c, status: $s, message: $m}]')
    fi

    # 5. 检查孤立开发者
    local idle_devs=0
    while read -r dev_id; do
        [[ -z "$dev_id" ]] && continue
        local task
        task=$(jq -r ".developers.\"$dev_id\".current_task" "$STATE_FILE" 2>/dev/null)
        if [[ "$task" == "null" || -z "$task" ]]; then
            ((idle_devs++))
        fi
    done < <(jq -r '.developers | keys[]' "$STATE_FILE" 2>/dev/null)

    # 6. 检查阻塞的任务
    local blocked_count=0
    blocked_count=$(jq '[.specs | to_entries[] | select(.value.status == "blocked")] | length' "$STATE_FILE")

    jq -n \
        --arg a "health" \
        --arg s "$health_status" \
        --arg ts "$(pm_now_iso)" \
        --argjson checks "$checks" \
        --arg bc "$backup_count" \
        --arg el "$expired_locks" \
        --arg id "$idle_devs" \
        --arg bl "$blocked_count" \
        '{
            action: $a,
            status: $s,
            data: {
                health_status: $s,
                checks: $checks,
                metrics: {
                    backup_count: ($bc | tonumber),
                    expired_locks: $el,
                    idle_developers: $id,
                    blocked_specs: $bl
                }
            },
            timestamp: $ts
        }'
}

# ============================================
# metrics 命令 - 导出指标
# ============================================
cmd_metrics() {
    jq -n \
        --arg a "metrics" \
        --arg ts "$(pm_now_iso)" \
        '
        {
            action: $a,
            status: "success",
            data: {
                gauge: {
                    total_specs: (.specs | length),
                    completed_specs: ([.specs | to_entries[] | select(.value.status == "completed")] | length),
                    in_progress_specs: ([.specs | to_entries[] | select(.value.status == "in_progress")] | length),
                    ready_specs: ([.specs | to_entries[] | select(.value.status == "ready")] | length),
                    blocked_specs: ([.specs | to_entries[] | select(.value.status == "blocked")] | length),
                    active_locks: (.locks | length),
                    active_worktrees: (.worktrees | length)
                },
                counter: {
                    total_assignments: ([.developers | to_entries[] | .value.current_task | select(. != null)] | length),
                    total_completed: ([.developers | to_entries[] | .value.completed_specs | add | length]),
                    backup_count: ([. ($STATE_FILE + ".backup.d") // "null" | (. as $dir | ($dir | if test -d then ([($dir | ls // []) | length) // 0) else 0 end)))
                },
                developers: [
                    .developers | to_entries[] | {
                        id: .key,
                        current_task: .value.current_task,
                        completed_count: (.value.completed_specs | length),
                        is_idle: (.value.current_task == null)
                    }
                ]
            },
            timestamp: $ts
        }
        ' "$STATE_FILE"
}

# ============================================
# events 命令 - 事件日志
# ============================================
cmd_events() {
    local limit="${1:-20}"

    # 从 updated_at 推断事件序列
    jq -n \
        --arg a "events" \
        --arg ts "$(pm_now_iso)" \
        --arg l "$limit" \
        '
        {
            action: $a,
            status: "success",
            data: {
                recent_events: [
                    .specs | to_entries[] |
                    select(.value.started_at != null or .value.completed_at != null) |
                    {
                        event_type: (if .value.completed_at then "completed" else "started" end),
                        spec_id: .key,
                        developer: .value.assignee,
                        timestamp: (.value.completed_at // .value.started_at)
                    }
                ] | sort_by(.timestamp) | reverse | .[0:($l | tonumber)]
            },
            note: "完整事件日志未实现，当前仅显示 Spec 变更"
        }
        ' "$STATE_FILE"
}

# ============================================
# 操作分发
# ============================================
ACTION="${1:-}"
shift || true

case "$ACTION" in
    read)
        cmd_read "$@"
        ;;
    update)
        cmd_update "$@"
        ;;
    set)
        cmd_set
        ;;
    assign)
        cmd_assign "$@"
        ;;
    complete)
        cmd_complete "$@"
        ;;
    unassign)
        cmd_unassign "$@"
        ;;
    progress)
        cmd_progress
        ;;
    validate)
        cmd_validate
        ;;
    backup)
        cmd_backup
        ;;
    restore)
        cmd_restore "$@"
        ;;
    list-backups)
        cmd_list_backups
        ;;
    rollback)
        cmd_rollback "$@"
        ;;
    health)
        cmd_health
        ;;
    metrics)
        cmd_metrics
        ;;
    events)
        cmd_events "$@"
        ;;
    *)
        cat <<EOF
用法: $(basename "$0") <command> [args]

核心命令:
  read [path]           读取状态 (jq 路径)
  assign <spec> <dev>   分配任务
  complete <spec> <dev> 完成任务
  progress              计算进度统计

维护命令:
  validate              验证状态文件
  backup / restore / rollback  备份与恢复
  health                健康检查
  metrics               指标导出
  events [n]            事件日志 (默认 20 条)

示例:
  $(basename "$0") read .specs.CORE-01
  $(basename "$0\") progress
  $(basename "$0") health
  $(basename "$0") metrics

EOF
        exit 1
        ;;
esac
