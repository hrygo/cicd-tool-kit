#!/bin/bash
# test_runner.sh - 主测试运行器

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOTAL_FAILED=0

# 颜色输出
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    NC=''
fi

echo "=========================================="
echo "  Project Manager Skill Test Suite"
echo "=========================================="
echo ""

# 检查依赖
echo "Checking dependencies..."
if ! command -v jq >/dev/null 2>&1; then
    echo -e "${RED}jq not found. Install: brew install jq${NC}"
    exit 1
fi
echo -e "${GREEN}jq found${NC}"

if ! command -v git >/dev/null 2>&1; then
    echo -e "${YELLOW}git not found. Some tests will be skipped.${NC}"
fi
echo ""

# 运行各个测试套件
run_test_suite() {
    local test_file="$1"
    local test_name

    test_name=$(basename "$test_file" .sh)
    test_name="${test_name#test_}"

    echo -e "${YELLOW}Running ${test_name} tests...${NC}"
    echo ""

    if bash "$test_file"; then
        echo -e "${GREEN}${test_name} tests passed${NC}"
    else
        echo -e "${RED}${test_name} tests failed${NC}"
        TOTAL_FAILED=$((TOTAL_FAILED + 1))
    fi
    echo ""
}

# 运行所有测试
for test_file in "$SCRIPT_DIR"/test_*.sh; do
    base_name=$(basename "$test_file")
    if [[ "$base_name" != "test_runner.sh" ]]; then
        run_test_suite "$test_file"
    fi
done

# 总结
echo "=========================================="
echo "  Test Summary"
echo "=========================================="

if [[ $TOTAL_FAILED -eq 0 ]]; then
    echo -e "${GREEN}All test suites passed!${NC}"
    exit 0
else
    echo -e "${RED}${TOTAL_FAILED} test suite(s) failed${NC}"
    exit 1
fi
