# Project Manager Skill - 使用指南

本 Skill 用于管理 CICD AI Toolkit 项目的 3 人 AI 开发团队。

## 前置准备

### 1. 创建 Git Worktrees

```bash
# 在主仓库目录执行
cd /Users/huangzhonghui/cicdtools

# 创建 worktree 目录
mkdir -p ../cicdtools-worktrees

# 为 3 个 AI 开发者创建独立 worktree
git worktree add ../cicdtools-worktrees/worktree-a main
git worktree add ../cicdtools-worktrees/worktree-b main
git worktree add ../cicdtools-worktrees/worktree-c main

# 验证
git worktree list
```

### 2. 初始化项目状态

```bash
# 创建报告目录
mkdir -p reports

# PROGRESS.md 和 COORDINATION_LOG.md 已创建
```

## Skill 使用方法

### Action 1: assign_tasks - 分配任务

为指定开发者分配下一个可执行的任务。

**输入参数**:
```
action: assign_tasks
developer_id: dev-a | dev-b | dev-c
worktree_path: /Users/huangzhonghui/cicdtools-worktrees/
```

**执行流程**:
1. 读取 `specs/PROGRESS.md` 获取当前状态
2. 读取 `specs/IMPLEMENTATION_PLAN.md` 获取依赖关系
3. 查找该开发者负责的、依赖已满足的、优先级最高的任务
4. 生成任务卡片并写入对应 worktree 的 `TASKS.md`
5. 更新 `specs/PROGRESS.md` 状态为 "Assigned"

**输出**: 任务分配结果 JSON

### Action 2: collect_progress - 收集进展

从三个 worktree 收集项目进展。

**输入参数**:
```
action: collect_progress
worktree_path: /Users/huangzhonghui/cicdtools-worktrees/
```

**执行流程**:
1. 读取每个 worktree 的 `TASKS.md`
2. 检查 git commits 和文件变更
3. 更新 `specs/PROGRESS.md`
4. 计算总体进度百分比

**输出**: 进展汇总 JSON

### Action 3: coordinate - 协调资源

处理跨开发者的依赖和冲突。

**输入参数**:
```
action: coordinate
trigger: dependency_ready | interface_change | conflict_detected
spec_id: SPEC-XXX
```

**执行流程**:
1. 识别触发事件的下游影响
2. 通知相关开发者
3. 更新 `specs/COORDINATION_LOG.md`
4. 生成协调建议

**输出**: 协调事项 JSON

### Action 4: report - 生成报告

生成项目进展报告。

**输入参数**:
```
action: report
report_type: weekly | milestone | executive
output_path: reports/
```

**执行流程**:
1. 读取 `specs/PROGRESS.md`
2. 读取 `specs/COORDINATION_LOG.md`
3. 按模板生成 Markdown 报告
4. 保存到 `reports/` 目录

**输出**: 报告文件路径

### Action 5: plan_review - 计划审查

审查实施计划并提出调整建议。

**输入参数**:
```
action: plan_review
```

**执行流程**:
1. 分析当前进展 vs 计划
2. 识别工作量不平衡
3. 检查依赖合理性
4. 提出优化建议

**输出**: 审查发现 JSON

## 典型工作流

### 每日工作流

```bash
# 1. 收集进展
action: collect_progress

# 2. 检查是否有开发者需要新任务
if dev-a.task_completed:
    action: assign_tasks, developer_id: dev-a
if dev-b.task_completed:
    action: assign_tasks, developer_id: dev-b
if dev-c.task_completed:
    action: assign_tasks, developer_id: dev-c

# 3. 检查协调需求
action: coordinate
```

### 周报工作流

```bash
# 1. 收集进展
action: collect_progress

# 2. 生成周报
action: report, report_type: weekly

# 3. 审查计划
action: plan_review
```

### 里程碑工作流

```bash
# 1. 收集进展
action: collect_progress

# 2. 生成里程碑报告
action: report, report_type: milestone

# 3. 里程碑复盘
action: coordinate, trigger: milestone_complete
```

## 开发者工作流程

每个 AI 开发者在其 worktree 中工作：

```bash
# 1. 读取当前任务
cd /Users/huangzhonghui/cicdtools-worktrees/worktree-a
cat TASKS.md

# 2. 执行任务
# (开发、测试、提交)

# 3. 更新任务状态
# 编辑 TASKS.md 中的进度

# 4. 提交代码
git add .
git commit -m "feat(spec): implement SPEC-XXX"

# 5. 请求代码审查
git push origin worktree-a
# (创建 PR)
```

## 文件结构

```
cicdtools/                          # 主仓库
├── specs/
│   ├── IMPLEMENTATION_PLAN.md      # 实施计划
│   ├── PROGRESS.md                 # 进展状态 (PM 维护)
│   └── COORDINATION_LOG.md         # 协调日志 (PM 维护)
│
├── skills/
│   └── project-manager/
│       ├── SKILL.md                # 本 Skill 定义
│       └── README.md               # 本文档
│
├── templates/
│   ├── TASKS_TEMPLATE.md           # 任务卡片模板
│   └── WEEKLY_REPORT_TEMPLATE.md   # 周报模板
│
└── reports/                        # 报告输出 (PM 生成)
    └── progress-YYYY-MM-DD.md

cicdtools-worktrees/                # 工作区
├── worktree-a/                     # dev-a 工作区
│   ├── TASKS.md                    # 当前任务 (PM 生成/维护)
│   └── ...
├── worktree-b/                     # dev-b 工作区
│   ├── TASKS.md
│   └── ...
└── worktree-c/                     # dev-c 工作区
    ├── TASKS.md
    └── ...
```

## 状态机

```
                    assign_tasks()
                        │
                        ▼
                    Pending
                        │
                   (任务分配)
                        │
                        ▼
                   Assigned
                        │
                  (开发者开始)
                        │
                        ▼
                 In Progress
                        │
                  (持续开发)
                        │
        ┌───────────────┴───────────────┐
        ▼                               ▼
     Review                          Blocked
        │                               │
    (PR提交)                      (依赖未满足)
        │                               │
        ▼                               ▼
    Completed ─────────────────► Assigned
        │                           (依赖解除)
        │
        ▼
    (更新 PROGRESS.md)
```

## 注意事项

1. **依赖检查**: 分配任务前必须检查依赖是否完成
2. **避免冲突**: 同一 Spec 只分配给一个开发者
3. **状态同步**: 每次操作后更新 `specs/PROGRESS.md`
4. **协调记录**: 跨开发者事项记录到 `specs/COORDINATION_LOG.md`

## 扩展

### 添加新开发者

1. 在 `SKILL.md` 中更新团队配置
2. 创建新的 worktree
3. 在 `IMPLEMENTATION_PLAN.md` 中分配 Specs

### 添加新 Action

1. 在 `SKILL.md` 中定义 Action
2. 实现分析步骤
3. 定义输出格式
