# 文件锁机制

## 目的

防止多个 AI 同时修改同一文件造成冲突。

## 快速参考

| 锁文件 | 保护范围 | 谁可以获取 |
|--------|----------|------------|
| `runner.lock` | `pkg/runner/` | dev-a |
| `config.lock` | `pkg/config/` | dev-a |
| `platform.lock` | `pkg/platform/` | dev-a |
| `security.lock` | `pkg/security/` | dev-b |
| `governance.lock` | `pkg/governance/` | dev-b |
| `observability.lock` | `pkg/observability/` | dev-b |
| `skill.lock` | `pkg/skill/`, `skills/` | dev-c |
| `mcp.lock` | `pkg/mcp/` | dev-c |
| `main.lock` | main 分支更新 | 项目经理 |

## 使用脚本

```bash
# 获取锁
.pm/scripts/lock.sh acquire dev-a runner CORE-01 "实现 Runner 生命周期"

# 释放锁
.pm/scripts/lock.sh release runner

# 列出所有锁
.pm/scripts/lock.sh list

# 查看锁状态
.pm/scripts/lock.sh status runner
```

## 锁格式

```yaml
# .pm/locks/{name}.lock
locked_by: dev-a
locked_at: 2026-01-25T10:30:00Z
spec_id: CORE-01
reason: "实现 Runner 生命周期"
files:
  - pkg/runner/lifecycle.go
  - pkg/runner/lifecycle_test.go
expires_at: 2026-01-25T18:00:00Z
```

参见模板: `.pm/templates/lock.lock`

## 超时规则

| 任务类型 | 超时时间 |
|----------|----------|
| 单 Spec 实现 | 6 小时 |
| 文档更新 | 2 小时 |
| main 分支更新 | 30 分钟 |

## 冲突解决

当两个开发者需要修改同一文件时：
1. 先获取锁的开发者继续
2. 后续开发者等待或调整任务顺序
3. 项目经理协调优先级
