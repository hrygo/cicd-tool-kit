#!/bin/bash
# Project Manager - 锁管理
# 版本: 2.3.0
# 用法: ./scripts/lock.sh <command> [args]
#
# 命令:
#   acquire <dev> <lock_name> <spec_id> <reason>  获取锁
#   release <lock_name>                           释放锁
#   force <lock_name>                            强制释放锁
#   list                                          列出所有锁
#   status <lock_name>                            查看锁状态

set -euo pipefail

# 导入通用函数
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib/pm.sh
source "$SCRIPT_DIR/lib/pm.sh"

# 初始化 PM 环境
pm_ensure_dirs

# ============================================================================
# 命令处理
# ============================================================================

show_usage() {
    cat <<EOF
用法: $(basename "$0") <command> [args]

命令:
  acquire <dev> <lock_name> <spec_id> <reason>  获取锁
  release <lock_name>                           释放锁
  force <lock_name>                            强制释放锁 (管理员)
  list                                          列出所有锁
  status <lock_name>                            查看锁状态

示例:
  $(basename "$0") acquire dev-a runner CORE-01 "实现 Runner 生命周期"
  $(basename "$0") release runner
  $(basename "$0") force runner
  $(basename "$0") list
  $(basename "$0") status runner

锁命名规则:
  runner     - pkg/runner/ (dev-a)
  config     - pkg/config/ (dev-a)
  platform   - pkg/platform/ (dev-a)
  security   - pkg/security/ (dev-b)
  governance - pkg/governance/ (dev-b)
  observability - pkg/observability/ (dev-b)
  skill      - pkg/skill/, skills/ (dev-c)
  mcp        - pkg/mcp/ (dev-c)
  main       - main 分支更新 (项目经理)

EOF
}

COMMAND="${1:-}"

case "$COMMAND" in
    acquire)
        if [[ $# -lt 5 ]]; then
            log_error "缺少参数"
            show_usage
            exit 1
        fi
        DEV="$2"
        LOCK_NAME="$3"
        SPEC_ID="$4"
        REASON="$5"

        if pm_acquire_lock "$DEV" "$LOCK_NAME" "$SPEC_ID" "$REASON"; then
            pm_json_output "lock" "success" "{\"action\": \"acquire\", \"lock\": \"$LOCK_NAME\", \"owner\": \"$DEV\"}"
        else
            pm_json_output "lock" "error" "{\"action\": \"acquire\", \"lock\": \"$LOCK_NAME\", \"error\": \"获取失败\"}"
            exit 1
        fi
        ;;

    release)
        if [[ $# -lt 2 ]]; then
            log_error "缺少锁名称"
            show_usage
            exit 1
        fi
        LOCK_NAME="$2"
        pm_release_lock "$LOCK_NAME"
        pm_json_output "lock" "success" "{\"action\": \"release\", \"lock\": \"$LOCK_NAME\"}"
        ;;

    force)
        if [[ $# -lt 2 ]]; then
            log_error "缺少锁名称"
            show_usage
            exit 1
        fi
        LOCK_NAME="$2"
        if pm_force_release_lock "$LOCK_NAME"; then
            pm_json_output "lock" "success" "{\"action\": \"force\", \"lock\": \"$LOCK_NAME\"}"
        else
            pm_json_output "lock" "error" "{\"action\": \"force\", \"lock\": \"$LOCK_NAME\", \"error\": \"锁不存在\"}"
            exit 1
        fi
        ;;

    list)
        echo "当前锁状态:"
        echo "------------"
        local has_locks=false
        for lock in $(pm_list_locks); do
            has_locks=true
            lock_file="$LOCKS_DIR/$lock.lock"
            owner=$(grep "^locked_by:" "$lock_file" 2>/dev/null | cut -d' ' -f2-)
            spec=$(grep "^spec_id:" "$lock_file" 2>/dev/null | cut -d' ' -f2)
            reason=$(grep "^reason:" "$lock_file" 2>/dev/null | cut -d' ' -f2-)
            expires_at=$(grep "^expires_at:" "$lock_file" 2>/dev/null | cut -d' ' -f2-)

            # 检查是否过期
            local expiry_ts now_ts
            expiry_ts=$(pm_date_parse "$expires_at")
            now_ts=$(pm_timestamp)
            local status=""
            if [[ $now_ts -gt $expiry_ts && $expiry_ts -gt 0 ]]; then
                status=" [已过期]"
            fi

            echo "🔒 $lock - ${owner:-unknown} (${spec:-unknown}): ${reason:-无}${status}"
        done

        if [[ "$has_locks" == false ]]; then
            echo "无活跃锁"
        fi
        ;;

    status)
        if [[ $# -lt 2 ]]; then
            log_error "缺少锁名称"
            show_usage
            exit 1
        fi
        LOCK_NAME="$2"
        LOCK_FILE="$LOCKS_DIR/$LOCK_NAME.lock"

        if [[ ! -f "$LOCK_FILE" ]]; then
            echo "锁 $LOCK_NAME: 未锁定"
            exit 0
        fi

        echo "锁状态: $LOCK_NAME"
        echo "------------"
        cat "$LOCK_FILE"
        ;;

    *)
        show_usage
        exit 1
        ;;
esac
