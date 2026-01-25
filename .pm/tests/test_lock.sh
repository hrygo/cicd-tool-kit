#!/bin/bash
# test_lock.sh - lock.sh 测试

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
        echo "FAIL: $*"
        return 1
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

test_lock_acquire_valid() {
    echo "Testing lock.sh acquire (valid)..."

    local result
    result=$("$SCRIPT_DIR/../scripts/lock.sh" acquire "test_lock_$$" "dev-a" "TEST-01" "test reason")

    assert_contains "$result" '"success"'
    assert_contains "$result" '"lock_name"'

    # 清理
    "$SCRIPT_DIR/../scripts/lock.sh" release "test_lock_$$" >/dev/null 2>&1 || true
}

test_lock_acquire_invalid_name() {
    echo "Testing lock.sh acquire (invalid name)..."

    local result
    result=$("$SCRIPT_DIR/../scripts/lock.sh" acquire "InvalidLock" "dev-a" "TEST-01" "test reason" 2>&1)

    assert_contains "$result" '"error"'
    assert_contains "$result" '"invalid"'
}

test_lock_acquire_duplicate() {
    echo "Testing lock.sh acquire (duplicate)..."

    "$SCRIPT_DIR/../scripts/lock.sh" acquire "test_lock_dup_$$" "dev-a" "TEST-01" >/dev/null

    local result
    result=$("$SCRIPT_DIR/../scripts/lock.sh" acquire "test_lock_dup_$$" "dev-b" "TEST-02" 2>&1)

    assert_contains "$result" '"error"'

    # 清理
    "$SCRIPT_DIR/../scripts/lock.sh" release "test_lock_dup_$$" >/dev/null 2>&1 || true
}

test_lock_release() {
    echo "Testing lock.sh release..."

    "$SCRIPT_DIR/../scripts/lock.sh" acquire "test_lock_rel_$$" "dev-a" "TEST-01" >/dev/null

    local result
    result=$("$SCRIPT_DIR/../scripts/lock.sh" release "test_lock_rel_$$")

    assert_contains "$result" '"success"'
}

test_lock_list() {
    echo "Testing lock.sh list..."

    local result
    result=$("$SCRIPT_DIR/../scripts/lock.sh" list)

    assert_contains "$result" '"locks"'
}

test_lock_check() {
    echo "Testing lock.sh check..."

    "$SCRIPT_DIR/../scripts/lock.sh" acquire "test_lock_check_$$" "dev-a" "TEST-01" >/dev/null

    local result
    result=$("$SCRIPT_DIR/../scripts/lock.sh" check "test_lock_check_$$")

    assert_contains "$result" '"locked": true'

    "$SCRIPT_DIR/../scripts/lock.sh" release "test_lock_check_$$" >/dev/null

    result=$("$SCRIPT_DIR/../scripts/lock.sh" check "test_lock_check_$$")
    assert_contains "$result" '"locked": false'
}

# ============================================
# 运行所有测试
# ============================================
main() {
    echo "Running lock.sh tests..."
    echo ""

    test_lock_acquire_valid
    test_lock_acquire_invalid_name
    test_lock_acquire_duplicate
    test_lock_release
    test_lock_list
    test_lock_check

    test_summary
}

main "$@"
