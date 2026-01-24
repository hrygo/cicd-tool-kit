# SPEC-LIB-04: Compliance Check

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24
**Covers**: PRD 2.2 (Phase 2), FR-8

## 1. 概述 (Overview)

Compliance Check Skill 对基础设施即代码 (IaC) 和配置文件进行自动化策略审查，确保符合安全最佳实践、组织合规要求和行业标准。支持 Terraform、Kubernetes、Docker、Ansible 等多种格式。

## 2. 核心职责 (Core Responsibilities)

- **IaC Scanning**: 审查 Terraform、Kubernetes、Docker 配置
- **Policy Validation**: 基于 OPA/Rego 策略进行验证
- **Security Best Practices**: 检测常见安全配置错误
- **Compliance Framework**: 支持 CIS Benchmark、NIST、SOC2、GDPR 等
- **Remediation Suggestions**: 提供修复建议和示例代码

## 3. 详细设计 (Detailed Design)

### 3.1 架构设计

```go
// ComplianceChecker Core Logic
type ComplianceChecker struct {
    analyzers    map[string]IaCAnalyzer
    policyEngine  *PolicyEngine
    reporters     []ComplianceReporter
    severityMap   map[string]int
}

type IaCAnalyzer interface {
    Name() string
    Detect() bool
    Parse(content []byte) (*IaCDocument, error)
    FindIssues(doc *IaCDocument) ([]ComplianceIssue, error)
}

type IaCDocument struct {
    Type       string // "terraform", "kubernetes", "docker", "ansible"
    Path       string
    Resources  []Resource
    Config     map[string]interface{}
}

type ComplianceIssue struct {
    ID          string   // Unique issue identifier
    Severity    string   // "critical", "high", "medium", "low"
    Category    string   // "security", "compliance", "best-practice"
    Title       string
    Description string
    Resource    string   // Affected resource name
    Line        int      // Line number
    Remediation string   // How to fix
    References  []string // Links to documentation
    Framework   []string // Applicable compliance frameworks
}

type PolicyEngine struct {
    regoFiles   []string
    policies    map[string]*Policy
    input       map[string]interface{}
}

func (pe *PolicyEngine) Evaluate(input map[string]interface{}) (*PolicyResult, error) {
    results := &PolicyResult{Violations: []Violation{}}

    for _, policy := range pe.policies {
        // Evaluate against OPA
        query := fmt.Sprintf("data.%s.violations", policy.Name)
        resp, err := opa.Eval(query, input)
        if err != nil {
            return nil, err
        }

        if len(resp) > 0 {
            results.Violations = append(results.Violations, parseViolations(resp)...)
        }
    }

    return results, nil
}
```

### 3.2 支持的 IaC 格式

| 格式 | 检测文件 | 检查项 |
|------|----------|--------|
| **Terraform** | `*.tf` | 安全配置、标签、版本约束、Secret 管理 |
| **Kubernetes** | `*.yaml`, `*.yml` | RBAC、资源限制、网络策略、Secret 管理 |
| **Docker** | `Dockerfile`, `*.dockerfile` | Root 用户、最小化镜像、CVE 扫描 |
| **Ansible** | `*.yml`, `*.yaml` | 幂等性、No-log、变量处理 |
| **CloudFormation** | `*.yaml`, `*.json` | IaC 最佳实践、Tag 策略 |
| **Helm** | `Chart.yaml`, `values.yaml` | Values 验证、Template 检查 |

### 3.3 Terraform 检查器

