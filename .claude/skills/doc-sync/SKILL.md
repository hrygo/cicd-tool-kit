---
name: doc-sync
allowed-tools: Read, Edit, Bash, Grep, Glob, AskUserQuestion, Task
description: 项目文档同步 - 引用完整性 + 代码实现一致性检查
version: 1.0
system: |-
  你是 CICD AI Toolkit 的文档管理员。

  **核心目标**:
  1. 引用完整性 - 确保 Markdown 链接无断链
  2. 代码实现一致性 - 确保文档描述与实际代码同步

  **执行模式**: SCAN → ANALYZE → COMPARE → REPORT → [UPDATE]

  **引用检测模式**:
  - `\[.*\]\(.*\.md\)` — Markdown 链接
  - `@docs/.*\.md` — @ 语法
  - `详见|see|参考.*\.md` — 注释引用

  **文档保鲜策略**:
  - 对比 SKILL.md vs 实际代码实现
  - 对比 PRD vs pkg/ 结构
  - 检查过时/不一致内容

  **安全规则**: 修改前必须展示影响并获用户确认。

  **项目结构**:
  - `docs/` - 项目文档 (PRD.md, CONTRIBUTING.md)
  - `skills/*/SKILL.md` - Skills 定义
  - `.claude/rules/` - Claude 规则
  - `pkg/` - Go 实现代码
---

# 文档同步 Skill (doc-sync)

> **设计哲学**: 文档与代码保持同步，过时的文档比没有文档更糟糕

## 🔄 状态机

```
IDLE ──/docs-sync──▶ SCAN ──▶ ANALYZE ──▶ COMPARE ──▶ REPORT ──▶ DONE
                    │         │           │           │
                    │         │           │           └─ confirm → UPDATE → VERIFY
                    │         │           │           └─ skip → DONE
                    │         │           └─ 无差异 → REPORT (干净)
                    │         └─ 代码分析完成
                    │
                    └─ /docs-check//ref/new ──▶ PLAN ──▶ CONFIRM ──▶ EXECUTE ──▶ VERIFY ──▶ DONE
                                                  │         │          │
                                                  │         │          └─ 失败 → ROLLBACK → IDLE
                                                  │         └─ 拒绝 → IDLE
                                                  └─ 需补充 → SCAN
```

| 状态           | 动作                       | 工具                                    |
| :------------- | :------------------------- | :-------------------------------------- |
| **SCAN**       | 发现目录结构、搜索引用     | `Glob`, `Grep`                          |
| **ANALYZE**    | 深度分析代码实现           | `Task` + `Explore` agent                 |
| **COMPARE**    | 对比文档 vs 实现           | 内置比对逻辑                            |
| **REPORT**     | 生成同步/健康报告          | 输出 Markdown 表格                      |
| **UPDATE**     | 更新文档内容               | `Edit`                                  |
| **PLAN**       | 构建影响图、生成变更清单   | `Read`                                  |
| **CONFIRM**    | 展示影响、获取确认         | `AskUserQuestion`                       |
| **EXECUTE**    | 创建/移动文件、更新引用    | `Bash`, `Edit`                          |
| **VERIFY**     | 验证无断链                 | `Grep`, `Glob`                          |
| **ROLLBACK**   | 回滚变更                   | `Bash` (`git checkout`)                 |

---

## 🎯 命令

### `/docs-check` — 检查文档健康

**状态路径**: `SCAN → REPORT` (只读)

**目标**: 发现断链、孤立文档、索引缺失

**策略**:
1. `Glob("docs/**/*.md")` 扫描结构
2. `Grep` 多模式搜索引用
3. 验证每个引用目标存在
4. 输出健康报告

### `/docs-ref <target>` — 查看引用关系

**状态路径**: `SCAN → REPORT` (只读)

**目标**: 理解文档连接网络

**策略**:
1. `Grep(target, "**/*.md")` 搜索所有引用
2. `Read(target)` 分析其引用的文档
3. 生成双向引用图

### `/docs-sync [scope]` — 文档同步

**状态路径**: `SCAN → ANALYZE → COMPARE → REPORT → [UPDATE]`

**目标**: 对比文档描述与实际代码实现，标记过时内容

