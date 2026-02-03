# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## Essential Commands

```bash
# Build
make build              # Build all packages
go build ./...          # Alternative

# Lint & Format
make lint               # gofmt + vet + staticcheck
make fmt                # Format code
golangci-lint run --config=.golangci.yaml  # Full lint (used by pre-push)

# Test
make test               # Short mode (no integration tests)
make test-full          # Full test suite
go test ./pkg/ai        # Test specific package
go test -run TestX ./... # Run specific test

# Install git hooks (recommended)
make install-hooks      # pre-commit (~2s), pre-push (~1min)

# Run the CLI
go run ./cmd/cicd-runner review --skills code-reviewer
```

---

## Architecture Overview

**CICD AI Toolkit = Go Runner + AI Skills + Platform Integration**

```
┌─────────────────────────────────────────────────────────────┐
│                        cicd-tool-kit                        │
│  ┌───────────────────────────────────────────────────────┐   │
│  │              Runner (Go) - pkg/runner/                │   │
│  │  - Orchestrates AI execution                          │   │
│  │  - Manages platform API clients                       │   │
│  │  - Builds git context for analysis                    │   │
│  └───────────────────────────────────────────────────────┘   │
│                           │                                 │
│                           ▼                                 │
│  ┌───────────────────────────────────────────────────────┐   │
│  │              AI Layer - pkg/ai/, pkg/claude/          │   │
│  │  - Factory pattern for multiple backends              │   │
│  │  - Claude Code CLI integration (default)               │   │
│  │  - Session management & caching                        │   │
│  └───────────────────────────────────────────────────────┘   │
│                           │                                 │
│                           ▼                                 │
│  ┌───────────────────────────────────────────────────────┐   │
│  │         Skills - skills/* (Markdown-defined)          │   │
│  │  - code-reviewer: Security, performance, logic        │   │
│  │  - test-generator: Generate tests from diffs          │   │
│  │  - change-analyzer: Impact analysis, summaries        │   │
│  │  - log-analyzer: Log parsing, anomaly detection       │   │
│  └───────────────────────────────────────────────────────┘   │
│                           │                                 │
│                           ▼                                 │
│  ┌───────────────────────────────────────────────────────┐   │
│  │      Platform Layer - pkg/platform/                   │   │
│  │  - GitHub, GitLab, Gitee API clients                  │   │
│  │  - Webhook handling - pkg/webhook/                    │   │
│  └───────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

**Key Insight**: Skills are defined as Markdown files in `skills/*/SKILL.md`. The Runner loads them and passes to Claude Code CLI as context. No recompilation needed to add/modify skills.

---

## Key Packages

| Package | Responsibility |
|---------|---------------|
| `pkg/runner/` | Core orchestration, skill execution, result aggregation |
| `pkg/ai/` | AI backend factory (Claude, Crush), prompt building, output parsing |
| `pkg/claude/` | Claude Code CLI session management, subprocess handling |
| `pkg/buildcontext/` | Git diff extraction, context building for analysis |
| `pkg/platform/` | Platform API clients (GitHub, GitLab, Gitee) |
| `pkg/skill/` | Skill loader, parses `SKILL.md` files into structured definitions |
| `pkg/config/` | YAML configuration loading and validation |
| `pkg/observability/` | Metrics, logging, structured output |
| `pkg/perf/` | Performance utilities (worker pools, rate limiting, Map/Filter/Each helpers) |
| `pkg/security/` | Security scanning, sandbox for command execution |

---

## Development Workflow

**Branch Protection is enabled on `main`**:
- All changes must go through PR
- Requires 1 approval
- Linear history enforced (squash merge)
- `golangci-lint` status check required

**Git Hooks**:
- **pre-commit** (~2s): `gofmt`, `go vet`, `go mod tidy` check
- **pre-push** (~1min): `golangci-lint`, `go test`
- Install with: `make install-hooks`

**Branch Naming**: `<type>/<issue-id>-<description>`
- `feat/123-add-async-mode`
- `fix/456-session-leak`
- `refactor/789-cleanup`

**Commit Format**:
```
<type>(<scope>): <subject>

Refs #<issue-id>

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

Types: `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `chore`

---

## Project Structure Notes

```
cicd-tool-kit/
├── cmd/                      # CLI entry points
│   ├── cicd-runner/          # Main CLI (Cobra-based)
│   └── mcp-server/           # MCP server for tool integration
│
├── pkg/                      # Core libraries (see table above)
│
├── skills/                   # Product capabilities (what users get)
│   ├── code-reviewer/SKILL.md
│   ├── test-generator/SKILL.md
│   └── ...
│
├── .claude/                  # Claude-specific (development helpers)
│   ├── skills/               # Skills for building this project
│   │   ├── feature-designer/ # Helps design new features
│   │   └── doc-sync/         # Keeps docs in sync with code
│   └── rules/
│       └── git-workflow.md   # Git workflow rules
│
├── docs/                     # Project documentation
│   └── PRD.md                # Product requirements
│
├── configs/                  # Configuration examples
├── scripts/                  # Build/install scripts
├── .githooks/                # Git hook scripts
└── .github/                  # GitHub-specific (workflows, templates)
```

**Distinction**:
- `skills/` → Product features (what CICD Toolkit offers to users)
- `.claude/skills/` → Development helpers (tools for building CICD Toolkit)

---

## Skills Format

Each skill in `skills/<name>/SKILL.md` follows this structure:

```yaml
---
name: skill-name
allowed-tools: [Read, Edit, Bash, Grep, ...]
description: Human-readable description
options:
  thinking:
    budget_tokens: 4096
---
```

**MCP Tools Available to Skills**:
- `get_pr_info(pr_id)` - PR metadata
- `get_pr_diff(pr_id)` - Code changes
- `get_file_content(path, ref)` - File content at revision
- `post_review_comment(pr_id, body, as_review)` - Post review

---

## Configuration

Config is YAML-based (`.cicd-ai-toolkit.yaml`):

```yaml
version: "1.0"

claude:
  model: "sonnet"           # claude, opus, sonnet, haiku
  max_budget_usd: 5.0       # Cost control

skills:
  - name: code-reviewer
    enabled: true

platform:
  github:
    post_comment: true
```

Config loading: `pkg/config/` validates and loads into structs.

---

## Platform Integration

**Key Pattern**: Each platform has its own sub-package in `pkg/platform/`:

- `github/` - GitHub Actions / GitHub API
- `gitlab/` - GitLab CI/CD
- `gitee/` - Gitee Enterprise

Each implements platform-specific logic for:
- Fetching PR/diff data
- Posting review comments
- Webhook handling (via `pkg/webhook/`)

---

## Testing Patterns

- **Table-driven tests** for multiple scenarios
- **Mock external dependencies** using interfaces
- **Coverage targets**: Core logic >70%, tools >80%

```go
func TestReview(t *testing.T) {
    tests := []struct {
        name    string
        diff    string
        wantErr bool
    }{
        {"empty diff", "", true},
        {"valid code", "package main\n\nfunc main(){}", false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

---

## Important Conventions

1. **Package names**: lowercase, single word (e.g., `runner`, `ai`, not `Runner` or `runner_pkg`)
2. **Exported functions**: Always have godoc comments
3. **Error wrapping**: Use `fmt.Errorf("context: %w", err)` for wrapping
4. **Interfaces**: Define at the usage side, not "just in case"
5. **Context**: First parameter in all exported functions
6. **Configuration structs**: Use `yaml`, `json`, and `env` tags

---

## Linter Configuration

`.golangci.yaml` has extensive path-based exclusions:
- Test files: relaxed error checking
- MCP generated code: `pkg/mcp/`
- Platform client code: `pkg/platform/`
- Cleanup/teardown patterns: many intentionally ignored errors

When adding new files, consider if they fit existing exclusion patterns.
