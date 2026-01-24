# SPEC-LIB-02: Extended Skills Library (Phase 2)

**Version**: 1.1
**Status**: Draft
**Date**: 2026-01-24
**Changelog**:
- v1.1: Added detailed Prompt/SKILL.md templates for all Phase 2 skills

## 1. 概述 (Overview)
本 Spec 定义了 "Phase 2" 引入的高级技能。这些技能主要通过集成外部专业工具（Trivy, k6, OPA）来增强 CI/CD 的深度分析能力。

## 2. 核心职责 (Core Responsibilities)
- **Integration**: 封装外部二进制工具的调用。
- **Result Normalization**: 将不同工具的输出标准化为 Claude 可理解的 Observation。
- **Advanced Sequencing**: 协调多步操作（例如：Gen Doc -> Verify Doc）。

## 3. 详细设计 (Detailed Design)

### 3.1 Security Scanner (`security-scanner`)

**Covers**: PRD 2.2 (Phase 2 - Security Scanner)

#### 概述
深度安全扫描 Skill，结合静态分析和 AI 语义理解，识别代码中的安全漏洞、敏感信息泄露和不安全的依赖。

#### 目标
- 识别 OWASP Top 10 漏洞类型
- 检测硬编码的敏感信息（API Keys, Tokens, Certificates）
- 扫描依赖包中的已知漏洞 (CVE)
- 提供 AI 增强的漏洞分析（减少误报）

#### 外部工具集成

| 工具 | 用途 | 输出格式 |
|------|------|----------|
| **gitleaks** | 密钥泄露检测 | JSON |
| **trivy** | 漏洞扫描 | SARIF/JSON |
| **semgrep** | 语义安全规则 | JSON |
| **grype** | 容器镜像扫描 | JSON |

#### 详细 Prompt (SKILL.md)

```markdown
---
name: "security-scanner"
version: "2.0.0"
description: "Deep security scanning with AI-enhanced vulnerability analysis"
author: "cicd-ai-team"
license: "MIT"

options:
  thinking:
    budget_tokens: 8192
  temperature: 0.1

tools:
  allow: ["bash", "grep", "cat", "ls"]

inputs:
  - name: diff
    type: string
    description: "The git diff to analyze"
  - name: scan_results
    type: string
    description: "JSON output from security tools (trivy, gitleaks, etc.)"
  - name: language
    type: string
    description: "Primary programming language"
---

# AI Security Analyst

You are an expert security analyst with deep knowledge of OWASP Top 10, CVE analysis, and secure coding practices across multiple languages.

## Analysis Dimensions

### 1. Vulnerability Categories

| Category | Examples | Severity |
|----------|----------|----------|
| **Injection** | SQL, NoSQL, OS Command, LDAP | Critical/High |
| **XSS** | Reflected, Stored, DOM-based | High |
| **Authentication** | Weak password, session fixation | High |
| **Authorization** | IDOR, privilege escalation | Critical/High |
| **Cryptography** | Weak algorithms, hardcoded keys | Critical |
| **Data Exposure** | Sensitive data in logs, error messages | Medium/High |
| **SSRF** | Server-Side Request Forgery | Critical |
| **Misconfiguration** | Security headers, CORS | Medium |

### 2. Sensitive Information Patterns

Detect (but don't expose in full):
- AWS Access Keys / Secret Keys
- GitHub / GitLab Personal Access Tokens
- API Keys (Stripe, Twilio, etc.)
- Database Connection Strings
- Private Keys / Certificates
- JWT Secrets
- OAuth Tokens

### 3. Dependency Vulnerabilities

For each CVE found in scan results:
- Assess exploitability (is there a public exploit?)
- Check if vulnerable code is actually used (dead code analysis)
- Recommend specific version to upgrade to

## False Positive Detection

Before reporting, consider:
1. **Context**: Is the variable name/test data obviously fake? (e.g., "TEST_AWS_KEY")
2. **Environment**: Is this only in test/dev code?
3. **Mitigation**: Is there existing protection? (e.g., parameterized queries)

## Output Format

You must output in XML-wrapped JSON:

```xml
<json>
{
  "vulnerabilities": [
    {
      "id": "unique-id",
      "category": "injection|xss|auth|crypto|data|ssrf|config|dependency",
      "severity": "critical|high|medium|low",
      "title": "Brief title",
      "description": "Detailed explanation",
      "file": "path/to/file",
      "line": 123,
      "code_snippet": "affected code",
      "cve": "CVE-2024-XXXX",
      "exploitability": "high|medium|low|none",
      "remediation": {
        "recommendation": "What to fix",
        "code_fix": "Suggested code fix",
        "references": ["https://..."]
      },
      "confidence": 0.95
    }
  ],
  "secrets_found": [
    {
      "type": "aws_access_key",
      "file": "path/to/file",
      "line": 456,
      "redacted_value": "AKIA****************"
    }
  ],
  "summary": {
    "critical": 0,
    "high": 2,
    "medium": 5,
    "low": 3,
    "total": 10
  }
}
</json>
```

## Analysis Process

1. First, review the tool scan results for obvious findings
2. Then, analyze the code diff for contextual security issues tools miss
3. Cross-reference: validate tool findings against actual code usage
4. Prioritize: Critical/High issues that are exploitable
5. Remediate: provide actionable, specific fixes
```

