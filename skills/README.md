# CICD AI Toolkit - Skills

This directory contains AI-powered skills that analyze code changes and provide automated feedback in CI/CD pipelines.

## Overview

Each skill is defined as a `SKILL.md` file that contains:
- **YAML frontmatter**: Metadata, allowed tools, thinking budget
- **Description**: What the skill does and when to use it
- **Output format**: Structured JSON output for integration
- **MCP tools**: Platform integration hooks

## Available Skills

| Skill | Description | Budget | Tools |
|-------|-------------|--------|-------|
| [code-reviewer](./code-reviewer/) | Security, performance, logic, architecture review | 4096 | Grep, Glob, Read, MCP |
| [change-analyzer](./change-analyzer/) | Impact analysis, risk assessment, changelog | 2048 | Grep, Glob, Read, MCP |
| [test-generator](./test-generator/) | Generate test cases from code changes | 4096 | Grep, Glob, Read, Write, MCP |
| [security-scanner](./security-scanner/) | Security vulnerability scanning | 4096 | Grep, Glob, Read, MCP |
| [perf-auditor](./perf-auditor/) | Performance anti-pattern detection | 3072 | Grep, Glob, Read |
| [log-analyzer](./log-analyzer/) | Log analysis and root cause identification | 2048 | Grep, Read, Glob |

## Usage

### Running via CLI

```bash
# Run code reviewer on current changes
cicd-runner review --skills code-reviewer

# Run multiple skills
cicd-runner review --skills code-reviewer,security-scanner,perf-auditor

# Run on specific PR
cicd-runner review --pr 123 --skills test-generator
```

### Skill Format

Each `SKILL.md` follows this structure:

```yaml
---
name: skill-name
description: Human-readable description
options:
  thinking:
    budget_tokens: 4096
allowed-tools:
  - Grep
  - Read
  - mcp:cicd-toolkit#custom_tool
---

# Skill Name

Detailed description of what this skill does...

## Output Format

```json
{
  "summary": { ... },
  "issues": [ ... ]
}
```
```

## MCP Tools

Skills can integrate with platform via MCP tools:

| Tool | Description |
|------|-------------|
| `get_pr_info(pr_id)` | PR metadata |
| `get_pr_diff(pr_id)` | Code changes diff |
| `get_file_content(path, ref)` | File at specific revision |
| `post_review_comment(pr_id, body, as_review)` | Post results to PR |

## Adding a New Skill

1. Create a new directory: `mkdir skills/your-skill`
2. Create `SKILL.md` with:
   - YAML frontmatter with metadata
   - Clear description of analysis scope
   - Structured output format (JSON)
   - MCP workflow if applicable
3. The skill will be automatically loaded by the runner

## Output Integration

All skills output structured JSON that can be:
- Posted as PR review comments
- Stored in build artifacts
- Parsed by downstream tools
- Displayed in CI logs

Example output:

```json
{
  "summary": {
    "total_issues": 3,
    "critical": 1,
    "high": 1,
    "medium": 1
  },
  "issues": [
    {
      "severity": "critical",
      "category": "security",
      "file": "pkg/auth/auth.go",
      "line": 42,
      "message": "SQL injection vulnerability",
      "suggestion": "Use parameterized queries"
    }
  ]
}
```

## Configuration

Skills can be enabled/disabled in `.cicd-ai-toolkit.yaml`:

```yaml
skills:
  - name: code-reviewer
    enabled: true
    options:
      severity_threshold: "medium"

  - name: security-scanner
    enabled: true
    options:
      owasp_top_10: true
```
