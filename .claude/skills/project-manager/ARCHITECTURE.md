# Project Manager - 架构设计 v3.1

## 核心原则

> **Scripts 处理确定性操作，Agent 处理不确定性决策**

### 职责边界矩阵

| 维度 | Scripts (确定性强) | Agent (智能性强) |
|------|-------------------|-----------------|
| **输入** | 结构化数据 (JSON) | 自然语言 + 上下文 |
| **处理** | 单一逻辑路径 | 多因素权衡 |
| **输出** | 标准化 JSON | 语义化决策 |
| **失败** | 明确错误码 | 可解释原因 |
| **特点** | 幂等、可重试 | 上下文感知 |

### 核心设计理念

```
┌─────────────────────────────────────────────────────────────────┐
│                     Agentic Layer (AI Agent)                    │
│                                                                  │
│  职责：                                                          │
│  • 语义理解：理解用户意图和项目上下文                            │
│  • 复杂决策：多目标权衡 (优先级、资源、依赖)                      │
│  • 异常处理：诊断问题并提出解决方案                              │
│  • 协调沟通：跨开发者协调、冲突解决                              │
│                                                                  │
│  不做：                                                          │
│  ✗ 不执行文件操作 (由脚本完成)                                   │
│  ✗ 不解析结构化数据 (由 jq 完成)                                 │
│  ✗ 不执行原子性操作 (由脚本保证)                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Scripts Layer (Bash)                         │
│                                                                  │
│  职责：                                                          │
│  • 数据操作：JSON 读写、状态管理                                 │
│  • 系统调用：Git、文件系统                                       │
│  • 原子操作：锁管理、事务                                        │
│  • 输入验证：格式检查、注入防护                                  │
│                                                                  │
│  特点：                                                          │
│  • 幂等性：多次调用结果一致                                      │
│  • 可回滚：失败时恢复原状态                                      │
│  • 可测试：纯函数式设计                                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Data Layer (state.json)                      │
│                                                                  │
│  • 单一真相源                                                    │
│  • JSON Schema 约束                                              │
│  • 事务保护 (flock)                                              │
└─────────────────────────────────────────────────────────────────┘
```

## 详细职责划分

### 1. 任务分配 (assign)

| 步骤 | 负责方 | 原因 |
|------|--------|------|
| 读取 Spec 信息 | Scripts | 结构化数据读取 |
| 检查依赖完整性 | **Agent** | 需要理解依赖语义 |
| 检查开发者可用性 | Scripts | 简单状态查询 |
| 检查锁冲突 | Scripts | 结构化检查 |
| 评估优先级 | **Agent** | 需要业务判断 |
| 获取锁 | Scripts | 原子操作 |
| 创建 worktree | Scripts | 系统调用 |
| 更新状态 | Scripts | 数据操作 |
| 生成 TASK.md | **Agent** | 创造性工作 |

### 2. 进度收集 (progress)

| 步骤 | 负责方 | 原因 |
|------|--------|------|
| 统计原始数据 | Scripts | 聚合计算 |
| 识别阻塞原因 | **Agent** | 需要分析 |
| 生成建议 | **Agent** | 需要推理 |
| 识别风险 | **Agent** | 需要预测 |

### 3. 资源协调 (coordinate)

| 步骤 | 负责方 | 原因 |
|------|--------|------|
| 读取所有锁状态 | Scripts | 数据查询 |
| 分析资源冲突 | **Agent** | 需要优化 |
| 生成协调方案 | **Agent** | 需要谈判 |
| 执行锁调整 | Scripts | 原子操作 |

### 4. 异常恢复 (recover)

| 步骤 | 负责方 | 原因 |
|------|--------|------|
| 诊断异常 | **Agent** | 需要理解 |
| 提出恢复方案 | **Agent** | 需要决策 |
| 执行恢复 | Scripts | 确定性操作 |
| 验证恢复 | Scripts | 状态检查 |

## 脚本 API 设计

### 设计原则

1. **单一职责**：每个脚本只做一件事
2. **幂等性**：多次调用结果一致
3. **可回滚**：失败时能恢复原状态
4. **可测试**：纯函数式设计
5. **JSON 优先**：结构化输入输出

### state.sh - 状态操作

