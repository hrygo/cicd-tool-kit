# SPEC-PLAT-05: GitHub Composite Actions Design

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24
**Covers**: PRD 1.2 (GitHub Actions 参考模式), 3.7

## 1. 概述 (Overview)

GitHub Composite Actions 允许将多个步骤组合成可复用的工作流单元。本 Spec 定义如何将 `cicd-ai-toolkit` 的不同能力封装为可复用的 Composite Actions，使用户可以灵活组合 AI 能力。

## 2. 核心职责 (Core Responsibilities)

- **Modular Design**: 将不同 Skill 封装为独立可复用的 Action
- **Composition**: 支持组合多个 Actions
- **Versioning**: 语义化版本管理
- **Discovery**: 用户能方便地发现和了解可用 Actions

## 3. 详细设计 (Detailed Design)

### 3.1 Action 层级结构

```
cicd-ai-toolkit/
├── actions/
│   ├── setup/              # 基础环境安装
│   │   └── action.yml
│   ├── review/             # 代码审查
│   │   └── action.yml
│   ├── test-gen/           # 测试生成
│   │   └── action.yml
│   ├── analyze/            # 变更分析
│   │   └── action.yml
│   ├── security-scan/      # 安全扫描
│   │   └── action.yml
│   └── all/                # 全功能组合
│       └── action.yml
```

### 3.2 基础 Setup Action

**用途**: 安装 cicd-ai-toolkit 和准备环境

```yaml
# actions/setup/action.yml
name: 'Setup cicd-ai-toolkit'
description: 'Install and configure cicd-ai-toolkit for AI-powered CI/CD'
author: 'cicd-ai-toolkit'

inputs:
  version:
    description: 'Version of cicd-ai-toolkit to install'
    required: false
    default: 'latest'
  claude-version:
    description: 'Claude Code version to use'
    required: false
    default: 'latest'
  config:
    description: 'Path to configuration file'
    required: false
    default: '.cicd-ai-toolkit.yaml'

outputs:
  runner-path:
    description: 'Path to the cicd-runner binary'
    value: ${{ steps.install.outputs.path }}

runs:
  using: 'composite'
  steps:
    - name: Detect OS
      id: os
      shell: bash
      run: |
        if [[ "$RUNNER_OS" == "Linux" ]]; then
          echo "arch=$(uname -m)" >> $GITHUB_OUTPUT
          echo "platform=linux" >> $GITHUB_OUTPUT
        elif [[ "$RUNNER_OS" == "macOS" ]]; then
          echo "arch=$(uname -m)" >> $GITHUB_OUTPUT
          echo "platform=darwin" >> $GITHUB_OUTPUT
        fi

    - name: Install cicd-runner
      id: install
      shell: bash
      run: |
        VERSION="${{ inputs.version }}"
        [[ "$VERSION" == "latest" ]] && VERSION=$(curl -s https://api.github.com/repos/cicd-ai-toolkit/releases/latest | jq -r .tag_name)

        ARCH="${{ steps.os.outputs.arch }}"
        PLATFORM="${{ steps.os.outputs.platform }}"

        URL="https://github.com/cicd-ai-toolkit/releases/download/${VERSION}/cicd-runner-${PLATFORM}-${ARCH}"
        echo "Installing from $URL"

        curl -fsSL "$URL" -o /usr/local/bin/cicd-runner
        chmod +x /usr/local/bin/cicd-runner

        echo "path=/usr/local/bin/cicd-runner" >> $GITHUB_OUTPUT
        cicd-runner --version

    - name: Validate config
      shell: bash
      run: |
        if [[ -f "${{ inputs.config }}" ]]; then
          cicd-runner validate --config "${{ inputs.config }}"
        else
          echo "No config found, using defaults"
        fi
```

### 3.3 Code Review Action

**用途**: 执行代码审查

