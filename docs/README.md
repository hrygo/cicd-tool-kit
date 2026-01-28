# CICD AI Toolkit Documentation

Welcome to the CICD AI Toolkit documentation. This toolkit is an enterprise-grade CI/CD automation solution powered by Anthropic Claude.

## Quick Links

- [Getting Started](#getting-started)
- [Configuration](configuration.md)
- [Skills Development](development/skills.md)
- [API Reference](api/README.md)
- [Architecture](architecture/README.md)

## Getting Started

### Installation

```bash
go install github.com/cicd-ai-toolkit/cicd-runner@latest
```

### Basic Usage

```bash
# Review code changes
cicd-runner review --skills code-reviewer

# Generate tests
cicd-runner test-generate --skill test-generator

# Analyze changes
cicd-runner analyze --skills change-analyzer
```

## Documentation Structure

```
docs/
├── README.md              # This file
├── configuration.md       # Configuration reference
├── cli.md                 # CLI command reference
├── security.md            # Security best practices
├── troubleshooting.md     # Common issues and solutions
├── api/                   # API documentation
│   └── README.md
├── architecture/          # Architecture documentation
│   ├── README.md
│   └── overview.md
├── development/           # Development guide
│   ├── README.md
│   ├── skills.md          # Skills development guide
│   └── testing.md
├── guides/                # User guides
│   ├── github-actions.md
│   ├── gitlab-ci.md
│   └── gitee-ci.md
└── specs/                 # Feature specifications
    ├── skill-format.md
    └── platform-support.md
```

## Core Concepts

### Skills

Skills are the building blocks of the toolkit. Each skill is defined in Markdown with YAML frontmatter:

```markdown
---
name: code-reviewer
version: 1.0.0
description: Review code for security and performance issues
author: CICD AI Toolkit
license: Apache-2.0
tools:
  - grep
  - read
inputs:
  - path: string (required): Path to review
  - depth: int (default: 3): Search depth
---

# Code Review Skill

You are a code reviewer. Analyze the code for...
```

### Platform Support

- GitHub Actions
- GitLab CI/CD
- Gitee Enterprise
- Jenkins
- Azure Pipelines
- Local execution

## Contributing

See [Development Guide](development/README.md) for details on contributing to the toolkit.

## License

Apache License 2.0
