#!/bin/bash
# ai-suggest.sh - AI 辅助决策脚本 v3.1
# 脚本准备结构化数据，AI Agent 做最终决策

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
    pm_json_output "ai-suggest" "error" '{"error": "需要 jq"}' >&2
    exit 1
fi

if [[ ! -f "$STATE_FILE" ]]; then
    pm_json_output "ai-suggest" "error" '{"error": "状态文件不存在"}' >&2
    exit 1
fi

# ============================================
# next-task 命令 - 推荐下一任务
# 脚本准备候选任务数据，AI 做最终推荐
# ============================================
cmd_next_task() {
    local developer_id="${1:-}"

    if [[ -n "$developer_id" ]]; then
        pm_validate_developer_id "$developer_id" "ai-suggest" || return 1
        pm_check_developer_exists "$developer_id" "ai-suggest" || return 1
    fi

    # 获取所有可分配的任务候选
    local candidates_json
    candidates_json=$(jq '
        .specs | to_entries[] |
        select(.value.status == "ready" or .value.status == "pending") |
        {
            spec_id: .key,
            name: .value.name,
            status: .value.status,
            dependencies: .value.dependencies,
            priority: .value.priority // "p2",
            phase: .value.phase // ""
        }
    ' "$STATE_FILE" | jq -s '.')

    # 检查每个候选任务的依赖状态
    local ready_candidates_json
    ready_candidates_json=$(echo "$candidates_json" | jq 'map(. + {dependencies_met: true})')

    # 获取开发者状态
    local developers_json
    if [[ -n "$developer_id" ]]; then
        developers_json=$(jq ".developers | {\"$developer_id\": .developers[\"$developer_id\"]}" "$STATE_FILE")
    else
        developers_json=$(jq '.developers' "$STATE_FILE")
    fi

    # 获取锁状态
    local locks_json
    locks_json=$(jq '.locks | to_entries | map({name: .key, locked_by: .value.locked_by, spec_id: .value.spec_id})' "$STATE_FILE")

    # 输出结构化数据供 AI 分析
    jq -n \
        --arg a "next-task" \
        --arg s "success" \
        --arg ts "$(pm_now_iso)" \
        --argjson candidates "$ready_candidates_json" \
        --argjson developers "$developers_json" \
        --argjson locks "$locks_json" \
        '{
            action: $a,
            status: $s,
            data: {
                candidates: $candidates,
                developers: $developers,
                active_locks: $locks,
                note: "AI 请分析并推荐最佳任务"
            },
            timestamp: $ts
        }'
}

