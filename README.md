# CICD AI Toolkit

AI-powered code analysis for CI/CD pipelines. Automate code reviews, test generation, and change analysis using Claude.

## Features

- **Automated Code Review**: Catch bugs and security issues before merge
- **Test Generation**: Generate unit tests from code changes
- **Change Analysis**: Categorize and assess risk of changes
- **Platform Agnostic**: Works with GitHub, GitLab, Gitee, and Jenkins
- **Extensible Skills**: Create custom skills using Markdown
- **Idempotent with Caching**: Same inputs produce same results, fast

## Quick Start

### Installation

```bash
# Using Docker
docker pull ghcr.io/cicd-ai-toolkit/cicd-ai-toolkit:latest

# Or build from source
git clone https://github.com/cicd-ai-toolkit/cicd-ai-toolkit.git
cd cicd-ai-toolkit
make build
```

### GitHub Actions Integration

```yaml
name: AI Review
on:
  pull_request:

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: cicd-ai-toolkit/actions/review@v1
        with:
          api-key: ${{ secrets.ANTHROPIC_API_KEY }}
```

## Configuration

Create `.cicd-ai-toolkit.yaml` in your repository:

```yaml
runner:
  enabled_skills:
    - code-reviewer
    - change-analyzer

platform:
  type: auto

security:
  sandbox_enabled: true
```

## Built-in Skills

| Skill | Description |
|-------|-------------|
| `code-reviewer` | AI-powered code review |
| `test-generator` | Automated test generation |
| `change-analyzer` | Change categorization and risk assessment |
| `log-analyzer` | Log parsing and analysis |
| `issue-triage` | Issue classification and prioritization |

## Documentation

- [Getting Started](docs/guides/getting-started.md)
- [Custom Skills](docs/guides/custom-skills.md)
- [Platform Integration](docs/guides/platform-integration.md)
- [Architecture](docs/architecture/overview.md)
- [Contributing](docs/development/contributing.md)

## Development

```bash
# Install dependencies
make deps

# Run tests
make test

# Run linter
make lint

# Build
make build
```

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## Status

This is an active development project. See [specs/](specs/) for the implementation plan.