```yaml
# actions/review/action.yml
name: 'AI Code Review'
description: 'Perform AI-powered code review using Claude'
author: 'cicd-ai-toolkit'

inputs:
  skills:
    description: 'Comma-separated list of skills to run'
    required: false
    default: 'code-reviewer,change-analyzer'
  severity-threshold:
    description: 'Minimum severity to report (critical, high, medium, low)'
    required: false
    default: 'warning'
  fail-on-error:
    description: 'Fail the workflow if critical issues are found'
    required: false
    default: 'false'
  config:
    description: 'Path to configuration file'
    required: false
    default: '.cicd-ai-toolkit.yaml'
  post-comment:
    description: 'Post results as PR comment'
    required: false
    default: 'true'

outputs:
  issues-found:
    description: 'Number of issues found'
    value: ${{ steps.run.outputs.issues }}
  critical-count:
    description: 'Number of critical issues'
    value: ${{ steps.run.outputs.critical }}
  high-count:
    description: 'Number of high severity issues'
    value: ${{ steps.run.outputs.high }}

runs:
  using: 'composite'
  steps:
    - name: Run AI Review
      id: run
      shell: bash
      env:
        GITHUB_TOKEN: ${{ github.token }}
      run: |
        cicd-runner run \
          --skills "${{ inputs.skills }}" \
          --severity-threshold "${{ inputs.severity-threshold }}" \
          --config "${{ inputs.config }}" \
          --post-comment "${{ inputs.post-comment }}" \
          --output-json > /tmp/cicd-result.json

        # Extract metrics
        echo "issues=$(jq '.total_issues // 0' /tmp/cicd-result.json)" >> $GITHUB_OUTPUT
        echo "critical=$(jq '.issues | map(select(.severity == "critical")) | length' /tmp/cicd-result.json)" >> $GITHUB_OUTPUT
        echo "high=$(jq '.issues | map(select(.severity == "high")) | length' /tmp/cicd-result.json)" >> $GITHUB_OUTPUT

        # Display summary
        jq '.' /tmp/cicd-result.json

    - name: Fail on critical issues
      if: inputs.fail-on-error == 'true' && steps.run.outputs.critical != '0'
      shell: bash
      run: |
        echo "::error::Critical issues found: ${{ steps.run.outputs.critical }}"
        exit 1
```

### 3.4 Test Generator Action

**用途**: 生成测试用例

```yaml
# actions/test-gen/action.yml
name: 'AI Test Generator'
description: 'Generate unit tests based on code changes'
author: 'cicd-ai-toolkit'

inputs:
  target-path:
    description: 'Path to generate tests for (default: all changes)'
    required: false
    default: ''
  test-framework:
    description: 'Test framework to use (auto, jest, pytest, go-test)'
    required: false
    default: 'auto'
  create-pr:
    description: 'Create PR with generated tests'
    required: false
    default: 'true'
  pr-branch-prefix:
    description: 'Branch name prefix for test PRs'
    required: false
    default: 'ai-tests/'

outputs:
  tests-generated:
    description: 'Number of test files generated'
    value: ${{ steps.run.outputs.count }}
  pr-url:
    description: 'URL of created PR (if applicable)'
    value: ${{ steps.pr.outputs.url }}

runs:
  using: 'composite'
  steps:
    - name: Generate Tests
      id: run
      shell: bash
      env:
        GITHUB_TOKEN: ${{ github.token }}
      run: |
        cicd-runner run \
          --skills test-generator \
          --target-path "${{ inputs.target-path }}" \
          --test-framework "${{ inputs.test-framework }}" \
          --output-dir /tmp/generated-tests

        echo "count=$(find /tmp/generated-tests -name '*.test.*' | wc -l)" >> $GITHUB_OUTPUT

    - name: Commit tests
      if: inputs.create-pr == 'true'
      shell: bash
      run: |
        git config user.name "cicd-ai-toolkit"
        git config user.email "ai@cicd-toolkit.com"

        BRANCH="${{ inputs.pr-branch-prefix }}$(date +%s)"
        git checkout -b $BRANCH

        cp -r /tmp/generated-tests/* .
        git add .
        git commit -m "AI: Generate unit tests

        Generated by cicd-ai-toolkit
        Tests: ${{ steps.run.outputs.count }} files"

        git push origin $BRANCH
        echo "branch=$BRANCH" >> $GITHUB_OUTPUT

    - name: Create PR
      if: inputs.create-pr == 'true'
      id: pr
      shell: bash
      env:
        GITHUB_TOKEN: ${{ github.token }}
      run: |
        PR_URL=$(gh pr create \
          --title "AI: Generate unit tests (${{ steps.run.outputs.count }} files)" \
          --body "Automatically generated by cicd-ai-toolkit" \
          --base ${{ github.ref_name }} \
          --head "${{ steps.run.outputs.branch }}")

        echo "url=$PR_URL" >> $GITHUB_OUTPUT
```

