# Getting Started

## Installation

### Using Docker

```bash
docker pull ghcr.io/cicd-ai-toolkit/cicd-ai-toolkit:latest
```

### Building from Source

```bash
git clone https://github.com/cicd-ai-toolkit/cicd-ai-toolkit.git
cd cicd-ai-toolkit
make build
```

## Quick Start

### 1. Create Configuration

Create `.cicd-ai-toolkit.yaml` in your repository:

```yaml
runner:
  enabled_skills:
    - code-reviewer

platform:
  type: auto
```

### 2. Set API Key

```bash
export ANTHROPIC_API_KEY="your-api-key"
```

### 3. Run Analysis

```bash
cicd-runner run --skills code-reviewer
```

## GitHub Actions Integration

Add to `.github/workflows/ai-review.yml`:

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

## Next Steps

- Read [Custom Skills Guide](custom-skills.md)
- Explore [Platform Integration](platform-integration.md)
- Check [Configuration Examples](../../configs/)
