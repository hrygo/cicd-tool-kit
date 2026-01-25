#!/bin/bash
# Project Manager - 生成报告
# 版本: 2.3.0
# 用法: ./scripts/report.sh [--type weekly|milestone|executive] [--output path]
#
# 示例:
#   ./scripts/report.sh
#   ./scripts/report.sh --type weekly
#   ./scripts/report.sh --type milestone --output reports/

set -euo pipefail

# 导入通用函数
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib/pm.sh
source "$SCRIPT_DIR/lib/pm.sh"

REPORT_TYPE="weekly"
OUTPUT_PATH=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --type)
            REPORT_TYPE="$2"
            shift 2
            ;;
        --output)
            OUTPUT_PATH="$2"
            shift 2
            ;;
        *)
            shift
            ;;
    esac
done

# 初始化 PM 环境
pm_ensure_dirs

log_info "=== 项目经理：生成报告 ($REPORT_TYPE) ==="

# 读取 PROGRESS.md 获取数据
PROGRESS_FILE="$SPECS_DIR/PROGRESS.md"

if [[ ! -f "$PROGRESS_FILE" ]]; then
    log_error "PROGRESS.md 不存在: $PROGRESS_FILE"
    pm_json_output "report" "error" "{\"error\": \"PROGRESS.md 不存在\"}"
    exit 1
fi

# 提取数据
total_progress=$(grep "^\\*\\*总进度\\*\\*" "$PROGRESS_FILE" 2>/dev/null | sed 's/.*\([0-9]\+%\).*/\1/' || echo "N/A")
current_phase=$(grep "^\\*\\*当前 Phase\\*\\*" "$PROGRESS_FILE" 2>/dev/null | sed 's/.*\*\*: //' || echo "Unknown")
report_date=$(date +"%Y-%m-%d")

# 生成报告内容
REPORT_CONTENT=""

case "$REPORT_TYPE" in
    weekly)
        REPORT_CONTENT="# 项目周报 - $report_date

## 摘要

- **总进度**: $total_progress
- **报告日期**: $report_date
- **当前阶段**: $current_phase

## 本周完成

"
        # 获取最近完成的 5 个 specs
        local completed_specs
        completed_specs=$(grep "✅ Completed" "$PROGRESS_FILE" 2>/dev/null | tail -5 || echo "无")
        if [[ "$completed_specs" == "无" ]]; then
            REPORT_CONTENT="${REPORT_CONTENT}暂无完成的 Spec

"
        else
            while IFS= read -r line; do
                REPORT_CONTENT="${REPORT_CONTENT}- $line
"
            done <<< "$completed_specs"
        fi

        REPORT_CONTENT="${REPORT_CONTENT}
## 下周计划

"
        # 获取待开始的 5 个 specs
        local ready_specs
        ready_specs=$(grep "📋 Ready" "$PROGRESS_FILE" 2>/dev/null | head -5 || echo "无")
        if [[ "$ready_specs" == "无" ]]; then
            REPORT_CONTENT="${REPORT_CONTENT}暂无待开始的 Spec

"
        else
            while IFS= read -r line; do
                REPORT_CONTENT="${REPORT_CONTENT}- $line
"
            done <<< "$ready_specs"
        fi

        REPORT_CONTENT="${REPORT_CONTENT}
## 风险与阻塞

"
        # 检查阻塞
        if grep -q "阻塞" "$PROGRESS_FILE" 2>/dev/null; then
            local blocked
            blocked=$(grep "阻塞" "$PROGRESS_FILE" | head -3 || echo "")
            while IFS= read -r line; do
                REPORT_CONTENT="${REPORT_CONTENT}- $line
"
            done <<< "$blocked"
        else
            REPORT_CONTENT="${REPORT_CONTENT}无重大风险
"
        fi
        ;;

    milestone)
        REPORT_CONTENT="# 里程碑报告 - $report_date

## 里程碑状态

"
        # 提取里程碑表格
        local milestone_section
        milestone_section=$(sed -n '/## 2. 里程碑追踪/,/## 3./p' "$PROGRESS_FILE" 2>/dev/null | head -n -1 || echo "")
        if [[ -n "$milestone_section" ]]; then
            REPORT_CONTENT="${REPORT_CONTENT}${milestone_section}
"
        else
            REPORT_CONTENT="${REPORT_CONTENT}无里程碑数据
"
        fi
        ;;

    executive)
        REPORT_CONTENT="# 项目执行摘要 - $report_date

## 关键指标

- **总进度**: $total_progress
- **当前阶段**: $current_phase

## 里程碑状态

"
        # 提取里程碑行
        local milestones
        milestones=$(grep -E "^\\| M[0-9]+:" "$PROGRESS_FILE" 2>/dev/null || grep -E "M[0-9]+:" "$PROGRESS_FILE" 2>/dev/null || echo "")
        if [[ -z "$milestones" ]]; then
            REPORT_CONTENT="${REPORT_CONTENT}无里程碑数据
"
        else
            while IFS= read -r line; do
                REPORT_CONTENT="${REPORT_CONTENT}- $line
"
            done <<< "$milestones"
        fi
        ;;

    *)
        log_error "不支持的报告类型: $REPORT_TYPE"
        pm_json_output "report" "error" "{\"error\": \"不支持的报告类型: $REPORT_TYPE\"}"
        exit 1
        ;;
esac

# 输出报告
echo "$REPORT_CONTENT"

# 如果指定了输出路径，写入文件
if [[ -n "$OUTPUT_PATH" ]]; then
    mkdir -p "$OUTPUT_PATH"
    local output_file
    output_file="$OUTPUT_PATH/report-${REPORT_TYPE}-${report_date}.md"
    echo "$REPORT_CONTENT" > "$output_file"
    log_success "报告已保存: $output_file"
fi

log_success "报告生成完成"
