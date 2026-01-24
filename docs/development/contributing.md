# Contributing

## Development Setup

```bash
# Clone repository
git clone https://github.com/cicd-ai-toolkit/cicd-ai-toolkit.git
cd cicd-ai-toolkit

# Install dependencies
go mod download

# Run tests
go test ./...

# Run linter
golangci-lint run

# Build
go build -o bin/cicd-runner ./cmd/cicd-runner
```

## Project Structure

```
cicd-ai-toolkit/
├── cmd/              # Executable entry points
├── pkg/              # Core library packages
├── skills/           # Built-in skills
├── configs/          # Configuration examples
├── actions/          # GitHub Actions
├── docs/             # Documentation
├── test/             # Tests
└── build/            # Build scripts
```

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add comments for exported types
- Write tests for new functionality

## Submitting Changes

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run `make lint` and `make test`
6. Submit a pull request

## RFC Process

For significant changes, submit an RFC first:

1. Create RFC in `docs/rfcs/`
2. Open discussion PR
3. Get approval from maintainers
4. Implement
