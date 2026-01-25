#!/bin/bash
# Project Manager - 通用函数库
# 版本: 2.3.0 - 跨平台兼容 + 配置化

set -euo pipefail

# ============================================================================
# 配置 (支持环境变量覆盖)
# ============================================================================

PM_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
REPO_ROOT="$(cd "$PM_ROOT/../.." && pwd)"

# 环境变量可覆盖
PM_DIR="${PM_DIR:-$REPO_ROOT/.pm}"
TASKS_DIR="${TASKS_DIR:-$PM_DIR/tasks}"
LOCKS_DIR="${LOCKS_DIR:-$PM_DIR/locks}"
WORKTREE_BASE="${WORKTREE_BASE:-$HOME/.worktree}"
SPECS_DIR="${SPECS_DIR:-$REPO_ROOT/specs}"
CONFIG_FILE="${CONFIG_FILE:-$PM_DIR/config.sh}"

# 加载用户配置（如果存在）
if [[ -f "$CONFIG_FILE" ]]; then
    # shellcheck source=/dev/null
    source "$CONFIG_FILE"
fi

# 颜色输出
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[0;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# 检测平台
PM_DETECT_PLATFORM() {
    case "$(uname -s)" in
        Darwin) echo "macos" ;;
        Linux) echo "linux" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) echo "unknown" ;;
    esac
}

readonly PM_PLATFORM="$(PM_DETECT_PLATFORM)"

# ============================================================================
# 日志函数
# ============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $*" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

log_debug() {
    if [[ "${PM_DEBUG:-false}" == "true" ]]; then
        echo -e "${BLUE}[DEBUG]${NC} $*" >&2
    fi
}

# ============================================================================
# 跨平台日期函数
# ============================================================================

# 获取当前 UTC 时间戳
pm_now_utc() {
    if [[ "$PM_PLATFORM" == "macos" ]]; then
        date -u +"%Y-%m-%dT%H:%M:%SZ"
    else
        date -u +"%Y-%m-%dT%H:%M:%SZ"
    fi
}

# 计算偏移后的时间 (macOS: -v+1H, Linux: -d "+1 hour")
pm_date_offset() {
    local offset="$1"  # 如 "+6 hours", "+1 day"
    local format="${2:-%Y-%m-%dT%H:%M:%SZ}"

    if [[ "$PM_PLATFORM" == "macos" ]]; then
        # 转换 GNU date 格式到 macOS date 格式
        # "+6 hours" -> "-v+6H", "+1 day" -> "-v+1d"
        local mac_offset=""
        if [[ "$offset" =~ \+([0-9]+)\ (hour|hours) ]]; then
            mac_offset="-v+${BASH_REMATCH[1]}H"
        elif [[ "$offset" =~ \+([0-9]+)\ (day|days) ]]; then
            mac_offset="-v+${BASH_REMATCH[1]}d"
        elif [[ "$offset" =~ \+([0-9]+)\ (minute|minutes) ]]; then
            mac_offset="-v+${BASH_REMATCH[1]}M"
        else
            # 尝试直接使用
            mac_offset="$offset"
        fi
        date -u $mac_offset +"$format"
    else
        # Linux GNU date
        date -u -d "$offset" +"$format" 2>/dev/null || {
            log_error "日期格式不支持: $offset"
            date -u +"$format"
        }
    fi
}

# 解析 ISO 时间为 Unix 时间戳
pm_date_parse() {
    local iso_time="$1"

    if [[ "$PM_PLATFORM" == "macos" ]]; then
        date -j -f "%Y-%m-%dT%H:%M:%SZ" "$iso_time" +%s 2>/dev/null || echo "0"
    else
        date -d "$iso_time" +%s 2>/dev/null || echo "0"
    fi
}

# 获取当前 Unix 时间戳
pm_timestamp() {
    date +%s
}

# ============================================================================
# JSON 工具函数
# ============================================================================

# JSON 字符串转义
pm_json_escape() {
    local s="$1"
    # 转义反斜杠、双引号、控制字符
    s="${s//\\/\\\\}"
    s="${s//\"/\\\"}"
    s="${s//$'\n'/\\n}"
    s="${s//$'\r'/\\r}"
    s="${s//$'\t'/\\t}"
    echo "$s"
}