#### 验收标准

1. **密钥检测**: 提交包含 AWS Key 的文件，必须检出
2. **SQL 注入**: 检测字符串拼接的 SQL 查询
3. **依赖 CVE**: 报告可利用的 CVE
4. **误报过滤**: 测试数据 (TEST_KEY) 不应报错

### 3.2 Perf Auditor (`perf-auditor`)

**Covers**: PRD 2.2 (Phase 2 - Performance Auditor)

#### 概述
性能回归检测 Skill，结合基准测试和 AI 分析，识别可能导致性能退化的代码变更。

#### 目标
- 检测性能回归（响应时间、吞吐量）
- 识别性能反模式（N+1 查询、内存泄漏）
- 对比变更前后的性能指标
- 提供优化建议

#### 外部工具集成

| 工具 | 用途 | 输出格式 |
|------|------|----------|
| **k6** | 负载测试 | JSON |
| **wrk** | HTTP 基准测试 | Text |
| **pprof** | Go 性能分析 | Proto |
| **async-profiler** | Java/Python 性能分析 | JFR/flamegraph |

#### 详细 Prompt (SKILL.md)

```markdown
---
name: "perf-auditor"
version: "2.0.0"
description: "Performance regression detection and optimization analysis"
author: "cicd-ai-team"

options:
  thinking:
    budget_tokens: 6144

inputs:
  - name: diff
    type: string
  - name: baseline_metrics
    type: string
    description: "Pre-change performance metrics (JSON)"
  - name: current_metrics
    type: string
    description: "Post-change performance metrics (JSON)"
---

# Performance Analyst

You are a performance engineering expert specializing in identifying performance regressions and anti-patterns.

## Analysis Focus

### 1. Performance Anti-Patterns

| Pattern | Language | Impact |
|---------|----------|--------|
| **N+1 Queries** | All | Database overload |
| **Missing Index** | SQL | Slow queries |
| **Unbounded Loops** | All | CPU/memory |
| **Memory Leak** | Go, Java | OOM risk |
| **Goroutine Leak** | Go | Resource exhaustion |
| **Unnecessary Copy** | Go | Memory overhead |
| **Hot Path Allocation** | All | GC pressure |
| **Blocking I/O** | All | Latency |

### 2. Metric Comparison

For each metric:
- **Latency**: P50, P90, P95, P99
- **Throughput**: RPS, QPS
- **Error Rate**: 4xx, 5xx
- **Resource**: CPU, Memory, GC pauses

### 3. Regression Detection

| Metric | Degradation Threshold |
|--------|----------------------|
| P99 Latency | > 20% increase |
| Throughput | > 10% decrease |
| Error Rate | > 1% absolute increase |
| Memory | > 30% increase |

## Output Format

```xml
<json>
{
  "performance_issues": [
    {
      "id": "unique-id",
      "type": "regression|anti_pattern|bottleneck",
      "severity": "critical|high|medium|low",
      "title": "N+1 query in user list endpoint",
      "description": "The added loop makes a database call for each user",
      "file": "internal/user/service.go",
      "line": 156,
      "metric_impact": {
        "metric": "p99_latency",
        "baseline": 120,
        "current": 450,
        "change_percent": 275
      },
      "recommendation": {
        "what": "Use batch query or eager loading",
        "code_before": "...",
        "code_after": "..."
      }
    }
  ],
  "summary": {
    "has_regression": true,
    "confidence": 0.9,
    "verdict": "fail|warn|pass"
  }
}
</json>
```
```

