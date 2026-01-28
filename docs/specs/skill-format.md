# Skill Format Specification

## Overview

Skills are the primary extension mechanism for CICD AI Toolkit. Each skill is a Markdown file with YAML frontmatter that defines the skill's metadata, inputs, and behavior.

## File Structure

Skills are stored in the `skills/` directory:

```
skills/
├── code-reviewer/
│   └── SKILL.md
├── test-generator/
│   └── SKILL.md
└── change-analyzer/
    └── SKILL.md
```

## Frontmatter Schema

```yaml
---
name: skill-name              # Required: Skill identifier
version: 1.0.0                # Optional: Semantic version
description: Skill description # Required: Human-readable description
author: Author Name           # Optional: Skill author
license: Apache-2.0            # Optional: License identifier
thinking_enabled: true        # Optional: Enable thinking mode (default: false)
max_turns: 10                 # Optional: Maximum conversation turns
output_format: json           # Optional: Output format (json, markdown, text)
budget_usd: 0.50              # Optional: Max budget in USD
tools:                        # Optional: Allowed tools
  - grep
  - read
inputs:                       # Optional: Input parameters
  - name: string (required): Parameter name
  - count: int (default: 5): Parameter count
---

# Skill Content

The skill prompt goes here...
```

## Metadata Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Skill identifier (slug format) |
| `description` | string | Yes | Human-readable description |
| `version` | string | No | Semantic version (e.g., 1.0.0) |
| `author` | string | No | Author name or organization |
| `license` | string | No | License identifier (e.g., Apache-2.0) |
| `thinking_enabled` | boolean | No | Enable thinking mode (default: false) |
| `max_turns` | int | No | Maximum conversation turns |
| `output_format` | string | No | Output format: json, markdown, text |
| `budget_usd` | float | No | Maximum budget in USD |
| `tools` | list | No | List of allowed tools |
| `budget_tokens` | int | No | Thinking budget in tokens |

## Input Specification

Inputs can be specified in two formats:

### Inline Format

```yaml
inputs:
  - path: string (required): File path to analyze
  - depth: int (default: 3): Search depth
```

### Nested Format

```yaml
inputs:
  - name: repository
    type: string
    description: Repository URL
    required: true
  - name: branch
    type: string
    description: Branch name
    required: false
    default: main
```

### Input Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Parameter name |
| `type` | string | Yes | Parameter type (string, int, float, bool) |
| `description` | string | No | Parameter description |
| `required` | boolean | No | Whether parameter is required |
| `default` | any | No | Default value if not required |

## Content Format

The content section uses standard Markdown with the following conventions:

1. **Role Definition**: Start with a clear role statement
2. **Instructions**: Provide step-by-step instructions
3. **Output Format**: Specify expected output format
4. **Examples**: Include examples when helpful

## Example Skill

```markdown
---
name: security-scanner
version: 1.2.0
description: Scan code for security vulnerabilities
author: Security Team
license: Apache-2.0
thinking_enabled: true
budget_tokens: 8192
tools:
  - grep
  - read
  - write
inputs:
  - path: string (required): Path to scan
  - severity: string (default: medium): Minimum severity level
---

# Security Scanner

You are a security expert. Analyze the code at the given path for:
1. SQL injection vulnerabilities
2. XSS vulnerabilities
3. Authentication/authorization issues
4. Sensitive data exposure

## Output Format

Provide results in JSON format:

```json
{
  "issues": [
    {
      "severity": "high",
      "type": "sql-injection",
      "file": "src/db.go",
      "line": 42,
      "description": "Unparameterized query"
    }
  ]
}
```
```

## Best Practices

1. **Clear Naming**: Use descriptive, lowercase names with hyphens
2. **Version Management**: Update version when changing behavior
3. **Input Validation**: Clearly specify required vs optional inputs
4. **Tool Restrictions**: Only specify tools that are actually needed
5. **Output Format**: Always specify expected output format
6. **Examples**: Provide examples for complex skills
