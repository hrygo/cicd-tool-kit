# Development Guide

## Setup

### 1. Install Git Hooks

After cloning the repository, install the git hooks:

```bash
make install-hooks
# or
./scripts/install-hooks.sh
```

This installs three hooks:
- **pre-commit**: Runs `gofmt`, `go vet`, `go test`, and `staticcheck`
- **pre-push**: Runs full test suite and build check
- **commit-msg**: Validates conventional commit format (warning only)

### 2. Install Tools

```bash
# Install staticcheck (optional but recommended)
go install honnef.co/go/tools/cmd/staticcheck@latest

# Install gofumports (optional, stricter than gofmt)
go install mvdan.cc/gofumpt@latest
```

## Development Workflow

### Making Changes

1. Create a feature branch
2. Make your changes
3. Run tests locally: `make test`
4. Commit - pre-commit hook will run automatically
5. Push - pre-push hook will run full test suite

### Bypassing Hooks (Emergency Only)

```bash
# Bypass pre-commit
git commit --no-verify -m "message"

# Bypass pre-push
GIT_SKIP_PRE_PUSH=1 git push
```

### Available Make Commands

| Command | Description |
|---------|-------------|
| `make install-hooks` | Install git hooks |
| `make lint` | Run all linters |
| `make test` | Run tests (short mode) |
| `make test-full` | Run full test suite |
| `make build` | Build all packages |
| `make check-all` | Run all checks (CI use) |
| `make fmt` | Format Go code |

## Conventional Commits

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style (formatting, etc.)
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Test changes
- `build`: Build system changes
- `ci`: CI configuration changes
- `chore`: Other changes

**Examples:**
```
feat(auth): implement OIDC provider
fix(runner): resolve goroutine leak in cleanup loop
docs: update README with usage examples
refactor(ai): simplify prompt building logic
```

## Testing

### Unit Tests

```bash
# Run short tests (skip integration tests)
make test
# or
go test -short ./...

# Run specific package
go test ./pkg/runner

# Run with verbose output
go test -v ./pkg/runner
```

### Full Test Suite

```bash
make test-full
# or
go test -timeout=5m ./...
```

### Coverage

```bash
# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Code Quality

### Linting

```bash
# Run all linters
make lint

# Individual commands
make fmt    # Format code
make vet    # Run go vet
make staticcheck  # Run staticcheck
```

### Static Analysis

```bash
# staticcheck provides advanced analysis
staticcheck ./...

# Common checks:
# - Unused code
# - Unreachable code
# - Performance issues
# - Potential bugs
```

## Project Structure

```
cicd-tool-kit/
├── cmd/                 # CLI commands
├── pkg/                 # Go packages
│   ├── ai/             # AI brain implementations (Claude, Crush)
│   ├── claude/         # Claude Code CLI integration
│   ├── config/         # Configuration management
│   ├── platform/       # Platform adapters (GitHub, GitLab, Gitee)
│   ├── runner/         # Core CI/CD runner
│   ├── security/       # Security utilities
│   └── ...
├── scripts/            # Utility scripts
├── .githooks/          # Git hooks
└── Makefile            # Build commands
```

## Troubleshooting

### Pre-commit hook fails

1. Check the error message
2. Run the specific check manually: `make vet` or `make test`
3. Fix the issues
4. Try committing again

### Tests fail locally but pass in CI

- Make sure you're using the same Go version
- Clear cache: `go clean -cache -testcache`
- Run `make test-full` to see all tests

### Staticcheck not found

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
```

Make sure `$GOPATH/bin` is in your `$PATH`.