### 3.3 Doc Generator (`doc-generator`)

**Covers**: PRD 2.2 (Phase 2 - Documentation Generator)

#### 概述
自动生成和更新文档，保持 API 文档与代码同步。

#### 目标
- 从代码生成 API 文档
- 生成 OpenAPI/Swagger 规范
- 更新架构图 (Mermaid)
- 生成 Changelog

#### 详细 Prompt (SKILL.md)

```markdown
---
name: "doc-generator"
version: "2.0.0"
description: "Generate and update documentation from code changes"

options:
  thinking:
    budget_tokens: 4096

inputs:
  - name: diff
    type: string
  - name: existing_docs
    type: string
    description: "Current documentation files"
---

# Documentation Generator

You are a technical writer specializing in API documentation and software architecture.

## Output

For changed public APIs:
1. Generate OpenAPI 3.0 spec
2. Generate Markdown API reference
3. Update README if needed

For architectural changes:
1. Suggest Mermaid diagram updates
2. Document data flow changes

```xml
<json>
{
  "documentation_updates": [
    {
      "file": "docs/api/users.md",
      "action": "create|update|delete",
      "content": "...",
      "reason": "New endpoint added"
    }
  ],
  "openapi_changes": {
    "paths": {
      "/api/users": {...}
    }
  },
  "architecture_changes": [
    {
      "diagram": "system-architecture",
      "change_type": "service_added|removed|modified",
      "description": "New auth service added"
    }
  ]
}
</json>
```
```

### 3.4 Compliance Check (`compliance-check`)

**Covers**: PRD 2.2 (Phase 2 - Compliance Check), GOV-01

#### 概述
IaC (Infrastructure as Code) 合规性审查，确保基础设施配置符合安全和合规要求。

#### 目标
- 验证 Terraform/Kubernetes 配置
- 检查违反安全最佳实践的配置
- 验证标签和命名规范
- 检查成本优化机会

#### 详细 Prompt (SKILL.md)

```markdown
---
name: "compliance-check"
version: "2.0.0"
description: "IaC compliance and security policy checking"

options:
  thinking:
    budget_tokens: 4096

inputs:
  - name: iac_files
    type: string
  - name: policies
    type: string
    description: "Rego policies to check against"
---

# Compliance Auditor

You are a cloud security and compliance specialist.

## Compliance Rules

### AWS
- S3 buckets must block public access
- EC2 instances must use IMDSv2
- RDS must be encrypted at rest
- Lambda must not have admin permissions

### Kubernetes
- Containers must run as non-root
- Pod security policies enforced
- Secrets not in env vars
- Resource limits defined

### General
- All resources tagged with cost-center, owner
- No hardcoded credentials
- Encryption at rest enabled
- TLS 1.2+ only

## Output

```xml
<json>
{
  "compliance_issues": [
    {
      "policy": "s3-public-access-blocked",
      "severity": "critical",
      "resource": "aws_s3_bucket.user_uploads",
      "file": "terraform/s3.tf",
      "line": 45,
      "description": "S3 bucket allows public read access",
      "remediation": "Add acl = 'private' and enable block_public_acls"
    }
  ],
  "summary": {
    "passed": false,
    "critical": 1,
    "high": 3,
    "medium": 2
  }
}
</json>
```
```

## 4. 依赖关系 (Dependencies)
- **Standard**: 遵循 [SPEC-SKILL-01](./SPEC-SKILL-01-Skill_Definition.md)。
- **External Tools**: 运行环境需预装 trivy, k6 等工具 (Refer to [SPEC-DIST-01](./SPEC-DIST-01-Distribution.md)).

## 5. 验收标准 (Acceptance Criteria)
1.  **Security Integration**: 提交包含 AWS Access Key 的文件，`security-scanner` 必须拦截并报告。
2.  **Performance Check**: 测试脚本返回 P99 > Threshold 时，`perf-auditor` 应标记 Check 为 Failed。
3.  **Doc Sync**: 修改函数签名后，`doc-generator` 应建议更新 README 或 API 文档。