### 3.5 变更分析 Action

**用途**: 生成 PR 摘要和影响分析

```yaml
# actions/analyze/action.yml
name: 'AI Change Analyzer'
description: 'Analyze PR changes and generate summary with risk scoring'
author: 'cicd-ai-toolkit'

inputs:
  include-changelog:
    description: 'Include changelog entry in output'
    required: false
    default: 'true'
  risk-threshold:
    description: 'Risk score threshold (0-100) for warnings'
    required: false
    default: '50'

outputs:
  summary:
    description: 'PR summary text'
    value: ${{ steps.analyze.outputs.summary }}
  risk-score:
    description: 'Calculated risk score (0-100)'
    value: ${{ steps.analyze.outputs.risk }}
  labels:
    description: 'Suggested labels (comma-separated)'
    value: ${{ steps.analyze.outputs.labels }}
  changelog:
    description: 'Changelog entry'
    value: ${{ steps.analyze.outputs.changelog }}

runs:
  using: 'composite'
  steps:
    - name: Analyze Changes
      id: analyze
      shell: bash
      env:
        GITHUB_TOKEN: ${{ github.token }}
      run: |
        RESULT=$(cicd-runner run \
          --skills change-analyzer \
          --include-changelog "${{ inputs.include-changelog }}" \
          --output-json)

        echo "summary=$(echo $RESULT | jq -r '.summary')" >> $GITHUB_OUTPUT
        echo "risk=$(echo $RESULT | jq -r '.risk_score')" >> $GITHUB_OUTPUT
        echo "labels=$(echo $RESULT | jq -r '.labels | join(",")')" >> $GITHUB_OUTPUT
        echo "changelog=$(echo $RESULT | jq -r '.changelog')" >> $GITHUB_OUTPUT

        # Warn on high risk
        RISK=$(echo $RESULT | jq -r '.risk_score')
        if (( $(echo "$RISK > ${{ inputs.risk-threshold }}" | bc -l) )); then
          echo "::warning::High risk change detected (score: $RISK)"
        fi

    - name: Apply labels
      shell: bash
      env:
        GITHUB_TOKEN: ${{ github.token }}
        LABELS: ${{ steps.analyze.outputs.labels }}
      run: |
        if [[ -n "$LABELS" ]]; then
          IFS=',' read -ra LABEL_ARRAY <<< "$LABELS"
          for label in "${LABEL_ARRAY[@]}"; do
            gh label add "$label" "$GITHUB_REPOSITORY/pulls/${{ github.event.number }}" 2>/dev/null || true
          done
        fi
```

### 3.6 全功能组合 Action (All-in-One)

```yaml
# actions/all/action.yml
name: 'AI CI/CD Complete'
description: 'Run complete AI-powered CI/CD pipeline'
author: 'cicd-ai-toolkit'

inputs:
  skip-review:
    description: 'Skip code review'
    required: false
    default: 'false'
  skip-tests:
    description: 'Skip test generation'
    required: false
    default: 'false'
  skip-analyze:
    description: 'Skip change analysis'
    required: false
    default: 'false'

runs:
  using: 'composite'
  steps:
    - uses: cicd-ai-toolkit/setup@v1

    - name: Code Review
      if: inputs.skip-review != 'true'
      uses: cicd-ai-toolkit/review@v1
      with:
        fail-on-error: 'true'

    - name: Change Analysis
      if: inputs.skip-analyze != 'true'
      uses: cicd-ai-toolkit/analyze@v1

    - name: Generate Tests
      if: inputs.skip-tests != 'true'
      uses: cicd-ai-toolkit/test-gen@v1
      with:
        create-pr: 'true'
```