| 命令 | 输入 | 输出 | 职责 |
|------|------|------|------|
| `read [path]` | jq 路径 | JSON 值 | 读取状态 |
| `update <path> <value>` | 路径, JSON 值 | 更新后的值 | 更新状态 |
| `assign <spec> <dev>` | spec_id, dev_id | {spec, developer} | 分配任务 |
| `complete <spec> <dev>` | spec_id, dev_id | {spec, developer} | 完成任务 |
| `rollback <backup_id>` | 备份 ID | {restored} | 回滚状态 |
| `progress` | - | 统计 JSON | 计算进度 |
| `validate` | - | {valid, errors} | 验证状态 |
| `backup` | - | {backup_id} | 创建备份 |
| `restore <backup_id>` | 备份 ID | {restored} | 恢复备份 |

### worktree.sh - Git Worktree

| 命令 | 输入 | 输出 | 职责 |
|------|------|------|------|
| `create <dev> <spec>` | dev_id, spec_id | {path, branch} | 创建 worktree |
| `remove <dev> <spec>` | dev_id, spec_id | {removed} | 删除 worktree |
| `list` | - | [{worktree}] | 列出 worktree |
| `sync` | - | {actual_count, state_count} | 同步状态 |
| `verify <path>` | 路径 | {valid, errors} | 验证完整性 |

### lock.sh - 锁管理

| 命令 | 输入 | 输出 | 职责 |
|------|------|------|------|
| `acquire <lock> <dev> <spec>` | lock_name, dev_id, spec_id | {lock} | 获取锁 |
| `release <lock>` | lock_name | {released} | 释放锁 |
| `list` | - | {locks} | 列出锁 |
| `check <lock>` | lock_name | {locked, info} | 检查锁 |
| `prune` | - | {pruned, locks} | 清理过期锁 |
| `force-release <lock>` | lock_name | {released} | 强制释放 |

### ai-suggest.sh - AI 辅助决策 (新增)

| 命令 | 输入 | 输出 | 职责 |
|------|------|------|------|
| `next-task <dev>` | dev_id | {suggestions} | 推荐下一任务 |
| `analyze-blockers` | - | {blockers, solutions} | 分析阻塞 |
| `optimize-workload` | - | {rebalances} | 优化负载 |
| `detect-conflicts` | - | {conflicts, resolutions} | 检测冲突 |

## 数据结构

### state.json 完整格式

```json
{
  "version": "1.0",
  "updated_at": "2026-01-25T10:00:00Z",
  "backups": [],

  "developers": {
    "dev-a": {
      "id": "dev-a",
      "name": "Core Platform Engineer",
      "namespace": ["pkg/runner/", "pkg/platform/", "pkg/config/"],
      "current_task": null,
      "completed_specs": ["PLAT-07", "CONF-01"],
      "worktree": null,
      "branch": null,
      "stats": {
        "assigned_count": 5,
        "completed_count": 2,
        "avg_duration_hours": 4.5
      }
    }
  },

  "specs": {
    "CORE-01": {
      "id": "CORE-01",
      "name": "Runner Lifecycle",
      "status": "ready",
      "dependencies": ["CONF-01", "SKILL-01"],
      "assignee": "dev-a",
      "priority": "p0",
      "phase": "Phase 2",
      "estimated_hours": 4,
      "blocked_by": [],
      "blocking": ["CORE-03", "PLAT-01"]
    }
  },

  "locks": {
    "runner": {
      "locked_by": "dev-a",
      "locked_at": "2026-01-25T10:00:00Z",
      "spec_id": "CORE-01",
      "reason": "实现 Runner 生命周期",
      "expires_at": "2026-01-25T18:00:00Z"
    }
  },

  "worktrees": [
    {
      "developer": "dev-a",
      "spec_id": "CORE-01",
      "path": "/Users/huangzhonghui/.worktree/pr-a-CORE-01",
      "branch": "pr-a-CORE-01",
      "created_at": "2026-01-25T10:00:00Z",
      "status": "active"
    }
  ],

  "milestones": {},

  "metadata": {
    "total_specs": 30,
    "completed_specs": 3,
    "in_progress_specs": 3,
    "blocked_specs": 1,
    "progress_percentage": 10
  }
}
```

## 状态机