```go
// TerraformAnalyzer checks Terraform configurations
type TerraformAnalyzer struct {
    builtinRules []TerraformRule
    customRules  []TerraformRule
}

type TerraformRule struct {
    ID          string
    Name        string
    Severity    string
    Category    string
    Description string
    CheckFunc   func(*hcl.File, []string) ([]ComplianceIssue, error)
}

func (ta *TerraformAnalyzer) Parse(content []byte) (*IaCDocument, error) {
    file, diags := hcl.ParseBytes(content, "")
    if diags.HasErrors() {
        return nil, fmt.Errorf("parse error: %v", diags.Error())
    }

    doc := &IaCDocument{
        Type: "terraform",
    }

    // Parse Terraform configuration
    body, _ := file.Body.(*hcl.Body)
    schema := &hcl.BodySchema{}
    body.Schema(schema)

    // Extract resources
    for _, block := range body.Blocks.OfType("resource") {
        resource := Resource{
            Type: block.Labels[0],
            Name: block.Labels[1],
        }
        doc.Resources = append(doc.Resources, resource)
    }

    return doc, nil
}

// Built-in Terraform Rules
var terraformBuiltinRules = []TerraformRule{
    {
        ID:       "TF001",
        Name:     "S3 Bucket Encryption",
        Severity: "critical",
        Category: "security",
        Description: "S3 buckets must have encryption enabled",
        CheckFunc: func(f *hcl.File, refs []string) []ComplianceIssue {
            // Find all aws_s3_bucket resources
            // Check if server_side_encryption_configuration is present
            var issues []ComplianceIssue

            for _, res := range findResources(f, "aws_s3_bucket") {
                if !hasChildBlock(res, "server_side_encryption_configuration") {
                    issues = append(issues, ComplianceIssue{
                        ID:       "TF001",
                        Severity: "critical",
                        Category: "security",
                        Title:    "S3 Bucket Missing Encryption",
                        Resource: res.Name,
                        Line:     res.Range.Start.Line,
                        Description: "S3 bucket must have server-side encryption enabled",
                        Remediation: `Add server_side_encryption_configuration block:

resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"

  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default = true
    }
  }
}`,
                        References: []string{
                            "https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucket-encryption.html",
                        },
                        Framework: []string{"CIS", "NIST", "SOC2"},
                    })
                }
            }

            return issues
        },
    },
    {
        ID:       "TF002",
        Name:     "S3 Bucket Public Access",
        Severity: "critical",
        Category: "security",
        Description: "S3 buckets must block public access",
        CheckFunc: func(f *hcl.File, refs []string) []ComplianceIssue {
            var issues []ComplianceIssue

            for _, res := range findResources(f, "aws_s3_bucket") {
                // Check for aws_s3_bucket_public_access_block
                bucketName := res.Labels[1]
                if !hasPublicAccessBlock(f, bucketName) {
                    issues = append(issues, ComplianceIssue{
                        ID:       "TF002",
                        Severity: "critical",
                        Category: "security",
                        Title:    "S3 Bucket Allows Public Access",
                        Resource: bucketName,
                        Line:     res.Range.Start.Line,
                        Description: "S3 bucket must block all public access",
                        Remediation: `Add aws_s3_bucket_public_access_block:

resource "aws_s3_bucket_public_access_block" "example" {
  bucket = aws_s3_bucket.example.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}`,
                        Framework: []string{"CIS", "NIST", "SOC2", "GDPR"},
                    })
                }
            }

            return issues
        },
    },
    {
        ID:       "TF003",
        Name:     "RDS Encryption",
        Severity: "high",
        Category: "security",
        Description: "RDS instances must have encryption at rest",
        CheckFunc: func(f *hcl.File, refs []string) []ComplianceIssue {
            var issues []ComplianceIssue

            for _, res := range findResources(f, "aws_db_instance") {
                if !hasAttribute(res, "storage_encrypted") ||
                   getAttributeValue(res, "storage_encrypted") != "true" {
                    issues = append(issues, ComplianceIssue{
                        ID:       "TF003",
                        Severity: "high",
                        Category: "security",
                        Title:    "RDS Instance Missing Encryption",
                        Resource: res.Labels[1],
                        Line:     res.Range.Start.Line,
                        Description: "RDS instance must have storage_encrypted = true",
                        Remediation: `Set storage_encrypted = true:

resource "aws_db_instance" "example" {
  ...
  storage_encrypted = true
  ...
}`,
                        Framework: []string{"CIS", "HIPAA", "GDPR"},
                    })
                }
            }

            return issues
        },
    },
    {
        ID:       "TF004",
        Name:     "Resource Tagging",
        Severity: "medium",
        Category: "compliance",
        Description: "Resources must have required tags for cost allocation",
        CheckFunc: func(f *hcl.File, refs []string) []ComplianceIssue {
            var issues []ComplianceIssue

            requiredTags := []string{"Environment", "Owner", "CostCenter", "Compliance"}

            for _, res := range getAllResources(f) {
                tags := getTagsAttribute(res)
                missing := findMissingTags(tags, requiredTags)

                if len(missing) > 0 {
                    issues = append(issues, ComplianceIssue{
                        ID:       "TF004",
                        Severity: "medium",
                        Category: "compliance",
                        Title:    "Resource Missing Required Tags",
                        Resource: fmt.Sprintf("%s.%s", res.Type, res.Labels[1]),
                        Line:     res.Range.Start.Line,
                        Description: fmt.Sprintf("Missing tags: %s", strings.Join(missing, ", ")),
                        Remediation: fmt.Sprintf(`Add tags block:

