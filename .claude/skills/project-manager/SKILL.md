---
name: "project-manager"
version: "3.2.0"
description: "AI 项目经理 - 管理 3 人开发团队，state.json 单一真相源"
author: "cicd-ai-toolkit"
license: "MIT"

options:
  thinking:
    budget_tokens: 8000
  temperature: 0.2

tools:
  allow: ["read", "bash"]

inputs:
  - name: action
    type: string
    description: "assign, progress, coordinate, cleanup, suggest"
  - name: developer_id
    type: string
    description: "dev-a, dev-b, dev-c"
  - name: spec_id
    type: string
    description: "CORE-01, SEC-01, etc."
---

# Project Manager Skill v3.2

## 核心原则

```
Scripts = 确定性操作 (读取、验证、执行)
Agent   = 不确定性决策 (分析、判断、建议)
```

| 你 (Agent) | Scripts (Bash) |
|-----------|---------------|
| 分析依赖链 | 读取 state.json |
| 检测冲突 | 验证输入 |
| 制定方案 | 执行原子操作 |
| 生成报告 | 返回 JSON |

## 团队

```
dev-a: runner, config, platform
dev-b: security, governance, observability
dev-c: skill, mcp
```

---

# 核心 API (80% 使用场景)

## 分配任务

```bash
# 1. 读取状态
state.sh read .specs["SPEC_ID"]
state.sh read .developers["dev_id"]

# 2. 你检查: 依赖满足? 锁可用?

# 3. 执行 (事务性)
lock.sh acquire <lock> <dev> <spec> "<reason>"
worktree.sh create <dev> <spec>
state.sh assign <spec> <dev>
```

## 查看进度

```bash
state.sh progress    # JSON 输出 → 你分析 → 自然语言报告
state.sh health      # 健康检查
state.sh events [n]  # 最近事件
```

## 完成任务

```bash
lock.sh release <lock>
worktree.sh remove <dev> <spec>
state.sh complete <spec> <dev>
```

---

# 完整命令索引

### state.sh
```bash
read [jq_path]          # 读取状态
assign <spec> <dev>     # 分配
complete <spec> <dev>   # 完成
unassign <spec> <dev>   # 取消分配
progress                # 进度统计
health                  # 健康检查
metrics                 # Prometheus 指标
events [n]              # 事件日志 (默认 10)
backup                  # 创建备份
restore <id>            # 恢复备份
rollback                # 回滚到最新
list-backups            # 列出备份
```

### lock.sh
```bash
acquire <lock> <dev> <spec> [reason] [duration]
release <lock>
force-release <lock>    # 危险操作
list
check <lock>
prune                   # 清理过期锁
```

### worktree.sh
```bash
create <dev> <spec>
remove <dev> <spec>
list
sync                    # 同步 state.json
```

### ai-suggest.sh
```bash
next-task [dev_id]      # 推荐下一任务
analyze-blockers        # 分析阻塞
optimize-workload       # 负载优化建议
detect-conflicts        # 冲突检测
readiness-report        # 就绪报告
```

---

# 参考数据

## 锁映射

```
CORE-* → runner   | CONF-* → config    | PLAT-* → platform
SEC-*  → security | GOV-* → governance | OBS-* → observability
SKILL-*/LIB-*/MCP-* → skill, mcp
```

## 验证规则

```
spec_id:      ^[A-Z]+-[0-9]+$      (如 CORE-01)
developer_id: ^dev-[a-z]$         (如 dev-a)
lock_name:    ^[a-z_]+$           (如 runner)
```

## 错误代码

```
E1xxx → 输入无效 (提示正确格式)
E2xxx → 状态冲突 (建议替代方案)
E3xxx → 锁冲突   (等待/释放/换任务)
E4xxx → 文件系统 (rollback)
```

## 状态转换

```
pending → ready → in_progress → completed
           ↑         ↓
           └───── blocked
```
