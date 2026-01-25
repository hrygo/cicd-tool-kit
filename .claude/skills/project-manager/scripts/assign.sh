#!/bin/bash
# Project Manager - 分配任务
# 版本: 2.3.0
# 用法: ./scripts/assign.sh <developer_id> <spec_id>
#
# 示例:
#   ./scripts/assign.sh dev-a CORE-01
#   ./scripts/assign.sh dev-b SEC-01

set -euo pipefail

# 导入通用函数
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib/pm.sh
source "$SCRIPT_DIR/lib/pm.sh"

# ============================================================================
# 参数验证
# ============================================================================

show_usage() {
    cat <<EOF
用法: $(basename "$0") <developer_id> <spec_id>

为开发者分配任务，创建 worktree 和 PR 分支。

参数:
  developer_id  开发者 ID (dev-a, dev-b, dev-c)
  spec_id       Spec ID (如 CORE-01, SEC-01)

环境变量:
  WORKTREE_BASE  Worktree 基目录 (默认: ~/.worktree)
  PM_DIR         PM 状态目录 (默认: .pm)
  PM_DEBUG       启用调试输出

示例:
  $(basename "$0") dev-a CORE-01
  $(basename "$0") dev-b SEC-01
  WORKTREE_BASE=/tmp/worktree $(basename "$0") dev-c LIB-01

EOF
}

if [[ $# -lt 2 ]]; then
    show_usage
    pm_json_output "assign" "error" "{\"error\": \"缺少参数\"}"
    exit 1
fi

DEVELOPER="$1"
SPEC_ID="$2"

# 初始化 PM 环境
pm_ensure_dirs

# 验证开发者 ID (使用配置化函数)
if ! pm_is_valid_developer "$DEVELOPER"; then
    log_error "无效的开发者 ID: $DEVELOPER"
    pm_json_output "assign" "error" "{\"error\": \"开发者 ID 无效，可用: $(pm_get_developers)\"}"
    exit 1
fi

# 验证 Spec ID 格式
if [[ ! "$SPEC_ID" =~ ^[A-Z]+-[0-9]+$ ]]; then
    log_error "无效的 Spec ID 格式: $SPEC_ID (应为 XXX-NN)"
    pm_json_output "assign" "error" "{\"error\": \"Spec ID 格式无效\"}"
    exit 1
fi

# ============================================================================
# 预检查
# ============================================================================

log_info "=== 项目经理：分配任务 ==="
log_debug "开发者: $DEVELOPER, Spec: $SPEC_ID"
log_debug "工作区: $REPO_ROOT"
log_debug "Worktree 基目录: $WORKTREE_BASE"

# 1. 检查 Git 仓库
if ! pm_check_git_repo; then
    log_error "不是 Git 仓库: $REPO_ROOT"
    pm_json_output "assign" "error" "{\"error\": \"不是 Git 仓库\"}"
    exit 1
fi

# 2. 检查必要文件
if ! pm_check_required_files 2>/dev/null; then
    log_warn "PROGRESS.md 不存在，跳过依赖检查"
else
    # 3. 检查 Spec 依赖
    log_info "检查依赖..."
    if ! dep_error=$(pm_check_dependencies "$SPEC_ID" 2>&1); then
        log_error "$dep_error"
        pm_json_output "assign" "error" "{\"spec_id\": \"$SPEC_ID\", \"error\": \"$dep_error\"}"
        exit 1
    fi
    log_success "依赖检查通过"
fi

# 4. 获取对应的锁名
LOCK_NAME=$(pm_get_lock_for_dev "$DEVELOPER")
NAMESPACE=$(pm_get_namespace "$DEVELOPER")

if [[ -z "$LOCK_NAME" ]]; then
    log_error "无法确定 $DEVELOPER 的锁映射"
    pm_json_output "assign" "error" "{\"error\": \"锁映射未配置\"}"
    exit 1
fi

# 5. 检查文件锁
log_info "检查文件锁: $LOCK_NAME.lock"
if pm_is_locked "$LOCK_NAME"; then
    lock_file="$LOCKS_DIR/$LOCK_NAME.lock"
    owner=$(grep "^locked_by:" "$lock_file" 2>/dev/null | cut -d' ' -f2-)
    log_error "锁已被 ${owner:-unknown} 持有"
    pm_json_output "assign" "error" "{\"lock\": \"$LOCK_NAME\", \"owner\": \"${owner:-unknown}\"}"
    exit 1
fi

# 6. 检查开发者当前任务
log_info "检查当前任务..."
task_file="$TASKS_DIR/$DEVELOPER.md"
if [[ -f "$task_file" ]]; then
    current_task=$(grep "状态.*🔄" "$task_file" 2>/dev/null | head -1 || true)
    if [[ -n "$current_task" ]]; then
        log_warn "开发者有进行中的任务"
        pm_json_output "assign" "error" "{\"developer\": \"$DEVELOPER\", \"current_task\": \"进行中\"}"
        exit 1
    fi
fi

# ============================================================================
# 执行分配
# ============================================================================

# 1. 创建 worktree
log_info "创建 worktree..."
if ! WORKTREE_PATH=$(pm_create_worktree "$DEVELOPER" "$SPEC_ID"); then
    pm_json_output "assign" "error" "{\"error\": \"创建 worktree 失败\"}"
    exit 1
fi

# 2. 获取文件锁
log_info "获取文件锁: $LOCK_NAME.lock"
if ! pm_acquire_lock "$DEVELOPER" "$LOCK_NAME" "$SPEC_ID" "实现 $SPEC_ID" ""; then
    pm_json_output "assign" "error" "{\"error\": \"获取锁失败\"}"
    exit 1
fi

# 3. 计算分支名
dev_short="${DEVELOPER#dev-}"
branch_name="pr-${dev_short}-$SPEC_ID"

# 4. 输出结果
cat <<EOF

{
  "action": "assign",
  "status": "success",
  "data": {
    "developer": "$DEVELOPER",
    "spec_id": "$SPEC_ID",
    "worktree_path": "$(pm_json_escape "$WORKTREE_PATH")",
    "branch": "$branch_name",
    "lock_file": ".pm/locks/$LOCK_NAME.lock",
    "namespace": "$(pm_json_escape "$NAMESPACE")",
    "instructions": "请在 worktree 中实现 $SPEC_ID，完成后更新任务卡片并创建 PR"
  },
  "timestamp": "$(pm_now_utc)"
}

EOF

log_success "任务分配完成"
log_info "Worktree: $WORKTREE_PATH"
log_info "分支: $branch_name"
log_info "下一步: cd $WORKTREE_PATH"
