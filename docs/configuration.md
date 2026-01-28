# Configuration Reference

## Overview

CICD AI Toolkit is configured via YAML configuration files. The configuration file is typically named `.cicd-ai-toolkit.yaml` and placed in the root of your repository.

## Configuration File Structure

```yaml
---
version: "1.0"

# Claude API configuration
claude:
  model: "sonnet"
  max_budget_usd: 5.0
  timeout: 30m

# Skill configuration
skills:
  - name: code-reviewer
    enabled: true
    config:
      severity: "high"

  - name: test-generator
    enabled: false

# Platform configuration
platform:
  github:
    post_comment: true
    draft_mode: false

# Security configuration
security:
  sandbox_enabled: true
  injection_detection: true

# Logging configuration
logging:
  level: info
  format: text
```

## Configuration Sections

### Claude Section

```yaml
claude:
  # Model to use: sonnet, opus, haiku
  model: "sonnet"

  # Maximum budget per execution (USD)
  max_budget_usd: 5.0

  # Execution timeout
  timeout: 30m

  # Maximum tokens per request
  max_tokens: 4096

  # Enable extended thinking
  thinking_enabled: false

  # Thinking budget in tokens
  thinking_budget: 8192
```

### Skills Section

```yaml
skills:
  - name: code-reviewer        # Skill name
    enabled: true              # Enable/disable
    weight: 1.0                # Priority weight
    config:                    # Skill-specific config
      severity: "high"
      categories:
        - security
        - performance

  - name: test-generator
    enabled: true
    config:
      coverage_target: 80
```

### Platform Section

```yaml
platform:
  # GitHub Actions configuration
  github:
    post_comment: true         # Post PR comments
    draft_mode: false          # Create draft PRs
    check_status: true         # Post status checks

  # GitLab CI configuration
  gitlab:
    post_comment: true
    merge_request: true

  # Gitee configuration
  gitee:
    post_comment: true
    pull_request: true

  # Jenkins configuration
  jenkins:
    post_comment: false
```

### Security Section

```yaml
security:
  # Enable sandbox execution
  sandbox_enabled: true

  # Enable prompt injection detection
  injection_detection: true

  # Allowed tools
  allowed_tools:
    - read
    - grep
    - write

  # Denied paths
  denied_paths:
    - /etc/*
    - ~/.ssh/*
```

### Logging Section

```yaml
logging:
  # Log level: debug, info, warn, error
  level: info

  # Log format: text, json
  format: text

  # Log output: stdout, stderr, or file path
  output: stdout
```

## Environment Variables

Configuration can be overridden via environment variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `ANTHROPIC_API_KEY` | Anthropic API key | `sk-ant-...` |
| `CICD_MODEL` | Claude model | `sonnet` |
| `CICD_MAX_BUDGET` | Max budget in USD | `5.0` |
| `CICD_TIMEOUT` | Execution timeout | `30m` |
| `CICD_LOG_LEVEL` | Log level | `debug` |
| `CICD_PLATFORM` | CI/CD platform | `github` |

## Platform-Specific Configuration

### GitHub Actions

```yaml
# .github/workflows/ai-review.yml
name: AI Code Review

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  ai-review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: AI Code Review
        uses: cicd-ai-toolkit/action@v1
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          anthropic_key: ${{ secrets.ANTHROPIC_API_KEY }}
          config: .cicd-ai-toolkit.yaml
```

### GitLab CI/CD

```yaml
# .gitlab-ci.yml
stages:
  - review

ai_review:
  stage: review
  image: cicd-ai-toolkit:latest
  script:
    - cicd-runner review --skills code-reviewer
  only:
    - merge_requests
```

### Gitee Enterprise

```yaml
# .gitee/Pipelines/config.yml
name: AI Review
on:
  - pull_request

steps:
  - name: checkout
    uses: checkout@v1

  - name: ai-review
    uses: cicd-toolkit/action@v1
    with:
      platform: gitee
```

## Skill Configuration

Each skill can have custom configuration:

```yaml
skills:
  - name: code-reviewer
    config:
      # Security settings
      security_scan: true
      vulnerability_check: true

      # Performance settings
      performance_scan: true
      complexity_threshold: 10

      # Output settings
      output_format: markdown
      include_suggestions: true

  - name: test-generator
    config:
      # Test generation settings
      framework: testify
      coverage_target: 80

      # Output settings
      output_path: tests/
      update_existing: true
```

## Validation

Configuration can be validated before deployment:

```bash
cicd-runner validate-config
```

## Examples

### Minimal Configuration

```yaml
---
version: "1.0"

claude:
  model: "sonnet"

skills:
  - name: code-reviewer
    enabled: true
```

### Full Configuration

```yaml
---
version: "1.0"

claude:
  model: "sonnet"
  max_budget_usd: 10.0
  timeout: 45m
  thinking_enabled: true
  thinking_budget: 16384

skills:
  - name: code-reviewer
    enabled: true
    weight: 1.0
    config:
      severity: "medium"
      categories:
        - security
        - performance
        - maintainability

  - name: test-generator
    enabled: true
    weight: 0.8
    config:
      framework: testify
      coverage_target: 80

  - name: change-analyzer
    enabled: true
    weight: 0.9
    config:
      impact_analysis: true
      risk_scoring: true

platform:
  github:
    post_comment: true
    check_status: true
    draft_mode: false

security:
  sandbox_enabled: true
  injection_detection: true
  allowed_tools:
    - read
    - grep
    - write

logging:
  level: debug
  format: json
  output: stdout
```

## Troubleshooting

### Configuration Not Found

```
Error: configuration file not found: .cicd-ai-toolkit.yaml
```

**Solution**: Create the configuration file in the repository root.

### Invalid YAML

```
Error: failed to parse configuration: yaml: line 10: mapping values are not allowed
```

**Solution**: Validate YAML syntax using a linter.

### Unknown Skill

```
Error: skill not found: unknown-skill
```

**Solution**: Check the skill name and ensure it's available in the skills directory.