**策略**:
1. **SCAN**: 扫描需要检查的文档
2. **ANALYZE**: 使用 `Task` + `Explore` agent 分析代码实现
3. **COMPARE**: 生成差异清单
4. **REPORT**: 输出同步报告
5. **UPDATE** (可选): 经用户确认后更新文档

**scope 选项**:
- `all` (默认) — 检查所有文档
- `skills` — 仅检查 Skills (`skills/*/SKILL.md`)
- `docs` — 仅检查项目文档 (`docs/`)
- `<path>` — 检查指定路径

### `/docs-new <type> <name>` — 创建文档

**状态路径**: `SCAN → PLAN → EXECUTE` (无需 CONFIRM)

**目标**: 在正确位置创建符合规范的文档

**策略**:
1. `Glob` 分析现有结构
2. `Bash` 创建文件
3. 更新相关索引

---

## 📋 文档同步检查项

### Skills 同接

| 检查项       | 验证方法                     |
| :----------- | :--------------------------- |
| 声明的工具   | `allowed-tools` vs 实际使用   |
| 描述的功能   | vs `pkg/` 中实际实现          |
| 平台支持声明 | vs `pkg/platform/` 实现       |

### 项目文档同步

| 文档       | 同步目标                     |
| :--------- | :--------------------------- |
| PRD.md     | vs `pkg/` 目录结构            |
| CONTRIBUTING.md | vs 实际开发流程        |
| README.md  | vs 实际功能特性               |

---

## 📊 报告格式

```markdown
## 文档同步报告

### 断链检查
| 文件   | 链接 | 状态   |
| :----- | :--- | :----- |
| README.md | docs/PRD.md | ✅ 有效 |
| SKILL.md | ../pkg/ai/ | ❌ 断链 |

### 内容过时检查
| 文档   | 声明 | 实际 | 状态 |
| :----- | :--- | :--- | :--- |
| PRD.md | 支持 GitLab | pkg/platform/gitlab.go 不存在 | ⚠️ 过时 |
| skills/test-generator/SKILL.md | 支持 Go 1.21 | go.mod 使用 1.23 | ✅ 同步 |

### 建议
- [ ] 更新 PRD.md 中 GitLab 支持状态
- [ ] 修复 SKILL.md 中的断链
```

---

## 🛠️ 工具使用

| 任务     | 工具   | 示例                         |
| :------- | :----- | :--------------------------- |
| 发现文档 | `Glob` | `docs/**/*.md`, `skills/**/SKILL.md` |
| 搜索引用 | `Grep` | `\[.*\]\(.*\.md\)`           |
| 读取内容 | `Read` | 验证目标存在                 |
| 更新引用 | `Edit` | 精确替换路径                 |
| 搜索代码 | `Grep` | 查找实际实现                 |

---

## ✅ 执行前自检

| 检查项   | 验证方法             | 通过标准 |
| :------- | :------------------- | :------- |
| 引用覆盖 | 使用 ≥3 种 Grep 模式 | 无遗漏   |
| 影响完整 | 反向引用全部发现     | 100%     |
| 路径正确 | 新路径可达性验证     | 存在     |
| 可回滚   | 记录 git 状态        | 有快照   |

---

## ⚠️ 错误恢复

| 错误场景       | 恢复策略                      |
| :------------- | :---------------------------- |
| 引用更新失败   | `git checkout` 回滚受影响文件 |
| 文件移动失败   | 报告错误，保持原状态          |
| 发现新引用格式 | 添加到检测模式，重新扫描      |
| 用户取消       | 无副作用退出                  |

---

## 📖 项目结构

```
cicd-tool-kit/
├── docs/                    # 项目文档
│   ├── PRD.md
│   ├── CONTRIBUTING.md
│   └── architecture/
├── skills/                  # Skills 定义
│   ├── code-reviewer/SKILL.md
│   ├── test-generator/SKILL.md
│   └── ...
├── pkg/                     # Go 实现代码
│   ├── ai/
│   ├── platform/
│   └── ...
└── .claude/
    └── rules/               # Claude 规则
```

---

> **版本**: v1.0 | **理念**: 状态机驱动 + 文档与代码同步