```
                    ┌─────────────┐
                    │   pending   │  初始状态
                    └──────┬──────┘
                           │
                      检查依赖 (Agent)
                           │
                ┌──────────┴──────────┐
                ▼                     ▼
           ┌─────────┐          ┌──────────┐
           │  ready  │          │ blocked  │  依赖未满足
           └────┬────┘          └──────────┘
                │
           assign() (Agent 决策 + Scripts 执行)
                │
                ▼
          ┌─────────────┐
          │ in_progress │  开发中
          └──────┬──────┘
                │
             PR合并 (Agent 验证)
                │
                ▼
          ┌─────────────┐
          │ completed   │  已完成
          └─────────────┘
```

## Agent 工作流

### assign 任务流程

```python
def assign_task(spec_id, developer_id):
    # ========== Agent 决策层 ==========
    # 1. 读取 state.json
    state = read_state()

    # 2. Agent 分析：检查依赖 (Agent 负责)
    deps_status = check_dependencies(state, spec_id)
    if not deps_status.all_completed:
        return Agent.format_error(
            "依赖未满足",
            missing=deps_status.missing,
            suggestion=f"等待 {deps_status.missing} 完成后重试"
        )

    # 3. Agent 分析：检查开发者状态
    developer = state.developers[developer_id]
    if developer.current_task:
        return Agent.format_error(
            "开发者有进行中任务",
            current_task=developer.current_task,
            suggestion="先完成当前任务或重新分配"
        )

    # 4. Agent 分析：检查锁冲突并推荐解决方案
    lock_name = infer_lock_from_spec(spec_id)
    if lock_conflict(lock_name):
        return Agent.suggest_alternatives(
            conflict=lock_name,
            alternatives=find_available_specs(developer_id)
        )

    # ========== Scripts 执行层 ==========
    # 5. 获取锁 (脚本保证原子性)
    lock_result = run_script("lock.sh", "acquire", lock_name, developer_id, spec_id)
    if lock_result.status != "success":
        return Agent.handle_lock_failure(lock_result)

    # 6. 创建 worktree
    worktree_result = run_script("worktree.sh", "create", developer_id, spec_id)
    if worktree_result.status != "success":
        # 回滚：释放锁
        run_script("lock.sh", "release", lock_name)
        return Agent.handle_worktree_failure(worktree_result)

    # 7. 分配任务
    assign_result = run_script("state.sh", "assign", spec_id, developer_id)
    if assign_result.status != "success":
        # 回滚：释放锁 + 删除 worktree
        run_script("lock.sh", "release", lock_name)
        run_script("worktree.sh", "remove", developer_id, spec_id)
        return Agent.handle_assign_failure(assign_result)

    # ========== Agent 总结层 ==========
    return Agent.format_success(
        action="assign",
        spec=assign_result.data.spec,
        developer=assign_result.data.developer,
        worktree=worktree_result.data.path,
        next_steps=[
            f"1. 进入 worktree: cd {worktree_result.data.path}",
            f"2. 查看任务: cat TASK.md",
            f"3. 开始开发"
        ]
    )
```

### progress 收集流程

```python
def collect_progress():
    # ========== Scripts 执行层 ==========
    # 1. 获取原始统计数据
    stats = run_script("state.sh", "progress")
    locks = run_script("lock.sh", "list")
    worktrees = run_script("worktree.sh", "list")

    # ========== Agent 分析层 ==========
    # 2. Agent 分析：识别可分配的 Spec
    ready_specs = []
    blocked_specs = []

    for spec_id, spec in state.specs.items():
        if spec.status == "completed":
            continue

        # Agent 判断：依赖是否满足
        deps_ok = Agent.check_dependencies(spec)
        if deps_ok:
            ready_specs.append(spec_id)
        else:
            # Agent 分析：为什么阻塞？
            blockers = Agent.get_blockers(spec)
            blocked_specs.append({
                "spec": spec_id,
                "blocked_by": blockers,
                "estimate": Agent.estimate_wait_time(blockers)
            })

    # 3. Agent 生成：优先级建议
    recommendations = Agent.prioritize_tasks(
        ready_specs,
        developer_workload=state.developers,
        business_priority=state.specs
    )

    # 4. Agent 检测：风险预警
    risks = Agent.detect_risks(
        blocked_specs=blocked_specs,
        in_progress=state.specs.filter(status="in_progress"),
        locks=locks
    )

    return {
        "summary": stats.data.summary,
        "ready_specs": ready_specs,
        "blocked_specs": blocked_specs,
        "recommendations": recommendations,
        "risks": risks,
        "narrative": Agent.generate_progress_report(...)  # 自然语言报告
    }
```

