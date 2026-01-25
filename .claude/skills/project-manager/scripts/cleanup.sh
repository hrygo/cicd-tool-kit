#!/bin/bash
# Project Manager - 清理工作区
# 版本: 2.3.0
# 用法: ./scripts/cleanup.sh <developer_id> <spec_id> [--force]
#
# 示例:
#   ./scripts/cleanup.sh dev-a CORE-01
#   ./scripts/cleanup.sh dev-a CORE-01 --force

set -euo pipefail

# 导入通用函数
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib/pm.sh
source "$SCRIPT_DIR/lib/pm.sh"

show_usage() {
    cat <<EOF
用法: $(basename "$0") <developer_id> <spec_id> [--force]

清理已完成任务的 worktree 并释放锁。

参数:
  developer_id  开发者 ID (dev-a, dev-b, dev-c)
  spec_id       Spec ID (如 CORE-01, SEC-01)
  --force       强制清理 (即使 worktree 不存在)

示例:
  $(basename "$0") dev-a CORE-01
  $(basename "$0") dev-b SEC-01 --force

EOF
}

if [[ $# -lt 2 ]]; then
    show_usage
    pm_json_output "cleanup" "error" "{\"error\": \"缺少参数\"}"
    exit 1
fi

DEVELOPER="$1"
SPEC_ID="$2"
FORCE_CLEANUP=false

if [[ "$3" == "--force" ]] || [[ "$SPEC_ID" == "--force" ]]; then
    FORCE_CLEANUP=true
fi

# 初始化 PM 环境
pm_ensure_dirs

log_info "=== 项目经理：清理工作区 ==="
log_info "开发者: $DEVELOPER, Spec: $SPEC_ID"

# 验证开发者
if ! pm_is_valid_developer "$DEVELOPER"; then
    log_error "无效的开发者 ID: $DEVELOPER"
    pm_json_output "cleanup" "error" "{\"error\": \"开发者 ID 无效\"}"
    exit 1
fi

# 获取对应的锁名
LOCK_NAME=$(pm_get_lock_for_dev "$DEVELOPER")

# 1. 删除 worktree
log_info "删除 worktree..."
if pm_remove_worktree "$DEVELOPER" "$SPEC_ID"; then
    log_success "Worktree 已删除"
else
    if [[ "$FORCE_CLEANUP" == true ]]; then
        log_warn "强制清理: worktree 处理完成"
    else
        log_warn "Worktree 不存在或已删除"
    fi
fi

# 2. 释放锁
if [[ -n "$LOCK_NAME" ]]; then
    log_info "释放锁: $LOCK_NAME.lock"
    pm_release_lock "$LOCK_NAME"
fi

# 3. 输出结果
pm_json_output "cleanup" "success" "{\"developer\": \"$DEVELOPER\", \"spec_id\": \"$SPEC_ID\", \"lock_released\": \"${LOCK_NAME:-none}\"}"

log_success "清理完成"