resource "%s" "example" {
  ...
  tags = {
    Environment = var.environment
    Owner       = var.owner
    CostCenter   = var.cost_center
    Compliance  = var.compliance
  }
}`, res.Type),
                        Framework: []string{"IT-Governance"},
                    })
                }
            }

            return issues
        },
    },
}
```

### 3.4 Kubernetes 检查器

```go
// KubernetesAnalyzer checks Kubernetes manifests
type KubernetesAnalyzer struct {
    builtinRules []KubernetesRule
}

type KubernetesRule struct {
    ID          string
    Name        string
    Severity    string
    Category    string
    CheckFunc   func(*yaml.Node) ([]ComplianceIssue, error)
}

var kubernetesBuiltinRules = []KubernetesRule{
    {
        ID:       "K8S001",
        Name:     "Privileged Container",
        Severity: "critical",
        Category: "security",
        CheckFunc: func(node *yaml.Node) []ComplianceIssue {
            var issues []ComplianceIssue

            // Find containers with privileged: true
            for _, container := range findContainers(node) {
                if isPrivileged(container) {
                    issues = append(issues, ComplianceIssue{
                        ID:       "K8S001",
                        Severity: "critical",
                        Category: "security",
                        Title:    "Privileged Container Detected",
                        Resource: getContainerName(container),
                        Description: "Container runs with privileged mode, which gives it full access to host system",
                        Remediation: `Remove privileged: true or set securityContext:

securityContext:
  privileged: false
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
    add:
      - NET_BIND_SERVICE`,
                        Framework: []string{"CIS", "NIST", "Pod Security"},
                    })
                }
            }

            return issues
        },
    },
    {
        ID:       "K8S002",
        Name:     "Resource Limits",
        Severity: "medium",
        Category: "best-practice",
        CheckFunc: func(node *yaml.Node) []ComplianceIssue {
            var issues []ComplianceIssue

            for _, container := range findContainers(node) {
                if !hasResourceLimits(container) {
                    issues = append(issues, ComplianceIssue{
                        ID:       "K8S002",
                        Severity: "medium",
                        Category: "best-practice",
                        Title:    "Container Missing Resource Limits",
                        Resource: getContainerName(container),
                        Description: "Container without limits can consume excessive resources",
                        Remediation: `Add resource limits:

resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "256Mi"
    cpu: "200m"`,
                        Framework: []string{"CIS", "Pod Security"},
                    })
                }
            }

            return issues
        },
    },
    {
        ID:       "K8S003",
        Name:     "Root User",
        Severity: "high",
        Category: "security",
        CheckFunc: func(node *yaml.Node) []ComplianceIssue {
            var issues []ComplianceIssue

            for _, container := range findContainers(node) {
                if runsAsRoot(container) {
                    issues = append(issues, ComplianceIssue{
                        ID:       "K8S003",
                        Severity: "high",
                        Category: "security",
                        Title:    "Container Runs as Root User",
                        Resource: getContainerName(container),
                        Description: "Container should run as non-root user",
                        Remediation: `Set runAsNonRoot and runAsUser:

securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  runAsGroup: 1000`,
                        Framework: []string{"CIS", "Pod Security"},
                    })
                }
            }

            return issues
        },
    },
    {
        ID:       "K8S004",
        Name:     "Secret Management",
        Severity: "critical",
        Category: "security",
        CheckFunc: func(node *yaml.Node) []ComplianceIssue {
            var issues []ComplianceIssue

            // Check for secrets in environment variables
            for _, container := range findContainers(node) {
                for _, env := range getEnvVars(container) {
                    if containsSensitiveData(env.Value) {
                        issues = append(issues, ComplianceIssue{
                            ID:       "K8S004",
                            Severity: "critical",
                            Category: "security",
                            Title:    "Sensitive Data in Environment Variable",
                            Resource: getContainerName(container),
                            Description: "Environment variable contains potential sensitive data",
                            Remediation: `Use Kubernetes Secrets instead of environment variables:

