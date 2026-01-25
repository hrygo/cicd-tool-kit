#!/bin/bash
# test_lib.sh - 公共库函数测试

# 测试框架
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# 断言函数
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

assert_eq() {
    ((TESTS_RUN++))
    if [[ "$1" == "$2" ]]; then
        ((TESTS_PASSED++))
        return 0
    else
        ((TESTS_FAILED++))
        echo "FAIL: '$1' != '$2'"
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

# 测试结果输出
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

# 加载被测试的库
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../scripts/lib/lib.sh"
source "$SCRIPT_DIR/../scripts/lib/validate.sh"

# ============================================
# 测试用例
# ============================================

test_pm_validate_spec_id() {
    echo "Testing pm_validate_spec_id..."

    # 有效的 spec_id
    assert_success pm_validate_spec_id "CORE-01" "test"
    assert_success pm_validate_spec_id "ABC-123" "test"
    assert_success pm_validate_spec_id "TEST-999" "test"

    # 无效的 spec_id
    assert_failure pm_validate_spec_id "core-01" "test"
    assert_failure pm_validate_spec_id "CORE01" "test"
    assert_failure pm_validate_spec_id "CORE-01-EXTRA" "test"
    assert_failure pm_validate_spec_id "" "test"
}

test_pm_validate_developer_id() {
    echo "Testing pm_validate_developer_id..."

    # 有效的 developer_id
    assert_success pm_validate_developer_id "dev-a" "test"
    assert_success pm_validate_developer_id "dev-b" "test"
    assert_success pm_validate_developer_id "dev-c" "test"
    assert_success pm_validate_developer_id "dev-z" "test"

    # 无效的 developer_id
    assert_failure pm_validate_developer_id "dev-1" "test"
    assert_failure pm_validate_developer_id "dev-A" "test"
    assert_failure pm_validate_developer_id "devel-a" "test"
    assert_failure pm_validate_developer_id "" "test"
}

test_pm_validate_lock_name() {
    echo "Testing pm_validate_lock_name..."

    # 有效的 lock_name
    assert_success pm_validate_lock_name "runner" "test"
    assert_success pm_validate_lock_name "config" "test"
    assert_success pm_validate_lock_name "my_lock" "test"
    assert_success pm_validate_lock_name "a" "test"

    # 无效的 lock_name
    assert_failure pm_validate_lock_name "Runner" "test"
    assert_failure pm_validate_lock_name "runner-lock" "test"
    assert_failure pm_validate_lock_name "" "test"
}

test_pm_now_iso() {
    echo "Testing pm_now_iso..."

    local now
    now=$(pm_now_iso)

    # 检查格式: YYYY-MM-DDTHH:MM:SSZ
    assert_contains "$now" "^[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}T[0-9]\{2\}:[0-9]\{2\}:[0-9]\{2\}Z$"
}

test_pm_timestamp_parse() {
    echo "Testing pm_timestamp_parse..."

    local ts
    ts=$(pm_timestamp_parse "2026-01-25T10:00:00Z")

    # 应该返回一个正整数
    assert_contains "$ts" "^[0-9]\+$"
}

# ============================================
# 运行所有测试
# ============================================
main() {
    echo "Running lib.sh tests..."
    echo ""

    test_pm_validate_spec_id
    test_pm_validate_developer_id
    test_pm_validate_lock_name
    test_pm_now_iso
    test_pm_timestamp_parse

    test_summary
}

main "$@"