# ============================================
# analyze-blockers 命令 - 分析阻塞
# ============================================
cmd_analyze_blockers() {
    # 获取所有被阻塞的任务
    local blocked_json
    blocked_json=$(jq '
        .specs | to_entries[] |
        select(.value.status == "blocked" or (.value.dependencies | length) > 0) |
        {
            spec_id: .key,
            name: .value.name,
            status: .value.status,
            dependencies: .value.dependencies
        }
    ' "$STATE_FILE" | jq -s '.')

    # 分析依赖链
    local dependency_chain
    dependency_chain=$(echo "$blocked_json" | jq '
        map(.spec_id as $spec_id |
            .dependencies[]? |
            select(. != null) as $dep |
            {dependent: $spec_id, depends_on: $dep}
        ) | select(. != null)
    ')

    # 获取每个依赖的状态
    local dep_status
    dep_status=$(jq '
        .specs | to_entries[] |
        {spec_id: .key, status: .value.status}
    ' "$STATE_FILE" | jq -s 'map({(.spec_id): .status}) | add')

    jq -n \
        --arg a "analyze-blockers" \
        --arg s "success" \
        --arg ts "$(pm_now_iso)" \
        --argjson blocked "$blocked_json" \
        --argjson chain "$dependency_chain" \
        --argjson dep_status "$dep_status" \
        '{
            action: $a,
            status: $s,
            data: {
                blocked_specs: $blocked,
                dependency_chain: $chain,
                dependency_status: $dep_status,
                note: "AI 请分析阻塞原因并给出解决方案"
            },
            timestamp: $ts
        }'
}

# ============================================
# optimize-workload 命令 - 优化负载
# ============================================
cmd_optimize_workload() {
    # 获取当前负载分布
    local workload_json
    workload_json=$(jq '
        .developers | to_entries[] |
        {
            developer_id: .key,
            name: .value.name,
            current_task: .value.current_task,
            completed_count: (.value.completed_specs | length),
            namespace: .value.namespace
        }
    ' "$STATE_FILE" | jq -s '.')

    # 获取待分配任务
    local pending_tasks
    pending_tasks=$(jq '
        .specs | to_entries[] |
        select(.value.status == "ready") |
        {
            spec_id: .key,
            name: .value.name,
            priority: .value.priority // "p2",
            phase: .value.phase // ""
        }
    ' "$STATE_FILE" | jq -s '.')

    # 获取每个命名空间的任务数量
    local namespace_load
    namespace_load=$(jq '
        .specs | to_entries[] |
        select(.value.status != "completed") |
        .spec_id as $id |
            if ($id | startswith("CORE")) then "runner"
            elif ($id | startswith("CONF")) then "config"
            elif ($id | startswith("SEC")) then "security"
            elif ($id | startswith("GOV")) then "governance"
            elif ($id | startswith("PLAT")) then "platform"
            elif ($id | startswith("SKILL") or ($id | startswith("LIB"))) then "skill"
            elif ($id | startswith("MCP")) then "mcp"
            else "other"
            end
        | {namespace: ., spec_id: $id}
    ' "$STATE_FILE" | jq -s 'group_by(.namespace) | map({namespace: .[0].namespace, count: length})')

    jq -n \
        --arg a "optimize-workload" \
        --arg s "success" \
        --arg ts "$(pm_now_iso)" \
        --argjson workload "$workload_json" \
        --argjson pending "$pending_tasks" \
        --argjson namespace_load "$namespace_load" \
        '{
            action: $a,
            status: $s,
            data: {
                current_workload: $workload,
                pending_tasks: $pending,
                namespace_load: $namespace_load,
                note: "AI 请分析负载均衡并给出优化建议"
            },
            timestamp: $ts
        }'
}

# ============================================
# detect-conflicts 命令 - 检测冲突
# ============================================
cmd_detect_conflicts() {
    # 检测锁冲突
    local lock_conflicts
    lock_conflicts=$(jq '
        .specs | to_entries[] |
        select(.value.status == "ready") |
        .spec_id as $id |
            (if ($id | startswith("CORE")) then "runner"
            elif ($id | startswith("CONF")) then "config"
            elif ($id | startswith("SEC")) then "security"
            elif ($id | startswith("PLAT")) then "platform"
            elif ($id | startswith("SKILL")) then "skill"
            elif ($id | startswith("MCP")) then "mcp"
            else null
            end) as $lock |
            select($lock != null) |
            {spec_id: $id, required_lock: $lock, available: (.locks[$lock] // null == null)}
    ' "$STATE_FILE" | jq -s '[.[] | select(.available == false)]')

    # 检测开发者冲突（多个任务需要同一个开发者）
    local developer_conflicts
    developer_conflicts=$(jq '
        .specs | to_entries[] |
        select(.value.status == "ready" and .value.assignee != null) |
        {spec_id: .key, assignee: .value.assignee}
    ' "$STATE_FILE" | jq -s '
        group_by(.assignee) |
        map(select(length > 1) | {developer: .[0].assignee, count: length, specs: map(.spec_id)})
    ')

    # 检测依赖循环
    local circular_deps
    circular_deps=$(jq '
        # 简化版：检测是否有 A 依赖 B，B 又依赖 A 的情况
        .specs | to_entries[] |
        select(.value.dependencies | length) > 0 |
        .key as $spec |
            .value.dependencies[] |
            select(. != null) as $dep |
            {spec: $spec, dep: $dep}
    ' "$STATE_FILE" | jq -s '
        map(select(
            any(.specs[]; select(.key == .dep and (.value.dependencies // []) | contains([.spec])))
        ))
    ')

    jq -n \
        --arg a "detect-conflicts" \
        --arg s "success" \
        --arg ts "$(pm_now_iso)" \
        --argjson lock_conflicts "$lock_conflicts" \
        --argjson dev_conflicts "$developer_conflicts" \
        --argjson circular "$circular_deps" \
        '{
            action: $a,
            status: $s,
            data: {
                lock_conflicts: $lock_conflicts,
                developer_conflicts: $dev_conflicts,
                potential_circular_deps: $circular,
                has_conflicts: ($lock_conflicts | length > 0 or $dev_conflicts | length > 0),
                note: "AI 请分析冲突并给出解决方案"
            },
            timestamp: $ts
        }'
}

# ============================================
# readiness-report 命令 - 准备就绪报告
# ============================================
cmd_readiness_report() {
    # 获取所有准备就绪的任务
    local ready_specs
    ready_specs=$(jq '
        .specs | to_entries[] |
        select(.value.status == "ready") |
        {
            spec_id: .key,
            name: .value.name,
            priority: .value.priority // "p2",
            assignee: .value.assignee,
            phase: .value.phase,
            dependencies: .value.dependencies
        }
    ' "$STATE_FILE" | jq -s '.')

    # 按优先级排序
    ready_specs=$(echo "$ready_specs" | jq '
        sort_by(.priority) |
        map(. + {
            priority_score: (if .priority == "p0" then 3 elif .priority == "p1" then 2 else 1 end)
        }) | sort_by(.priority_score) | reverse | map(del(.priority_score))
    ')

    # 获取空闲开发者
    local available_developers
    available_developers=$(jq '
        .developers | to_entries[] |
        select(.value.current_task == null) |
        {
            developer_id: .key,
            name: .value.name,
            namespace: .value.namespace
        }
    ' "$STATE_FILE" | jq -s '.')

    # 获取可用锁
    local available_locks
    available_locks=$(jq '
        .locks | to_entries |
        map({name: .key, is_locked: true}) |
        (["runner", "config", "platform", "security", "governance", "observability", "skill", "mcp"] |
            map({name: ., is_locked: false}) as $all |
            $all + map(select(.is_locked == true)))
    ' "$STATE_FILE" | jq '
        map(select(.is_locked == false)) |
        map(.name)
    ')

    jq -n \
        --arg a "readiness-report" \
        --arg s "success" \
        --arg ts "$(pm_now_iso)" \
        --argjson ready "$ready_specs" \
        --argjson developers "$available_developers" \
        --argjson locks "$available_locks" \
        '{
            action: $a,
            status: $s,
            data: {
                ready_specs: $ready,
                available_developers: $developers,
                available_locks: $locks,
                summary: {
                    ready_count: ($ready | length),
                    available_devs: ($developers | length),
                    available_locks: ($locks | length)
                },
                note: "AI 请基于此数据生成分配建议"
            },
            timestamp: $ts
        }'
}

# ============================================
# 操作分发
# ============================================
ACTION="${1:-}"
shift || true

case "$ACTION" in
    next-task)
        cmd_next_task "$@"
        ;;
    analyze-blockers)
        cmd_analyze_blockers
        ;;
    optimize-workload)
        cmd_optimize_workload
        ;;
    detect-conflicts)
        cmd_detect_conflicts
        ;;
    readiness-report)
        cmd_readiness_report
        ;;
    *)
        cat <<EOF
用法: $(basename "$0") <command> [args]

命令:
  next-task [dev_id]     推荐下一任务（准备候选数据供 AI 决策）
  analyze-blockers       分析阻塞原因
  optimize-workload      优化负载分配
  detect-conflicts       检测资源冲突
  readiness-report       生成准备就绪报告

说明:
  这些脚本只负责准备结构化数据，最终决策由 AI Agent 完成。

示例:
  $(basename "$0") next-task dev-a
  $(basename "$0") readiness-report

EOF
        exit 1
        ;;
esac
