# Architecture Overview

## System Architecture

CICD AI Toolkit follows a modular architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                         CLI Layer                           │
│                      (cmd/cicd-runner)                      │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                       Runner Layer                          │
│                    (pkg/runner)                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │ Orchestrator│  │  Executor   │  │   Result Handler    │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                      Platform Layer                         │
│                    (pkg/platform)                           │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌────────┐ ┌──────┐  │
│  │ GitHub  │ │ GitLab  │ │  Gitee  │ │Jenkins│ │ Local│  │
│  └─────────┘ └─────────┘ └─────────┘ └────────┘ └──────┘  │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                       Skill Layer                           │
│                     (pkg/skill)                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │   Loader    │  │   Parser    │  │   Skill Registry    │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                     Security Layer                          │
│                   (pkg/security)                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │  Sandbox     │  │  Injection   │  │  Path Validator  │  │
│  │  Execution   │  │  Detector    │  │                  │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                     Claude Layer                            │
│                     (pkg/claude)                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │   Client    │  │   Parser    │  │  Thinking Support   │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. CLI Layer (`cmd/`)

- **Entry Point**: `main.go`
- **Command Definition**: Cobra-based CLI commands
- **Flag Parsing**: Command-line argument handling

### 2. Runner Layer (`pkg/runner/`)

- **Orchestrator**: Manages skill execution flow
- **Executor**: Executes skills with Claude API
- **Result Handler**: Processes and formats results

### 3. Platform Layer (`pkg/platform/`)

- **Detection**: Auto-detects CI/CD platform
- **Adapters**: Platform-specific integration
- **Configuration**: Platform-specific settings

### 4. Skill Layer (`pkg/skill/`)

- **Loader**: Discovers and loads skill definitions
- **Parser**: Parses YAML frontmatter
- **Registry**: Manages available skills

### 5. Security Layer (`pkg/security/`)

- **Sandbox**: Secure command execution
- **Injection Detection**: Prompt injection protection
- **Path Validation**: File system access control

### 6. Claude Layer (`pkg/claude/`)

- **Client**: Anthropic API client
- **Parser**: Response parsing
- **Thinking**: Extended thinking support

## Data Flow

```
User Input
    │
    ▼
CLI Command
    │
    ▼
Platform Detection ──► Load Platform Config
    │
    ▼
Skill Discovery ──► Load Skills
    │
    ▼
Security Validation ──► Check Permissions
    │
    ▼
Claude API Call ──► Execute Skill
    │
    ▼
Result Processing
    │
    ▼
Output (Comment/Report/File)
```

## Key Design Decisions

### 1. Go Runner + Python Skills

The runner is implemented in Go for:
- Performance and efficiency
- Easy deployment (single binary)
- Strong typing and concurrency

Skills are in Markdown for:
- Human-readable and editable
- No compilation required
- Easy version control

### 2. Platform Abstraction

Platform-specific logic is isolated in adapters:
- Easy to add new platforms
- Consistent interface across platforms
- Platform auto-detection

### 3. Security First

- Sandbox execution for untrusted code
- Prompt injection detection
- Path validation and access control
- Budget limits (USD and tokens)

### 4. Extensibility

- Plugin-based skill system
- Configuration-driven behavior
- API for custom integrations

## Concurrency Model

The runner uses goroutines for:
- Concurrent skill execution
- Parallel file processing
- Non-blocking API calls

Synchronization primitives:
- `sync.WaitGroup` for task coordination
- `sync.RWMutex` for shared state
- Channels for communication

## Error Handling

- Wrapped errors with context
- Structured error types
- Retry logic for API calls
- Graceful degradation

## Observability

- Structured logging
- Metrics collection (planned)
- Distributed tracing (planned)
- Performance monitoring

## Security Model

### Defense in Depth

1. **Input Validation**: All inputs are validated
2. **Sandbox**: Commands run in restricted environment
3. **Injection Detection**: Prompt patterns are checked
4. **Path Validation**: File access is restricted
5. **Budget Limits**: API usage is capped

### Trust Boundaries

- Untrusted: User input, skill content
- Semi-trusted: Configuration, platform data
- Trusted: Runner code, security modules
