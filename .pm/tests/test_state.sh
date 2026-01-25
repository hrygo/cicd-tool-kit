#!/bin/bash
# test_state.sh - state.sh 测试

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

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

assert_json_eq() {
    ((TESTS_RUN++))
    local actual="$1"
    local expected="$2"
    local actual_val
    local expected_val
    actual_val=$(echo "$actual" | jq -r "$expected")
    expected_val=$(echo "$expected" | jq -r "$expected")
    if [[ "$actual_val" == "$expected_val" ]]; then
        ((TESTS_PASSED++))
        return 0
    else
        ((TESTS_FAILED++))
        echo "FAIL: JSON path '$expected' - got '$actual_val', expected '$expected_val'"
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

# 设置测试环境
setup_test_state() {
    TEST_STATE_FILE="/tmp/pm-test-state-$$.json"
    export STATE_FILE="$TEST_STATE_FILE"
    export STATE_FILE_TMP="${TEST_STATE_FILE}.tmp"
    export STATE_FILE_LOCK="${TEST_STATE_FILE}.lock"
    export PM_BACKUP_DIR="${TEST_STATE_FILE}.backup.d"

    # 复制测试 fixture
    cp "$SCRIPT_DIR/fixtures/state.json" "$TEST_STATE_FILE"

    # 初始化锁文件
    touch "$STATE_FILE_LOCK"
    exec 9>"$STATE_FILE_LOCK"
}

cleanup_test_state() {
    rm -f "$TEST_STATE_FILE" "$TEST_STATE_FILE.tmp" "$TEST_STATE_FILE.lock"
    rm -rf "$PM_BACKUP_DIR"
}

# ============================================
# 测试用例
# ============================================

test_state_read() {
    echo "Testing state.sh read..."

    local result
    result=$("$SCRIPT_DIR/../scripts/state.sh" read .version)
    assert_eq "$result" '"1.0"'

    result=$("$SCRIPT_DIR/../scripts/state.sh" read .updated_at)
    assert_contains "$result" "^[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}T"
}

test_state_read_spec() {
    echo "Testing state.sh read spec..."

    local result
    result=$("$SCRIPT_DIR/../scripts/state.sh" read .specs["CORE-01"].status)
    assert_eq "$result" '"in_progress"'

    result=$("$SCRIPT_DIR/../scripts/state.sh" read .specs["CORE-01"].name)
    assert_contains "$result" "Runner"
}

test_state_read_developer() {
    echo "Testing state.sh read developer..."

    local result
    result=$("$SCRIPT_DIR/../scripts/state.sh" read .developers["dev-a"].name)
    assert_contains "$result" "Core Platform"

    result=$("$SCRIPT_DIR/../scripts/state.sh" read .developers["dev-a"].current_task)
    assert_eq "$result" '"CORE-01"'
}

test_state_progress() {
    echo "Testing state.sh progress..."

    local result
    result=$("$SCRIPT_DIR/../scripts/state.sh" progress)

    assert_contains "$result" '"action"'
    assert_contains "$result" '"progress"'
    assert_contains "$result" '"summary"'
}

test_state_validate() {
    echo "Testing state.sh validate..."

    local result
    result=$("$SCRIPT_DIR/../scripts/state.sh" validate)

    assert_contains "$result" '"valid"'
    assert_contains "$result" '"validate"'
}

test_state_backup() {
    echo "Testing state.sh backup..."

    local result
    result=$("$SCRIPT_DIR/../scripts/state.sh" backup)

    assert_contains "$result" '"success"'
    assert_contains "$result" '"backup_id"'
    assert_contains "$result" '"path"'

    # 检查备份文件是否创建
    local backup_id
    backup_id=$(echo "$result" | jq -r '.data.backup_id')
    assert_success test -f "$PM_BACKUP_DIR/$backup_id.json"
}

test_state_list_backups() {
    echo "Testing state.sh list-backups..."

    # 先创建一个备份
    "$SCRIPT_DIR/../scripts/state.sh" backup >/dev/null

    local result
    result=$("$SCRIPT_DIR/../scripts/state.sh" list-backups)

    assert_contains "$result" '"backups"'
}

test_state_update() {
    echo "Testing state.sh update..."

    # 创建测试状态
    setup_test_state

    local result
    result=$("$SCRIPT_DIR/../scripts/state.sh" update .test_key '"test_value"')

    assert_contains "$result" '"test_value"'

    cleanup_test_state
}

# ============================================
# 运行所有测试
# ============================================
main() {
    echo "Running state.sh tests..."
    echo ""

    test_state_read
    test_state_read_spec
    test_state_read_developer
    test_state_progress
    test_state_validate
    test_state_backup
    test_state_list_backups
    test_state_update

    test_summary
}

main "$@"