envFrom:
  - secretRef:
      name: my-secret`,
                            Framework: []string{"CIS", "NIST", "SOC2"},
                        })
                    }
                }
            }

            return issues
        },
    },
}
```

### 3.5 Docker 检查器

```go
// DockerAnalyzer checks Dockerfiles
type DockerAnalyzer struct {
    builtinRules []DockerRule
}

type DockerRule struct {
    ID          string
    Name        string
    Severity    string
    Category    string
    CheckFunc   func(*dockerfile.Dockerfile) ([]ComplianceIssue, error)
}

var dockerBuiltinRules = []DockerRule{
    {
        ID:       "DK001",
        Name:     "Root User",
        Severity: "high",
        Category: "security",
        CheckFunc: func(df *dockerfile.Dockerfile) []ComplianceIssue {
            var issues []ComplianceIssue

            if !hasUserInstruction(df) {
                issues = append(issues, ComplianceIssue{
                    ID:       "DK001",
                    Severity: "high",
                    Category: "security",
                    Title:    "Dockerfile Runs as Root User",
                    Description: "Container should run as non-root user",
                    Remediation: `Add USER instruction:

FROM alpine:3.18
RUN addgroup -g appuser && adduser -D appuser
USER appuser

... rest of Dockerfile`,
                    Framework: []string{"CIS", "Docker Security"},
                })
            }

            return issues
        },
    },
    {
        ID:       "DK002",
        Name:     "Base Image Tag",
        Severity: "medium",
        Category: "best-practice",
        CheckFunc: func(df *dockerfile.Dockerfile) []ComplianceIssue {
            var issues []ComplianceIssue

            for _, cmd := range df.BaseInstructions {
                if cmd.Cmd == "from" {
                    image := cmd.Value[0]
                    if strings.HasPrefix(image, "alpine:latest") ||
                       strings.HasPrefix(image, "ubuntu:latest") {
                        issues = append(issues, ComplianceIssue{
                            ID:       "DK002",
                            Severity: "medium",
                            Category: "best-practice",
                            Title:    "Using Latest Tag",
                            Description: fmt.Sprintf("Base image uses 'latest' tag: %s", image),
                            Remediation: `Use specific version tag:

FROM alpine:3.18.4
# or
FROM ubuntu:22.04`,
                            Framework: []string{"Docker Security"},
                        })
                    }
                }
            }

            return issues
        },
    },
    {
        ID:       "DK003",
        Name:     "Layer Minimization",
        Severity: "low",
        Category: "best-practice",
        CheckFunc: func(df *dockerfile.Dockerfile) []ComplianceIssue {
            var issues []ComplianceIssue

            // Check for RUN apt-get update && apt-get install -y without cleanup
            for _, cmd := range df.AllInstructions {
                if cmd.Cmd == "run" {
                    cmdStr := strings.Join(cmd.Value, " ")
                    if strings.Contains(cmdStr, "apt-get install") &&
                       !strings.Contains(cmdStr, "rm -rf /var/lib/apt/lists/*") {
                        issues = append(issues, ComplianceIssue{
                            ID:       "DK003",
                            Severity: "low",
                            Category: "best-practice",
                            Title:    "Unclean APT Install",
                            Description: "APT cache should be cleaned in same layer",
                            Remediation: `Combine install and cleanup:

