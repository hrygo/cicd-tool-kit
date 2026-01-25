# Project Manager Skill

AI 项目经理 Skill，用于管理 CICD AI Toolkit 项目的 3 人开发团队。

## 架构 v3.1

```
┌─────────────────────────────────────────────────────────────────┐
│                     Agentic Layer (AI Agent)                    │
│  • 语义理解 • 复杂决策 • 异常处理 • 协调沟通 • 创造性任务        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Scripts Layer (Bash)                         │
│  • 数据操作 • 系统调用 • 原子操作 • 输入验证                    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Data Layer (state.json)                      │
└─────────────────────────────────────────────────────────────────┘
```

## 快速开始

### 查看进度

```bash
.pm/scripts/state.sh progress
```

### 分配任务

```bash
# 1. 检查 Spec 信息
.pm/scripts/state.sh read .specs["CORE-01"]

# 2. 获取锁
.pm/scripts/lock.sh acquire runner dev-a CORE-01 "实现 Runner 生命周期"

# 3. 创建 worktree
.pm/scripts/worktree.sh create dev-a CORE-01

# 4. 分配任务 (自动备份)
.pm/scripts/state.sh assign CORE-01 dev-a
```

### 完成任务

```bash
# 1. 释放锁
.pm/scripts/lock.sh release runner

# 2. 删除 worktree
.pm/scripts/worktree.sh remove dev-a CORE-01

# 3. 完成任务 (自动备份)
.pm/scripts/state.sh complete CORE-01 dev-a
```

## 脚本 API

### state.sh

```bash
.pm/scripts/state.sh read [jq_path]          # 读取状态
.pm/scripts/state.sh assign <spec> <dev>      # 分配任务
.pm/scripts/state.sh complete <spec> <dev>    # 完成任务
.pm/scripts/state.sh unassign <spec> <dev>     # 取消分配
.pm/scripts/state.sh progress                 # 进度统计
.pm/scripts/state.sh health                   # 健康检查
.pm/scripts/state.sh metrics                  # Prometheus 指标
.pm/scripts/state.sh events [n]               # 事件日志
.pm/scripts/state.sh backup                   # 创建备份
.pm/scripts/state.sh restore <backup_id>      # 恢复备份
.pm/scripts/state.sh rollback                 # 回滚到最新
.pm/scripts/state.sh list-backups             # 列出备份
```

### lock.sh

```bash
.pm/scripts/lock.sh acquire <lock> <dev> <spec> [reason] [duration]  # 获取锁
.pm/scripts/lock.sh release <lock>                                # 释放锁
.pm/scripts/lock.sh force-release <lock>                           # 强制释放
.pm/scripts/lock.sh list                                         # 列出所有锁
.pm/scripts/lock.sh check <lock>                                 # 检查锁状态
.pm/scripts/lock.sh prune                                        # 清理过期锁
```

### worktree.sh

```bash
.pm/scripts/worktree.sh create <dev> <spec>  # 创建 worktree
.pm/scripts/worktree.sh remove <dev> <spec>  # 删除 worktree
.pm/scripts/worktree.sh list                 # 列出所有 worktree
.pm/scripts/worktree.sh sync                 # 同步 state.json
```

### ai-suggest.sh

```bash
.pm/scripts/ai-suggest.sh next-task [dev_id]     # 推荐下一任务
.pm/scripts/ai-suggest.sh analyze-blockers        # 分析阻塞
.pm/scripts/ai-suggest.sh optimize-workload       # 优化负载
.pm/scripts/ai-suggest.sh detect-conflicts        # 检测冲突
.pm/scripts/ai-suggest.sh readiness-report        # 准备就绪报告
```

## 团队配置

| 开发者 | 角色 | 命名空间 | 默认锁 |
|--------|------|----------|--------|
| dev-a | Core Platform Engineer | `pkg/runner/`, `pkg/platform/`, `pkg/config/` | runner, config, platform |
| dev-b | Security & Infra Engineer | `pkg/security/`, `pkg/governance/`, `pkg/observability/` | security, governance, observability |
| dev-c | AI & Skills Engineer | `pkg/skill/`, `skills/`, `pkg/mcp/` | skill, mcp |

## 输入验证

- `spec_id`: `^[A-Z]+-[0-9]+$` (如 `CORE-01`)
- `developer_id`: `^dev-[a-z]$` (如 `dev-a`)
- `lock_name`: `^[a-z_]+$` (如 `runner`)

## 锁映射

