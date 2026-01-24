# Basic Code Review Example

This is a minimal example of using CICD AI Toolkit for code review.

## Setup

1. Copy `.cicd-ai-toolkit.yaml` to your repository
2. Set up `ANTHROPIC_API_KEY` in your CI/CD secrets
3. Add the workflow to your `.github/workflows/`

## Configuration

```yaml
runner:
  enabled_skills:
    - code-reviewer
```

## Usage

```bash
cicd-runner run --skills code-reviewer
```
