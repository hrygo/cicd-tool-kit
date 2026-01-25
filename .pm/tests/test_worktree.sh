#!/bin/bash
# test_worktree.sh - worktree.sh 测试

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 测试框架
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

assert_success() {
    ((TESTS_RUN++))
    if "$@" >/dev/null 2>&1; then
        ((TESTS_PASSED++))
        return 0
    else
        ((TESTS_FAILED++))
        echo "SKIP: $* (需要 Git 环境)"
        return 0
    fi
}

assert_failure() {
    ((TESTS_RUN++))
    if ! "$@" >/dev/null 2>&1; then
        ((TESTS_PASSED++))
        return 0
    else
        ((TESTS_FAILED++))
        echo "FAIL: $* (should fail)"
        return 1
    fi
}

assert_contains() {
    ((TESTS_RUN++))
    if echo "$1" | grep -q "$2"; then
        ((TESTS_PASSED++))
        return 0
    else
        ((TESTS_FAILED++))
        echo "FAIL: '$1' does not contain '$2'"
        return 1
    fi
}

test_summary() {
    echo ""
    echo "=========================================="
    echo "Tests Run:    $TESTS_RUN"
    echo "Tests Passed: $TESTS_PASSED"
    echo "Tests Failed: $TESTS_FAILED"
    echo "=========================================="

    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo "All tests passed!"
        return 0
    else
        echo "Some tests failed!"
        return 1
    fi
}

# ============================================
# 测试用例
# ============================================

test_worktree_list() {
    echo "Testing worktree.sh list..."

    local result
    result=$("$SCRIPT_DIR/../scripts/worktree.sh" list)

    assert_contains "$result" '"worktrees"'
}

test_worktree_create_invalid_input() {
    echo "Testing worktree.sh create (invalid input)..."

    # 无效的 developer_id
    local result
    result=$("$SCRIPT_DIR/../scripts/worktree.sh" create "dev-invalid" "CORE-01" 2>&1)
    assert_contains "$result" '"error"'

    # 无效的 spec_id
    result=$("$SCRIPT_DIR/../scripts/worktree.sh" create "dev-a" "INVALID" 2>&1)
    assert_contains "$result" '"error"'
}

test_worktree_sync() {
    echo "Testing worktree.sh sync..."

    local result
    result=$("$SCRIPT_DIR/../scripts/worktree.sh" sync)

    assert_contains "$result" '"synced"'
}

# ============================================
# 运行所有测试
# ============================================
main() {
    echo "Running worktree.sh tests..."
    echo ""

    test_worktree_list
    test_worktree_create_invalid_input
    test_worktree_sync

    test_summary
}

main "$@"