| Spec 前缀 | 锁名 |
|-----------|------|
| CORE-* | runner |
| CONF-* | config |
| PLAT-* | platform |
| SEC-* | security |
| GOV-* | governance |
| OBS-* | observability |
| SKILL-* / LIB-* | skill |
| MCP-* | mcp |

## 状态转换

```
pending → ready → in_progress → completed
           ↑         ↓
           └───── blocked
```

## 错误处理

### 错误格式

```json
{
  "action": "assign",
  "status": "error",
  "error_code": "E3001",
  "error_name": "LOCK_CONFLICT",
  "error_message": "锁已被占用",
  "context": {...},
  "suggestion": "等待锁释放或强制释放"
}
```

### 错误代码

| 代码 | 名称 | 处理建议 |
|------|------|----------|
| E1002 | INVALID_SPEC_ID | 提示正确格式 (CORE-01) |
| E1003 | INVALID_DEVELOPER_ID | 提示正确格式 (dev-a) |
| E2005 | DEVELOPER_BUSY | 建议先完成当前任务或更换开发者 |
| E2006 | DEPENDENCY_NOT_MET | 分析依赖链，给出等待建议 |
| E3001 | LOCK_CONFLICT | 检查锁状态，建议替代方案 |
| E4003 | STATE_CORRUPTED | 使用 rollback 恢复 |

## 最佳实践

### DO (应该做)

1. **先分析后操作**：调用脚本前先读取状态理解当前情况
2. **提供上下文**：调用脚本时提供有意义的 reason 参数
3. **处理错误**：根据错误代码提供有价值的建议
4. **生成报告**：用自然语言总结复杂情况
5. **主动建议**：发现问题时主动提出解决方案

### DON'T (不应该做)

1. **不要手动解析 JSON**：使用 jq 或让脚本返回结构化数据
2. **不要绕过脚本**：直接操作 state.json 会破坏一致性
3. **不要忽略备份**：关键操作前检查是否有备份
4. **不要硬编码开发者**：动态读取 state.json
5. **不要做脚本的工作**：文件操作、锁管理等交给脚本

## 目录结构

```
.pm/
├── state.json              # 单一真相源
├── state.schema.json       # JSON Schema
├── state.backup.d/         # 自动备份目录
├── scripts/
│   ├── lib/
│   │   ├── lib.sh          # 通用函数库
│   │   ├── validate.sh     # 输入验证
│   │   └── error.sh        # 错误处理
│   ├── state.sh            # 状态操作
│   ├── worktree.sh         # Git worktree
│   ├── lock.sh             # 锁操作
│   ├── ai-suggest.sh       # AI 辅助决策
│   └── gen-task.sh         # 任务生成
├── tests/                  # 测试套件
│   ├── test_lib.sh
│   ├── test_state.sh
│   ├── test_lock.sh
│   ├── test_worktree.sh
│   ├── fixtures/
│   │   └── state.json
│   └── test_runner.sh
└── tasks/                  # 任务卡片 (可选)
```

## 安全性

- 所有输入都经过严格验证
- 临时文件自动清理 (trap EXIT)
- 无路径穿越攻击风险
- 无 SQL/命令注入风险
- 自动备份 + 一键回滚

## 故障排除

### 锁被占用

```bash
# 查看锁状态
.pm/scripts/lock.sh check runner

# 查看锁详情
.pm/scripts/state.sh read .locks["runner"]

# 清理过期锁
.pm/scripts/lock.sh prune

# 强制释放 (谨慎使用)
.pm/scripts/lock.sh force-release runner
```

### Worktree 状态不一致

```bash
# 同步 state.json
.pm/scripts/worktree.sh sync
```

### 状态损坏

```bash
# 回滚到最新备份
.pm/scripts/state.sh rollback

# 或回滚到指定备份
.pm/scripts/state.sh restore 20260125-100000-1234
```

## 测试

```bash
# 运行所有测试
.pm/tests/test_runner.sh

# 运行单个测试套件
.pm/tests/test_lib.sh
.pm/tests/test_state.sh
.pm/tests/test_lock.sh
.pm/tests/test_worktree.sh
```

## 版本

- **当前版本**: 3.2.0
- **state.json 格式**: 1.0

### v3.2 变更

- ✅ 新增 `health` 健康检查命令
- ✅ 新增 `metrics` Prometheus 指标导出
- ✅ 新增 `events` 事件日志查询
- ✅ 优化 `progress` 单次 jq 调用
- ✅ 完善 `force-release` 强制释放锁
- ✅ SKILL.md 渐进式披露重构

## 参考资源

- [SKILL.md](./SKILL.md) - AI 执行指令
- [ARCHITECTURE.md](./ARCHITECTURE.md) - 架构设计