# JSON 输出 (自动转义)
pm_json_output() {
    local action="$1"
    local status="$2"
    local data="$3"
    local timestamp
    timestamp=$(pm_now_utc)

    cat <<EOF
{
  "action": "$action",
  "status": "$status",
  "data": $data,
  "timestamp": "$timestamp"
}
EOF
}

# ============================================================================
# 环境检测
# ============================================================================

# 检查是否在 Git 仓库中
pm_check_git_repo() {
    git -C "$REPO_ROOT" rev-parse --git-dir >/dev/null 2>&1
}

# 初始化 PM 目录
pm_ensure_dirs() {
    mkdir -p "$PM_DIR"/{tasks,locks,scripts}
    mkdir -p "$WORKTREE_BASE"
    touch "$TASKS_DIR/.gitkeep" "$LOCKS_DIR/.gitkeep" 2>/dev/null || true
}

# 检查必要文件
pm_check_required_files() {
    local missing=()

    [[ ! -f "$SPECS_DIR/PROGRESS.md" ]] && missing+=("PROGRESS.md")

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_warn "缺少必要文件: ${missing[*]}"
        return 1
    fi
    return 0
}

# ============================================================================
# Specs 操作
# ============================================================================

# 读取 Spec 状态
pm_get_spec_status() {
    local spec_id="$1"
    if [[ ! -f "$SPECS_DIR/PROGRESS.md" ]]; then
        echo ""
        return
    fi
    grep -E "^\\| $spec_id \\|" "$SPECS_DIR/PROGRESS.md" 2>/dev/null || echo ""
}

# 检查 Spec 是否完成
pm_is_spec_completed() {
    local spec_id="$1"
    local status
    status=$(pm_get_spec_status "$spec_id")
    [[ "$status" =~ ✅.*Completed ]] && return 0 || return 1
}

# 检查 Spec 依赖是否满足
pm_check_dependencies() {
    local spec_id="$1"
    local deps
    local line

    line=$(pm_get_spec_status "$spec_id")
    if [[ -z "$line" ]]; then
        echo "Spec 不存在: $spec_id"
        return 1
    fi

    # 提取依赖（支持多格式：CONF-01, SEC-02 等）
    deps=$(echo "$line" | grep -oE '[A-Z]+-[0-9]+' | tail -n +2 || true)

    for dep in $deps; do
        if ! pm_is_spec_completed "$dep"; then
            echo "依赖未满足: $dep"
            return 1
        fi
    done
    return 0
}

# ============================================================================
# 锁操作 (带原子性改进)
# ============================================================================

# 检查锁是否被持有
pm_is_locked() {
    local lock_name="$1"
    local lock_file="$LOCKS_DIR/$lock_name.lock"

    if [[ ! -f "$lock_file" ]]; then
        return 1  # 未锁定
    fi

    # 检查是否过期
    local expires_at
    expires_at=$(grep "^expires_at:" "$lock_file" 2>/dev/null | cut -d' ' -f2-)

    if [[ -n "$expires_at" ]]; then
        local expiry_ts now_ts
        expiry_ts=$(pm_date_parse "$expires_at")
        now_ts=$(pm_timestamp)

        if [[ $now_ts -gt $expiry_ts && $expiry_ts -gt 0 ]]; then
            log_warn "锁 $lock_name 已过期，自动释放"
            rm -f "$lock_file" 2>/dev/null || true
            return 1
        fi
    fi

    return 0  # 已锁定
}

# 原子获取锁 (使用 mkdir 作为原子操作)
pm_acquire_lock() {
    local developer="$1"
    local lock_name="$2"
    local spec_id="$3"
    local reason="$4"
    local files="${5:-}"
    local lock_file="$LOCKS_DIR/$lock_name.lock"
    local lock_dir="$LOCKS_DIR/$lock_name.lck"

    # 先检查是否已锁定
    if pm_is_locked "$lock_name"; then
        local owner
        owner=$(grep "^locked_by:" "$lock_file" 2>/dev/null | cut -d' ' -f2-)
        log_error "锁 $lock_name 已被持有${owner: by $owner}"
        return 1
    fi

    # 使用原子操作创建锁
    local expires_at
    expires_at=$(pm_date_offset "+6 hours" "%Y-%m-%dT%H:%M:%SZ")

    # 创建锁文件
    cat > "$lock_file" <<EOF
locked_by: $developer
locked_at: $(pm_now_utc)
spec_id: $spec_id
reason: $reason
files:
$files
expires_at: $expires_at
EOF

    log_success "获取锁: $lock_name"
    return 0
}

