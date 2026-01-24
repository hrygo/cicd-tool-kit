# Platform Integration

## Supported Platforms

| Platform | Status | Notes |
|----------|--------|-------|
| GitHub | ✅ Full support | Native integration |
| GitLab | ✅ Full support | Via platform adapter |
| Gitee | ✅ Full support | Via platform adapter |
| Jenkins | 🚧 Planned | Via plugin |

## GitHub Actions

### Using Composite Actions

```yaml
- uses: cicd-ai-toolkit/actions/setup@v1
  with:
    api-key: ${{ secrets.ANTHROPIC_API_KEY }}

- uses: cicd-ai-toolkit/actions/review@v1
  with:
    api-key: ${{ secrets.ANTHROPIC_API_KEY }}
```

### Manual Integration

```yaml
- name: Run AI Review
  run: |
    cicd-runner run --skills code-reviewer
  env:
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## GitLab CI

```yaml
ai_review:
  stage: test
  image: ghcr.io/cicd-ai-toolkit/cicd-ai-toolkit:latest
  script:
    - cicd-runner run --skills code-reviewer
  variables:
    ANTHROPIC_API_KEY: $ANTHROPIC_API_KEY
    GITLAB_TOKEN: $CI_JOB_TOKEN
```

## Gitee Go

```yaml
ai_review:
  runs-on: ubuntu-latest
  steps:
    - name: Checkout
      uses: actions/checkout@v4
    - name: AI Review
      uses: cicd-ai-toolkit/actions/review@v1
```

## Jenkins

```groovy
pipeline {
    agent any
    stages {
        stage('AI Review') {
            steps {
                docker.image('ghcr.io/cicd-ai-toolkit/cicd-ai-toolkit:latest').inside {
                    sh 'cicd-runner run --skills code-reviewer'
                }
            }
        }
    }
    environment {
        ANTHROPIC_API_KEY = credentials('anthropic-api-key')
    }
}
```