RUN apt-get update && apt-get install -y \\
    package1 \\
    package2 \\
    && rm -rf /var/lib/apt/lists/*`,
                        })
                    }
                }
            }

            return issues
        },
    },
}
```

### 3.6 OPA 策略集成

```go
// OPAIntegration integrates Open Policy Agent
type OPAIntegration struct {
    regoFiles []string
    policies  map[string]*OPAPolicy
}

type OPAPolicy struct {
    Name     string
    RegoFile string
    Input    map[string]interface{}
}

// Example OPA Policy for Terraform
const terraformOPAPolicy = `
package cicd.compliance.terraform

# Deny S3 buckets without encryption
deny_s3_no_encryption[msg] {
    input.resource.type == "aws_s3_bucket"
    not input.resource.encryption.enabled
    msg := "S3 bucket must have encryption enabled"
}

# Deny public S3 buckets
deny_s3_public_access[msg] {
    input.resource.type == "aws_s3_bucket"
    not input.resource.public_access_block.enabled
    msg := "S3 bucket must block public access"
}

# Deny RDS without encryption
deny_rds_no_encryption[msg] {
    input.resource.type == "aws_db_instance"
    input.resource.storage_encrypted != true
    msg := "RDS instance must have encryption at rest enabled"
}

# Require mandatory tags
deny_missing_tags[msg] {
    count(input.resource.tags) < 4
    msg := sprintf("Resource missing required tags: %v", input.resource.missing_tags)
}
`

func (opa *OPAIntegration) Evaluate(doc *IaCDocument) (*PolicyResult, error) {
    // Prepare input for OPA
    input := map[string]interface{}{
        "resource": doc,
        "frameworks": []string{"CIS", "NIST", "SOC2"},
    }

    // Load Rego policies
    policies, err := opa.loadPolicies()
    if err != nil {
        return nil, err
    }

    // Evaluate with OPA
    module := opa.CompileRego(policies)
    results, err := module.Eval(input)
    if err != nil {
        return nil, err
    }

    return parseOPAResults(results), nil
}
```

### 3.7 合规框架映射

```go
// ComplianceFramework defines a compliance framework
type ComplianceFramework struct {
    Name        string
    Version     string
    Description string
    Controls    []Control
}

type Control struct {
    ID          string
    Description string
    Rules       []string // Rule IDs that satisfy this control
}

// Built-in compliance frameworks
var complianceFrameworks = map[string]*ComplianceFramework{
    "CIS-AWS-1.4": {
        Name:    "CIS AWS Foundations Benchmark",
        Version: "1.4.0",
        Description: "CIS Benchmarks for AWS",
        Controls: []Control{
            {ID: "2.1.1", Description: "Ensure S3 buckets are not publicly accessible", Rules: []string{"TF002"}},
            {ID: "2.1.2", Description: "Ensure S3 bucket encryption is enabled", Rules: []string{"TF001"}},
            {ID: "2.2.1", Description: "Ensure RDS encryption is enabled", Rules: []string{"TF003"}},
            {ID: "2.3.1", Description: "Ensure resources have tags", Rules: []string{"TF004"}},
        },
    },
    "CIS-Kubernetes-1.23": {
        Name:    "CIS Kubernetes Benchmark",
        Version: "1.23.0",
        Description: "CIS Benchmarks for Kubernetes",
        Controls: []Control{
            {ID: "5.1.1", Description: "Ensure privileged containers are not used", Rules: []string{"K8S001"}},
            {ID: "5.2.1", Description: "Ensure resources have limits", Rules: []string{"K8S002"}},
            {ID: "5.2.2", Description: "Ensure containers run as non-root", Rules: []string{"K8S003"}},
        },
    },
    "CIS-Docker-1.2": {
        Name:    "CIS Docker Benchmark",
        Version: "1.2.0",
        Description: "CIS Benchmarks for Docker",
        Controls: []Control{
            {ID: "4.1", Description: "Ensure a user for the container has been created", Rules: []string{"DK001"}},
            {ID: "4.5", Description: "Ensure images are not tagged with latest", Rules: []string{"DK002"}},
        },
    },
    "NIST-800-53": {
        Name:    "NIST SP 800-53",
        Version: "Rev 5",
        Description: "Security and Privacy Controls",
        Controls: []Control{
            {ID: "SC-12", Description: "Cryptographic Key Management and Establishment", Rules: []string{"TF001", "K8S004"}},
            {ID: "SC-8", Description: "Transmission Confidentiality and Integrity", Rules: []string{"K8S004", "TF002"}},
            {ID: "AC-6", Description: "Least Privilege", Rules: []string{"K8S001", "K8S003"}},
        },
    },
    "SOC2": {
        Name:    "SOC 2 Type II",
        Version: "2017",
        Description: "Service Organization Control 2",
        Controls: []Control{
            {ID: "CC6.1", Description: "Encryption of Data at Rest", Rules: []string{"TF001", "TF003", "K8S004"}},
            {ID: "CC6.6", Description: "Confidentiality of Data in Transit", Rules: []string{"TF002", "K8S004"}},
        },
    },
    "GDPR": {
        Name:    "GDPR",
        Version: "2018",
        Description: "General Data Protection Regulation",
        Controls: []Control{
            {ID: "Art.32", Description: "Security of Processing", Rules: []string{"TF001", "TF003", "K8S004"}},
            {ID: "Art.25", Description: "Data Protection by Design", Rules: []string{"K8S002", "K8S003"}},
        },
    },
}
```

### 3.8 报告生成

```go
// ReportGenerator generates compliance reports
type ReportGenerator struct {
    formatters map[string]ReportFormatter
}

type ComplianceReport struct {
    Summary      ReportSummary
    Frameworks   []FrameworkResult
    Issues       []ComplianceIssue
    PassRate     float64
}

type ReportSummary struct {
    TotalFiles    int
    TotalIssues   int
    Critical      int
    High          int
    Medium        int
    Low           int
    Timestamp     time.Time
}

type FrameworkResult struct {
    Name        string
    Version     string
    Passed      int
    Failed      int
    Total       int
    Controls    map[string]ControlResult
}

type ReportFormatter interface {
    Format(report *ComplianceReport) (string, error)
    Extension() string
}

// SARIFFormatter generates SARIF format (Static Analysis Results Interchange Format)
type SARIFFormatter struct{}

func (f *SARIFFormatter) Format(report *ComplianceReport) (string, error) {
    sarif := map[string]interface{}{
        "version": "2.1.0",
        "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
        "runs": []map[string]interface{}{
            {
                "tool": map[string]interface{}{
                    "driver": map[string]interface{}{
                        "name": "cicd-ai-toolkit-compliance-check",
                        "version": "1.0.0",
                        "rules": f.buildRules(report.Issues),
                    },
                },
                "results": f.buildResults(report.Issues),
            },
        },
    }

    data, _ := json.MarshalIndent(sarif, "", "  ")
    return string(data), nil
}

// MarkdownFormatter generates human-readable Markdown
type MarkdownFormatter struct{}

func (f *MarkdownFormatter) Format(report *ComplianceReport) (string, error) {
    var sb strings.Builder

    sb.WriteString("# Compliance Check Report\n\n")
    sb.WriteString(fmt.Sprintf("**Generated**: %s\n\n", report.Summary.Timestamp.Format("2006-01-02 15:04:05")))

    // Executive Summary
    sb.WriteString("## Executive Summary\n\n")
    sb.WriteString(fmt.Sprintf("| Metric | Count |\n|--------|-------|\n"))
    sb.WriteString(fmt.Sprintf("| Total Files | %d |\n", report.Summary.TotalFiles))
    sb.WriteString(fmt.Sprintf("| Total Issues | %d |\n", report.Summary.TotalIssues))
    sb.WriteString(fmt.Sprintf("| Critical | %d |\n", report.Summary.Critical))
    sb.WriteString(fmt.Sprintf("| High | %d |\n", report.Summary.High))
    sb.WriteString(fmt.Sprintf("| Medium | %d |\n", report.Summary.Medium))
    sb.WriteString(fmt.Sprintf("| Low | %d |\n", report.Summary.Low))
    sb.WriteString(fmt.Sprintf("| **Pass Rate** | **%.1f%%** |\n\n", report.PassRate*100))

    // Issues by Severity
    if report.Summary.Critical > 0 {
        sb.WriteString("### 🚨 Critical Issues\n\n")
        for _, issue := range filterBySeverity(report.Issues, "critical") {
            sb.WriteString(f.formatIssue(issue))
        }
    }

    // Framework Results
    sb.WriteString("## Framework Results\n\n")
    for _, fw := range report.Frameworks {
        passRate := float64(fw.Passed) / float64(fw.Total) * 100
        status := "✅"
        if passRate < 100 {
            status = "⚠️"
        }

        sb.WriteString(fmt.Sprintf("### %s %s - %.1f%%\n\n", status, fw.Name, passRate))
    }

    return sb.String(), nil
}
```

### 3.9 SKILL.md 定义

```markdown
---
name: "compliance-check"
version: "1.0.0"
description: "Automated IaC compliance and security policy checking"
author: "cicd-ai-toolkit"
license: "MIT"

options:
  thinking:
    budget_tokens: 4096
  temperature: 0.0

tools:
  allow: ["read", "grep", "ls", "bash"]

inputs:
  - name: target_path
    type: string
    description: "Path to IaC files or directory"
  - name: framework
    type: string
    description: "Compliance framework (cis, nist, soc2, gdpr, custom)"
  - name: iac_type
    type: string
    description: "Type of IaC (terraform, kubernetes, docker, all)"
  - name: output_format
    type: string
    description: "Output format (markdown, json, sarif)"
  - name: fail_on
    type: string
    description: "Fail if severity level or higher found (critical, high, medium, low)"
---

# Compliance Checker

You are an IaC security and compliance expert. Review infrastructure code for security and policy violations.

## Check Categories

### Security
- Encryption at rest and in transit
- Access control and least privilege
- Secret and credential management
- Network security and isolation
- Container security

### Compliance
- Tagging and governance requirements
- Resource naming conventions
- Cost control measures
- Audit trail configuration

### Best Practices
- Resource limits and constraints
- High availability configuration
- Disaster recovery setup
- Documentation completeness

## Analysis Steps

1. **Detect IaC Type**: Identify Terraform, Kubernetes, Docker, or other formats
2. **Parse Resources**: Extract resources and their configurations
3. **Apply Rules**: Run built-in and custom compliance rules
4. **Map Frameworks**: Map violations to compliance framework controls
5. **Generate Report**: Create comprehensive compliance report

## Output Format

Output in XML-wrapped JSON:

```xml
<json>
{
  "summary": {
    "total_files": 5,
    "total_issues": 12,
    "critical": 2,
    "high": 4,
    "medium": 4,
    "low": 2,
    "pass_rate": 0.67
  },
  "frameworks": [
    {
      "name": "CIS AWS Foundations Benchmark",
      "version": "1.4.0",
      "passed": 15,
      "failed": 3,
      "total": 18,
      "pass_rate": 0.83
    }
  ],
  "issues": [
    {
      "id": "TF001",
      "severity": "critical",
      "category": "security",
      "title": "S3 Bucket Missing Encryption",
      "resource": "aws_s3_bucket.example",
      "file": "main.tf",
      "line": 42,
      "description": "S3 bucket must have encryption enabled",
      "remediation": "Add server_side_encryption_configuration block...",
      "references": ["https://..."],
      "frameworks": ["CIS", "NIST", "SOC2"]
    }
  ]
}
</json>
```
```

### 3.10 集成示例

```yaml
# .github/workflows/compliance-check.yml
name: IaC Compliance Check

on:
  pull_request:
    paths:
      - "**.tf"
      - "**.yaml"
      - "**.yml"
      - "Dockerfile*"

jobs:
  compliance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Compliance Check
        uses: cicd-ai-toolkit/action@v1
        with:
          run_skills: "compliance-check"
          framework: "cis,nist,soc2"
          iac_type: "all"
          fail_on: "critical"
          output_format: "sarif"

      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: compliance-results.sarif
```

## 4. 依赖关系 (Dependencies)

- **Related**: [SPEC-SKILL-01](./SPEC-SKILL-01-Skill_Definition.md) - Skill 定义标准
- **Related**: [SPEC-LIB-01](./SPEC-LIB-01-Standard_Skills.md) - Standard Skills 库
- **Related**: [SPEC-GOV-01](./SPEC-GOV-01-Policy_As_Code.md) - OPA 策略引擎

## 5. 验收标准 (Acceptance Criteria)

1. **Terraform 检查**: 能检测未加密的 S3 bucket、RDS 实例
2. **Kubernetes 检查**: 能检测特权容器、无资源限制的容器
3. **Docker 检查**: 能检测 Root 用户、latest 标签
4. **框架映射**: 能正确映射违反项到 CIS/NIST/SOC2 控制点
5. **SARIF 输出**: 能生成有效的 SARIF 2.1.0 格式文件
6. **Markdown 报告**: 能生成人类可读的 Markdown 报告
7. **Remediation 建议**: 每个问题提供具体的修复代码示例
8. **自定义规则**: 支持用户加载自定义 OPA Rego 策略