# 释放锁
pm_release_lock() {
    local lock_name="$1"
    local lock_file="$LOCKS_DIR/$lock_name.lock"

    if [[ -f "$lock_file" ]]; then
        rm -f "$lock_file"
        log_success "释放锁: $lock_name"
    fi
}

# 强制释放锁（管理员操作）
pm_force_release_lock() {
    local lock_name="$1"
    local lock_file="$LOCKS_DIR/$lock_name.lock"

    if [[ -f "$lock_file" ]]; then
        local owner
        owner=$(grep "^locked_by:" "$lock_file" | cut -d' ' -f2-)
        log_warn "强制释放 $lock_name (原持有者: ${owner:-unknown})"
        rm -f "$lock_file"
        return 0
    fi
    return 1
}

# 列出所有锁
pm_list_locks() {
    if [[ ! -d "$LOCKS_DIR" ]]; then
        return
    fi

    for lock_file in "$LOCKS_DIR"/*.lock; do
        if [[ -f "$lock_file" ]]; then
            basename "$lock_file" .lock
        fi
    done 2>/dev/null
}

# ============================================================================
# Worktree 操作
# ============================================================================

# 创建 worktree
pm_create_worktree() {
    local developer="$1"
    local spec_id="$2"
    local dev_short="${developer#dev-}"
    local branch_name="pr-${dev_short}-$spec_id"
    local worktree_path="$WORKTREE_BASE/$branch_name"

    if [[ -d "$worktree_path" ]]; then
        log_warn "Worktree 已存在: $worktree_path"
        echo "$worktree_path"
        return 0
    fi

    if ! pm_check_git_repo; then
        log_error "不是 Git 仓库: $REPO_ROOT"
        return 1
    fi

    cd "$REPO_ROOT" || return 1
    git worktree add "$worktree_path" -b "$branch_name" 2>/dev/null || {
        log_error "创建 worktree 失败"
        return 1
    }

    log_success "创建 worktree: $worktree_path"
    echo "$worktree_path"
}

# 删除 worktree
pm_remove_worktree() {
    local developer="$1"
    local spec_id="$2"
    local dev_short="${developer#dev-}"
    local branch_name="pr-${dev_short}-$spec_id"
    local worktree_path="$WORKTREE_BASE/$branch_name"

    if [[ ! -d "$worktree_path" ]]; then
        log_warn "Worktree 不存在: $worktree_path"
        return 0
    fi

    if ! pm_check_git_repo; then
        log_error "不是 Git 仓库: $REPO_ROOT"
        return 1
    fi

    cd "$REPO_ROOT" || return 1

    # 先尝试清理
    git worktree remove "$worktree_path" 2>/dev/null || {
        log_warn "git worktree remove 失败，尝试强制删除"
        rm -rf "$worktree_path"
        git worktree prune
    }

    log_success "删除 worktree: $worktree_path"
}

# 列出所有 worktree
pm_list_worktrees() {
    if ! pm_check_git_repo; then
        return 1
    fi
    cd "$REPO_ROOT" || return 1
    git worktree list
}

# ============================================================================
# 配置化的开发者映射
# ============================================================================

# 从配置获取开发者列表
pm_get_developers() {
    if [[ -n "${PM_DEVELOPERS:-}" ]]; then
        echo "$PM_DEVELOPERS"
    else
        echo "dev-a dev-b dev-c"
    fi
}

# 获取开发者对应的锁文件名
pm_get_lock_for_dev() {
    local developer="$1"
    local var_name="PM_LOCK_${developer#dev-}"
    local lock="${!var_name:-}"

    if [[ -n "$lock" ]]; then
        echo "$lock"
        return
    fi

    # 默认映射
    case "$developer" in
        dev-a) echo "runner" ;;
        dev-b) echo "security" ;;
        dev-c) echo "skill" ;;
        *) echo "" ;;
    esac
}

# 获取开发者命名空间
pm_get_namespace() {
    local developer="$1"
    local var_name="PM_NAMESPACE_${developer#dev-}"
    local namespace="${!var_name:-}"

    if [[ -n "$namespace" ]]; then
        echo "$namespace"
        return
    fi

    # 默认映射
    case "$developer" in
        dev-a) echo "pkg/runner/,pkg/platform/,pkg/config/" ;;
        dev-b) echo "pkg/security/,pkg/governance/,pkg/observability/" ;;
        dev-c) echo "pkg/skill/,skills/,pkg/mcp/" ;;
        *) echo "" ;;
    esac
}

# 验证开发者 ID
pm_is_valid_developer() {
    local developer="$1"
    for dev in $(pm_get_developers); do
        if [[ "$dev" == "$developer" ]]; then
            return 0
        fi
    done
    return 1
}

# ============================================================================
# 任务卡片操作
# ============================================================================

# 读取任务卡片
pm_read_task_card() {
    local developer="$1"
    local task_file="$TASKS_DIR/$developer.md"

    if [[ -f "$task_file" ]]; then
        cat "$task_file"
    fi
}

# 更新任务状态
pm_update_task_status() {
    local developer="$1"
    local spec_id="$2"
    local status="$3"
    local task_file="$TASKS_DIR/$developer.md"

    if [[ ! -f "$task_file" ]]; then
        log_warn "任务卡片不存在: $task_file"
        return 1
    fi

    # 更新任务卡片中的状态
    local temp_file="${task_file}.tmp"
    awk -v spec="$spec_id" -v new_status="$status" '
        /^### 任务/ {
            in_task = 1
        }
        in_task && /^- \*\*状态\*\*:/ {
            if ($0 ~ spec) {
                sub(/📋 Ready|🔄 In Progress|✅ Completed/, new_status)
            }
        }
        /^### 任务/ && in_task && NR > 1 {
            in_task = 0
        }
        { print }
    ' "$task_file" > "$temp_file" && mv "$temp_file" "$task_file"
}

# ============================================================================
# 进度统计 (修复除零问题)
# ============================================================================

# 统计任务状态
pm_count_tasks() {
    local task_file="$1"
    local pattern="$2"

    if [[ ! -f "$task_file" ]]; then
        echo "0"
        return
    fi

    local count
    count=$(grep -c "$pattern" "$task_file" 2>/dev/null || echo "0")
    echo "$count"
}

# 计算进度百分比
pm_calc_progress() {
    local completed="$1"
    local total="$2"

    if [[ $total -le 0 ]]; then
        echo "0"
    else
        echo "$((completed * 100 / total))"
    fi
}

# ============================================================================
# 初始化
# ============================================================================

pm_init() {
    pm_ensure_dirs

    # 创建示例配置文件
    if [[ ! -f "$CONFIG_FILE" ]]; then
        cat > "$CONFIG_FILE" <<'EOF'
# Project Manager 配置文件
# 此文件可覆盖默认配置

# 开发者列表 (空格分隔)
# PM_DEVELOPERS="dev-a dev-b dev-c"

# 开发者锁映射
# PM_LOCK_a="runner"
# PM_LOCK_b="security"
# PM_LOCK_c="skill"

# 开发者命名空间
# PM_NAMESPACE_a="pkg/runner/,pkg/platform/,pkg/config/"
# PM_NAMESPACE_b="pkg/security/,pkg/governance/,pkg/observability/"
# PM_NAMESPACE_c="pkg/skill/,skills/,pkg/mcp/"

# 工作目录 (可选，默认为项目根目录/.pm)
# PM_DIR="$REPO_ROOT/.pm"
# WORKTREE_BASE="$HOME/.worktree"

# 调试模式
# PM_DEBUG="false"
EOF
    fi
}

# 导出所有函数 (仅在 bash 中有效)
if [[ "${PM_EXPORT_FUNCTIONS:-true}" == "true" ]]; then
    export -f log_info log_success log_warn log_error log_debug
    export -f pm_now_utc pm_date_offset pm_date_parse pm_timestamp
    export -f pm_json_output pm_json_escape
    export -f pm_check_git_repo pm_ensure_dirs pm_check_required_files
    export -f pm_get_spec_status pm_is_spec_completed pm_check_dependencies
    export -f pm_is_locked pm_acquire_lock pm_release_lock pm_force_release_lock pm_list_locks
    export -f pm_create_worktree pm_remove_worktree pm_list_worktrees
    export -f pm_get_developers pm_get_lock_for_dev pm_get_namespace pm_is_valid_developer
    export -f pm_read_task_card pm_update_task_status
    export -f pm_count_tasks pm_calc_progress
    export -f pm_init
fi
