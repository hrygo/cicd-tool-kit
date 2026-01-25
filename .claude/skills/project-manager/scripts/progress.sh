#!/bin/bash
# Project Manager - 收集进展
# 版本: 2.3.0
# 用法: ./scripts/progress.sh [--update]
#
# 示例:
#   ./scripts/progress.sh          # 输出进展 JSON
#   ./scripts/progress.sh --update # 更新 PROGRESS.md

set -euo pipefail

# 导入通用函数
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib/pm.sh
source "$SCRIPT_DIR/lib/pm.sh"

UPDATE_PROGRESS=false

if [[ "${1:-}" == "--update" ]]; then
    UPDATE_PROGRESS=true
fi

# 初始化 PM 环境
pm_ensure_dirs

log_info "=== 项目经理：收集进展 ==="

# 读取所有任务卡片
declare -A dev_progress

for dev in $(pm_get_developers); do
    task_file="$TASKS_DIR/$dev.md"
    if [[ -f "$task_file" ]]; then
        # 使用新函数统计任务状态
        completed=$(pm_count_tasks "$task_file" "✅ Completed")
        in_progress=$(pm_count_tasks "$task_file" "🔄 In Progress")
        ready=$(pm_count_tasks "$task_file" "📋 Ready")

        dev_progress["$dev"]="{\"completed\": $completed, \"in_progress\": $in_progress, \"ready\": $ready}"
    else
        dev_progress["$dev"]="{\"completed\": 0, \"in_progress\": 0, \"ready\": 0}"
    fi
done

# 计算总体进度 (修复除零问题)
total_completed=0
total_specs=0

if [[ -f "$SPECS_DIR/PROGRESS.md" ]]; then
    total_completed=$(pm_count_tasks "$SPECS_DIR/PROGRESS.md" "✅ Completed")
    # 统计总 spec 数 (非空行)
    total_specs=$(grep -c "^\\| " "$SPECS_DIR/PROGRESS.md" 2>/dev/null || echo "0")
fi

# 确保总数至少为 1 避免除零
total_specs=$((total_specs == 0 ? 32 : total_specs))
progress_percent=$(pm_calc_progress "$total_completed" "$total_specs")

# 收集活跃锁
active_locks_json=""
for lock in $(pm_list_locks); do
    if [[ -n "$active_locks_json" ]]; then
        active_locks_json="$active_locks_json,"
    fi
    active_locks_json="$active_locks_json\"$lock\""
done

# 统计 worktree 数量
worktrees_count=0
if pm_check_git_repo 2>/dev/null; then
    worktrees_count=$(git -C "$REPO_ROOT" worktree list 2>/dev/null | wc -l | tr -d ' ' || echo "0")
fi

# 输出 JSON
cat <<EOF
{
  "action": "progress",
  "status": "success",
  "data": {
    "summary": {
      "total_progress": "$progress_percent%",
      "completed": $total_completed,
      "total": $total_specs
    },
    "developers": {
$(for dev in dev-a dev-b dev-c; do
    echo "      \"$dev\": ${dev_progress[$dev]:-{\"completed\": 0, \"in_progress\": 0, \"ready\": 0}},"
done | head -n -1
)
    },
    "active_locks": [${active_locks_json:-}],
    "worktrees": $worktrees_count
  },
  "timestamp": "$(pm_now_utc)"
}
EOF

# 如果需要更新 PROGRESS.md
if [[ "$UPDATE_PROGRESS" == true ]]; then
    log_info "更新 PROGRESS.md..."
    # TODO: 实现自动更新逻辑
    log_warn "自动更新功能待实现"
fi
