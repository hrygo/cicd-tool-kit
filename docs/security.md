# Security Guide

## Overview

CICD AI Toolkit is designed with security as a core principle. This document covers security features, best practices, and considerations for deploying the toolkit in enterprise environments.

## Security Features

### 1. Prompt Injection Detection

The toolkit includes built-in protection against prompt injection attacks:

```go
detector := security.NewPromptInjectionDetector()

// Validate user input
if err := detector.Validate(userInput); err != nil {
    return fmt.Errorf("invalid input: %w", err)
}

// Scan for suspicious patterns
result := detector.Scan(prompt)
if result.IsSuspicious {
    log.Warn("Suspicious prompt detected", "score", result.Score)
}
```

Protected patterns include:
- Instruction override attempts
- Role playing attacks
- System prompt extraction
- Delimiter manipulation

### 2. Sandbox Execution

Commands are executed in a restricted sandbox:

```go
sandbox := security.NewSandbox(nil)

// Validate tool
if !sandbox.ValidateTool("rm") {
    return errors.New("dangerous tool blocked")
}

// Validate path
if err := sandbox.ValidatePath("/etc/passwd"); err != nil {
    return err
}

// Run with limits
result, err := sandbox.Run(ctx, cmd)
```

Sandbox restrictions:
- Tool whitelist
- Path blacklist
- Resource limits (CPU, memory, time)
- Network policy (default: deny)

### 3. Path Validation

File system access is controlled:

```go
validator := security.NewPathValidator(
    []string{"/workspace", "/tmp"},  // Allowed prefixes
    []string{"*.key", "/etc/*"},     // Denied patterns
)

if err := validator.Validate("/workspace/main.go"); err != nil {
    // Access denied
}
```

### 4. Budget Limits

Control API usage and costs:

```yaml
claude:
  max_budget_usd: 5.0        # Per execution
  max_tokens: 100000         # Per execution
  timeout: 30m               # Execution timeout
```

## Security Best Practices

### 1. API Key Management

**Do:**
- Store API keys in secrets management
- Use environment variables for keys
- Rotate keys regularly
- Use separate keys for dev/prod

**Don't:**
- Commit keys to repositories
- Share keys in chat/emails
- Use production keys for testing

### 2. Skill Content

**Do:**
- Review skill content before deployment
- Validate skills from third parties
- Use tool restrictions
- Enable budget limits

**Don't:**
- Allow arbitrary tool access
- Trust unvalidated skill sources
- Disable security features

### 3. Platform Integration

**Do:**
- Use read-only tokens when possible
- Limit token permissions
- Use branch protection rules
- Review PR comments before merge

**Don't:**
- Use admin tokens unnecessarily
- Grant write access blindly
- Bypass review processes

### 4. Network Security

**Do:**
- Use HTTPS for API calls
- Verify SSL certificates
- Use VPNs for private networks
- Restrict outbound traffic

**Don't:**
- Disable SSL verification
- Use HTTP in production
- Open unnecessary ports

## Enterprise Deployment

### 1. Isolated Environment

Deploy in an isolated environment:

```yaml
# docker-compose.yml
services:
  cicd-toolkit:
    image: cicd-ai-toolkit:latest
    network_mode: none
    read_only: true
    tmpfs:
      - /tmp:rw,noexec,nosuid
    volumes:
      - ./workspace:/workspace:rw
```

### 2. Proxy Configuration

Use corporate proxies:

```bash
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=http://proxy.example.com:8080
export NO_PROXY=localhost,127.0.0.1
```

### 3. Audit Logging

Enable audit logging:

```yaml
logging:
  level: info
  format: json
  output: /var/log/cicd-toolkit/audit.log
  audit:
    enabled: true
    include:
      - api_requests
      - skill_execution
      - file_access
```

### 4. Compliance

For regulated environments:

- **SOC 2**: Enable audit logging and access controls
- **HIPAA**: Encrypt data at rest and in transit
- **GDPR**: Data residency and retention policies
- **PCI DSS**: Restrict access to cardholder data

## Threat Model

### Considered Threats

1. **Prompt Injection**: Mitigated by pattern detection
2. **Code Injection**: Mitigated by sandbox execution
3. **Path Traversal**: Mitigated by path validation
4. **Resource Exhaustion**: Mitigated by budget limits
5. **Data Exfiltration**: Mitigated by network policy

### Out of Scope

These threats are outside the current threat model:
- Physical attacks on infrastructure
- Compromised Anthropic API
- Social engineering of users
- Supply chain attacks in dependencies

## Reporting Security Issues

To report a security vulnerability:

1. **Do not** use public issues
2. Email: security@cicd-ai-toolkit.com
3. Include: Description, steps to reproduce, impact
4. Wait for confirmation before disclosing

## Security Checklist

Before deploying to production:

- [ ] API keys stored securely
- [ ] Budget limits configured
- [ ] Tool restrictions enabled
- [ ] Audit logging enabled
- [ ] Network policies configured
- [ ] Skills reviewed and validated
- [ ] Dependencies up to date
- [ ] Access controls configured
- [ ] Backup and recovery plan in place
- [ ] Incident response process defined

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CWE Top 25](https://cwe.mitre.org/top25/)
- [Anthropic Security](https://www.anthropic.com/security)
