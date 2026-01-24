# Architecture Overview

## System Design

CICD AI Toolkit is designed as a modular, extensible platform for AI-powered CI/CD analysis.

## Components

```
┌─────────────────────────────────────────────────────────────┐
│                        CI/CD Platform                        │
│  (GitHub Actions, GitLab CI, Gitee Go, Jenkins)             │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                      CICD AI Toolkit                         │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                     Runner                           │   │
│  │  • Lifecycle Management                             │   │
│  │  • Process Management                               │   │
│  │  • Task Execution                                   │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   Build Context                      │   │
│  │  • Git Diff Analysis                                │   │
│  │  • Context Chunking                                 │   │
│  │  • Context Pruning                                  │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   Skill Engine                       │   │
│  │  • Skill Loader                                     │   │
│  │  • Skill Registry                                   │   │
│  │  • Skill Executor                                   │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   Claude Integration                 │   │
│  │  • Process Management                               │   │
│  │  • Stream Handling                                  │   │
│  │  • Output Parsing                                   │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                      Skills                                  │
│  • code-reviewer   • test-generator   • change-analyzer     │
│  • log-analyzer    • issue-triage                           │
└─────────────────────────────────────────────────────────────┘
```

## Key Design Principles

1. **Platform Agnostic**: Works across GitHub, GitLab, Gitee, and Jenkins
2. **Extensible Skills**: Custom skills via simple markdown files
3. **Idempotent**: Same inputs produce same outputs with caching
4. **Secure**: Sandboxed execution with tool whitelisting
5. **Observable**: Full logging, metrics, and audit trails

## Module Structure

| Module | Purpose |
|--------|---------|
| `runner` | Core execution engine |
| `platform` | CI/CD platform adapters |
| `buildctx` | Context building and chunking |
| `claude` | Claude AI integration |
| `skill` | Skill management |
| `config` | Configuration handling |
| `cache` | Two-level caching |
| `output` | Result formatting |
| `security` | RBAC and sandboxing |
| `observability` | Logging and metrics |