## 错误处理策略

### Scripts 层错误

```bash
# 所有脚本返回统一格式
{
  "action": "assign",
  "status": "error",
  "error_code": "LOCK_CONFLICT",     # 机器可读
  "error_message": "锁已被占用",      # 人类可读
  "context": {...},                  # 上下文信息
  "suggestion": "等待锁释放或强制释放",  # 建议
  "timestamp": "2026-01-25T10:00:00Z"
}
```

### Agent 层错误处理

```python
class AgentErrorHandler:
    ERROR_CODES = {
        "LOCK_CONFLICT": "handle_lock_conflict",
        "DEPENDENCY_MISSING": "handle_dependency_missing",
        "DEVELOPER_BUSY": "handle_developer_busy",
        "WORKTREE_EXISTS": "handle_worktree_exists",
    }

    def handle(self, error):
        handler = getattr(self, self.ERROR_CODES.get(error.code, "handle_generic"))
        return handler(error)

    def handle_lock_conflict(self, error):
        # Agent 分析：是否可以等待？是否可以强制释放？
        lock = error.context["lock"]
        if lock.is_expired():
            return self.suggest_prune_lock(lock)
        else:
            return self.suggest_alternative_tasks()
```

## 目录结构

```
.pm/
├── state.json              # 单一真相源
├── state.schema.json       # JSON Schema
├── state.backup.d/         # 自动备份目录
│   ├── 20260125-100000.json
│   └── 20260125-110000.json
├── scripts/
│   ├── lib/
│   │   ├── lib.sh          # 通用函数库
│   │   ├── validate.sh     # 输入验证
│   │   └── error.sh        # 错误处理 (新增)
│   ├── state.sh            # 状态操作
│   ├── worktree.sh         # Git worktree
│   ├── lock.sh             # 锁操作
│   ├── ai-suggest.sh       # AI 辅助决策 (新增)
│   └── gen-task.sh         # 任务生成
├── tests/                  # 测试套件 (新增)
│   ├── test_lib.sh
│   ├── test_state.sh
│   ├── test_lock.sh
│   ├── test_worktree.sh
│   ├── test_ai_suggest.sh
│   ├── fixtures/
│   │   └── state.json
│   └── test_runner.sh
└── tasks/                  # (可选) 人类可读的任务卡片
```

## 测试策略

### 单元测试

```bash
# 测试单个函数
test_pm_validate_spec_id() {
    assert_success pm_validate_spec_id "CORE-01"
    assert_failure pm_validate_spec_id "invalid"
}
```

### 集成测试

```bash
# 测试完整工作流
test_assign_workflow() {
    # 创建测试状态
    setup_test_state

    # 执行分配
    result=$(run_script "state.sh" "assign" "CORE-01" "dev-a")
    assert_eq "$result.status" "success"

    # 验证状态
    assert_eq $(read_state ".specs.CORE-01.status") "in_progress"
    assert_eq $(read_state ".developers.dev-a.current_task") "CORE-01"

    # 清理
    cleanup_test_state
}
```

## 性能优化

1. **单次 jq 调用**：合并多个查询
2. **增量更新**：只更新变化的字段
3. **索引优化**：为常用查询添加索引字段
4. **缓存**：Agent 缓存常用状态

## 安全性

1. **输入验证**：白名单正则
2. **路径防护**：防止路径穿越
3. **原子操作**：flock 保证
4. **备份恢复**：自动备份 + 一键恢复
5. **审计日志**：所有操作可追溯

## 迁移指南

从 v3.0 迁移到 v3.1：

1. **备份现有状态**：`.pm/scripts/state.sh backup`
2. **更新脚本**：替换为新版本
3. **验证兼容性**：`.pm/scripts/state.sh validate`
4. **运行测试**：`.pm/tests/test_runner.sh`

## 总结

| 问题 | v3.0 | v3.1 |
|------|------|------|
| 职责划分 | 部分重叠 | 明确边界 |
| 事务支持 | 无 | 备份/回滚 |
| 错误处理 | 简单 | 分层处理 |
| AI 集成 | 基础 | 深度融合 |
| 测试覆盖 | 0% | 目标 80% |
| 性能 | 多次 jq | 优化后 |
| 可维护性 | 中等 | 高 |