### 3.7 Reusable Workflow

除了 Composite Actions，也提供 Reusable Workflow：

```yaml
# .github/workflows/ai-review-reusable.yml
on:
  workflow_call:
    inputs:
      skills:
        description: 'Skills to run'
        required: false
        type: string
        default: 'code-reviewer,change-analyzer'
      fail-on-critical:
        description: 'Fail on critical issues'
        required: false
        type: boolean
        default: true
    secrets:
      github-token:
        required: true
    outputs:
      issues-found:
        description: 'Number of issues found'
        value: ${{ jobs.review.outputs.issues }}

jobs:
  review:
    runs-on: ubuntu-latest
    outputs:
      issues: ${{ steps.run.outputs.issues }}
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: cicd-ai-toolkit/setup@v1

      - name: Run AI Review
        id: run
        env:
          GITHUB_TOKEN: ${{ secrets.github-token }}
        run: |
          cicd-runner run --skills "${{ inputs.skills }}"
```

### 3.8 使用示例

#### 基础用法

```yaml
name: AI Code Review
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  ai-review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: cicd-ai-toolkit/review@v1
        with:
          skills: 'code-reviewer'
          fail-on-error: 'true'
```

#### 组合多个 Actions

```yaml
name: AI Complete Pipeline
on:
  pull_request:

jobs:
  setup:
    runs-on: ubuntu-latest
    steps:
      - uses: cicd-ai-toolkit/setup@v1

  review:
    needs: setup
    runs-on: ubuntu-latest
    steps:
      - uses: cicd-ai-toolkit/review@v1
        with:
          skills: 'code-reviewer,security-scan'

  analyze:
    needs: setup
    runs-on: ubuntu-latest
    steps:
      - uses: cicd-ai-toolkit/analyze@v1

  tests:
    needs: analyze
    runs-on: ubuntu-latest
    if: needs.analyze.outputs.risk-score < 80
    steps:
      - uses: cicd-ai-toolkit/test-gen@v1
```

#### 调用 Reusable Workflow

```yaml
name: My Project CI
on:
  pull_request:

jobs:
  ai-review:
    uses: cicd-ai-toolkit/.github/workflows/ai-review-reusable.yml@v1
    with:
      skills: 'code-reviewer,change-analyzer'
      fail-on-critical: true
    secrets:
      github-token: ${{ secrets.GITHUB_TOKEN }}
```

### 3.9 版本策略

遵循 Semantic Versioning：

| Version | 说明 | 示例 |
|---------|------|------|
| **Major** | 破坏性变更，输入/输出接口变化 | v2.0.0 |
| **Minor** | 新增功能，向后兼容 | v1.1.0 |
| **Patch** | Bug 修复 | v1.0.1 |

发布标签：`v1`, `v1.0`, `v1.0.0`, `latest`

### 3.10 发现机制 (Discovery)

1. **GitHub Marketplace**: 发布到 GitHub Actions Marketplace
2. **README**: 每个 Action 的 README 说明用法
3. **Examples Repository**: 独立的示例仓库

## 4. 依赖关系 (Dependencies)

- **Depends on**: [SPEC-DIST-01](./SPEC-DIST-01-Distribution.md) - Action 分发
- **Related**: [SPEC-SKILL-01](./SPEC-SKILL-01-Skill_Definition.md) - Skill 定义

## 5. 验收标准 (Acceptance Criteria)

1. **独立运行**: 每个 Action 可独立使用
2. **可组合**: 多个 Actions 可组合使用
3. **输入输出**: 输入参数和输出变量正确传递
4. **错误处理**: 失败时有清晰的错误信息
5. **文档完整**: 每个 Action 有清晰的 README
6. **版本管理**: 版本号遵循 SemVer
7. **性能**: Action 启动时间 < 10s (不含 AI 分析)
