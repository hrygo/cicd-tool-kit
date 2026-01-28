# Skills Development Guide

## Overview

This guide covers how to create, test, and deploy custom skills for CICD AI Toolkit.

## Creating a Skill

### 1. Create Skill Directory

```bash
mkdir -p skills/my-skill
touch skills/my-skill/SKILL.md
```

### 2. Define Frontmatter

```markdown
---
name: my-skill
version: 1.0.0
description: Brief description of what this skill does
author: Your Name
tools:
  - grep
  - read
inputs:
  - target: string (required): Target to process
---
```

### 3. Write Skill Content

The content section is where you define the AI's behavior:

```markdown
# My Custom Skill

You are an expert in [domain]. Your task is to [goal].

## Steps

1. First, [step 1]
2. Then, [step 2]
3. Finally, [step 3]

## Output Format

Provide output in the following format:

```
[Expected output format]
```
```

## Skill Patterns

### Analysis Skill

For analyzing code, logs, or configurations:

```markdown
---
name: performance-analyzer
description: Analyze code for performance issues
tools:
  - grep
  - read
---

# Performance Analyzer

Analyze the given code for performance issues:

1. **Algorithmic Complexity**: Look for O(n²) or worse
2. **Memory Usage**: Check for memory leaks or excessive allocation
3. **I/O Operations**: Identify inefficient file/database operations
4. **Concurrency**: Check for race conditions or deadlocks

Report issues with:
- Severity level (critical/high/medium/low)
- File location and line number
- Suggested fix
```

### Generator Skill

For generating code, tests, or documentation:

```markdown
---
name: test-generator
description: Generate unit tests for Go code
tools:
  - read
  - write
inputs:
  - file: string (required): File to generate tests for
---

# Test Generator

Generate comprehensive unit tests for the given Go file.

For each function:
1. Test the happy path
2. Test edge cases
3. Test error conditions
4. Add table-driven tests when applicable

Use standard Go testing conventions and testify assertions.
```

### Transformation Skill

For modifying or refactoring code:

```markdown
---
name: go-upgrade
description: Upgrade Go code to latest version
tools:
  - read
  - write
  - bash
---

# Go Upgrade Helper

Upgrade Go modules to their latest compatible versions.

Steps:
1. Run `go get -u ./...`
2. Run `go mod tidy`
3. Check for breaking changes
4. Update deprecated API usage
```

## Testing Skills

### Local Testing

```bash
# Test with a specific file
cicd-runner test-generate --skill my-skill --file src/main.go

# Test with verbose output
cicd-runner review --skills my-skill --verbose
```

### Unit Testing

Create a test file `skills/my-skill/SKILL_test.md`:

```markdown
---
name: my-skill-test
description: Test cases for my-skill
---

# Test Cases

## Input: [test input]
Expected Output: [expected output]

## Input: [test input]
Expected Output: [expected output]
```

## Advanced Features

### Thinking Mode

Enable for complex reasoning:

```yaml
---
thinking_enabled: true
budget_tokens: 8192
---
```

### Tool Restrictions

Limit available tools for security:

```yaml
---
tools:
  - read      # Allow reading files
  - grep      # Allow searching
  # No write or bash for safety
---
```

### Budget Control

Limit API usage:

```yaml
---
budget_usd: 0.50
max_turns: 5
---
```

## Publishing Skills

### 1. Version Your Skill

Update the version in frontmatter when making changes:

```yaml
version: 1.0.0  # Major.Minor.Patch
```

### 2. Document Your Skill

Create a README.md in the skill directory:

```markdown
# My Skill

## Description
Brief description of what the skill does.

## Usage
```bash
cicd-runner review --skills my-skill
```

## Inputs
- `target`: Target to process

## Outputs
Description of output format.

## Examples
Example usage and output.
```

### 3. Share Your Skill

Skills can be shared by:
1. Committing to the main repository
2. Creating a separate repository with skills
3. Sharing the SKILL.md file directly

## Troubleshooting

### Skill Not Found

Ensure the skill directory structure is correct:

```
skills/
└── my-skill/
    └── SKILL.md
```

### Tools Not Available

Check that the required tools are listed in the frontmatter. Available tools:
- `read`: Read file contents
- `write`: Write to files
- `grep`: Search file contents
- `bash`: Execute shell commands
- `edit`: Edit files

### Output Format Issues

Always specify the expected output format in the skill content. Use examples to clarify.

## Best Practices

1. **Be Specific**: Clear instructions produce better results
2. **Provide Examples**: Show expected input/output
3. **Use Tools**: Leverage available tools effectively
4. **Set Budgets**: Prevent runaway API usage
5. **Version Carefully**: Follow semantic versioning
6. **Document Well**: Help others understand your skill
7. **Test Thoroughly**: Verify skill behavior before deploying
