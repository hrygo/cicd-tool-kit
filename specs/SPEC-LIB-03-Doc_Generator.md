# SPEC-LIB-03: Doc Generator

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24
**Covers**: PRD 2.2 (Phase 2), FR-7

## 1. 概述 (Overview)

Doc Generator Skill 自动从代码生成各类文档，包括 API 文档、架构图和变更日志。通过分析代码结构、注释和类型定义，生成符合 OpenAPI 规范的文档和 Mermaid 架构图。

## 2. 核心职责 (Core Responsibilities)

- **API Documentation**: 从代码生成 OpenAPI/Swagger 规范
- **Architecture Diagrams**: 生成 Mermaid 格式的架构图和流程图
- **Changelog Generation**: 自动生成结构化的变更日志
- **Code Comments**: 检查并补充缺失的文档注释
- **Multi-Language Support**: 支持 Go, Python, TypeScript, Java

## 3. 详细设计 (Detailed Design)

### 3.1 架构设计

```go
// DocGenerator Core Logic
type DocGenerator struct {
    analyzer      *CodeAnalyzer
    parsers       map[string]LanguageParser
    templates     *TemplateEngine
    formatters    map[string]Formatter
}

type LanguageParser interface {
    Name() string
    Detect() bool
    ParseAPISpec(source []byte) (*APISpec, error)
    ParseComments(source []byte) (*CommentSet, error)
    ParseImports(source []byte) (*ImportGraph, error)
}

type Formatter interface {
    Name() string
    Format(spec *APISpec) (string, error)
    FormatDiagram(graph *ImportGraph) (string, error)
}
```

### 3.2 支持的语言和格式

| 语言 | API 格式 | 架构图 | 注释风格 |
|------|----------|--------|----------|
| **Go** | OpenAPI (from annotations) | Mermaid | godoc |
| **Python** | OpenAPI (from FastAPI/Flask) | Mermaid | docstring |
| **TypeScript** | OpenAPI (from TSDoc/Express) | Mermaid | JSDoc |
| **Java** | OpenAPI (from Javadoc/Spring) | Mermaid | Javadoc |

### 3.3 API Documentation 生成

#### 3.3.1 Go API 解析

```go
// GoAPIParser parses Go code for API documentation
type GoAPIParser struct{}

type GoAPISpec struct {
    Package     string
    Version     string
    Title       string
    Description string
    BasePath    string
    Endpoints   []Endpoint
    Models      []Schema
}

type Endpoint struct {
    Method      string   // GET, POST, PUT, DELETE
    Path        string
    Summary     string
    Description string
    Tags        []string
    Parameters  []Parameter
    Request     *Schema
    Responses   map[int]*Schema
    Examples    []Example
}

func (p *GoAPIParser) ParseAPISpec(source []byte) (*GoAPISpec, error) {
    // Parse Go source using go/ast
    fset := token.NewFileSet()
    node, err := parser.ParseFile(fset, "", source, parser.ParseComments)
    if err != nil {
        return nil, err
    }

    spec := &GoAPISpec{}

    // Extract package info
    spec.Package = node.Name.Name

    // Scan for HTTP handler registrations
    ast.Inspect(node, func(n ast.Node) bool {
        switch x := n.(type) {
        case *ast.FuncDecl:
            if p.isHTTPHandler(x) {
                endpoint := p.parseEndpoint(x)
                spec.Endpoints = append(spec.Endpoints, endpoint)
            }
        case *ast.GenDecl:
            if x.Tok == token.TYPE {
                // Extract type definitions for models
                schema := p.parseSchema(x)
                spec.Models = append(spec.Models, schema)
            }
        }
        return true
    })

    return spec, nil
}

func (p *GoAPIParser) isHTTPHandler(fn *ast.FuncDecl) bool {
    // Check for common patterns
    // - Function with Receiver of *http.ServeMux
    // - Function name contains HTTP methods
    httpMethods := []string{"Get", "Post", "Put", "Delete", "Patch"}
    for _, method := range httpMethods {
        if strings.Contains(fn.Name.Name, method) {
            return true
        }
    }

    // Check for gin/echo/fiber framework patterns
    // ...
    return false
}
```

#### 3.3.2 OpenAPI 输出

```go
// OpenAPIFormatter formats API spec as OpenAPI 3.0
type OpenAPIFormatter struct{}

func (f *OpenAPIFormatter) Format(spec *GoAPISpec) (string, error) {
    openapi := map[string]interface{}{
        "openapi": "3.0.0",
        "info": map[string]interface{}{
            "title":       spec.Title,
            "version":     spec.Version,
            "description": spec.Description,
        },
        "servers": []map[string]interface{}{
            {"url": spec.BasePath},
        },
        "paths": f.buildPaths(spec),
        "components": map[string]interface{}{
            "schemas": f.buildSchemas(spec.Models),
        },
    }

    data, err := json.MarshalIndent(openapi, "", "  ")
    if err != nil {
        return "", err
    }

    return string(data), nil
}
```

### 3.4 Architecture Diagram 生成

#### 3.4.1 Import Graph 分析

```go
// ImportGraph represents module dependencies
type ImportGraph struct {
    Nodes     []GraphNodes
    Edges     []GraphEdge
    Clusters  []GraphCluster
}

type GraphNode struct {
    ID       string
    Label    string
    Type     string // "package", "service", "database", "external"
    Shape    string // "rectangle", "cylinder", "circle"
}

type GraphEdge struct {
    From     string
    To       string
    Label    string
    Style    string // "solid", "dashed"
}

func (p *GoAPIParser) ParseImports(source []byte) (*ImportGraph, error) {
    graph := &ImportGraph{}

    fset := token.NewFileSet()
    node, err := parser.ParseFile(fset, "", source, nil)
    if err != nil {
        return nil, err
    }

    currentPackage := node.Name.Name

    // Add current package as node
    graph.Nodes = append(graph.Nodes, GraphNode{
        ID:    currentPackage,
        Label: currentPackage,
        Type:  "package",
        Shape: "rectangle",
    })

    // Extract imports
    ast.Inspect(node, func(n ast.Node) bool {
        if imp, ok := n.(*ast.ImportSpec); ok {
            importPath := strings.Trim(imp.Path.Value, `"`)

            // Classify import
            nodeType := "package"
            shape := "rectangle"

            if strings.Contains(importPath, "github.com") {
                nodeType = "external"
                shape = "circle"
            } else if strings.Contains(importPath, "database/sql") ||
                     strings.Contains(importPath, "gorm.io") {
                nodeType = "database"
                shape = "cylinder"
            }

            // Add edge
            graph.Edges = append(graph.Edges, GraphEdge{
                From:  importPath,
                To:    currentPackage,
                Label: "imports",
                Style: "dashed",
            })
        }
        return true
    })

    return graph, nil
}
```

#### 3.4.2 Mermaid 输出

```go
// MermaidFormatter formats graphs as Mermaid diagrams
type MermaidFormatter struct{}

func (f *MermaidFormatter) FormatDiagram(graph *ImportGraph) (string, error) {
    var sb strings.Builder

    sb.WriteString("graph TD\n")
    sb.WriteString("    %%{ init: { 'theme': 'base', 'themeVariables': { 'primaryColor': '#ff6b6b' } } }%%\n\n")

    // Define styles
    sb.WriteString("    classDef packageStyle fill:#e1f5fe,stroke:#01579b,stroke-width:2px\n")
    sb.WriteString("    classDef serviceStyle fill:#f3e5f5,stroke:#4a148c,stroke-width:2px\n")
    sb.WriteString("    classDef databaseStyle fill:#e8f5e9,stroke:#1b5e20,stroke-width:2px\n")
    sb.WriteString("    classDef externalStyle fill:#fff3e0,stroke:#e65100,stroke-width:2px,stroke-dasharray: 5 5\n\n")

    // Add nodes
    for _, node := range graph.Nodes {
        styleClass := node.Type + "Style"
        sb.WriteString(fmt.Sprintf("    %s[%s]::%s\n",
            escapeID(node.ID), node.Label, styleClass))
    }

    // Add edges
    for _, edge := range graph.Edges {
        style := ""
        if edge.Style == "dashed" {
            style = "-.->"
        } else {
            style = "-->"
        }
        sb.WriteString(fmt.Sprintf("    %s %s|%s| %s\n",
            escapeID(edge.From), style, edge.Label, escapeID(edge.To)))
    }

    return sb.String(), nil
}

func escapeID(id string) string {
    // Replace special characters for Mermaid
    id = strings.ReplaceAll(id, ".", "_")
    id = strings.ReplaceAll(id, "/", "_")
    id = strings.ReplaceAll(id, "-", "_")
    return id
}
```

### 3.5 Changelog 生成

```go
// ChangelogGenerator generates changelog from git history
type ChangelogGenerator struct {
    gitClient   *GitClient
    formatters   map[string]ChangelogFormatter
}

type ChangelogEntry struct {
    Version     string
    Date        time.Time
    Type        string // "added", "changed", "deprecated", "removed", "fixed", "security"
    Category    string // "feature", "bugfix", "performance", "docs", "breaking"
    Message     string
    PR          string
    Author      string
    Breaking    bool
}

type ChangelogFormatter interface {
    Format(entries []ChangelogEntry) (string, error)
}

// KeepALikeFormatter formats changelog in Keep A Changelog format
type KeepALikeFormatter struct{}

func (f *KeepALikeFormatter) Format(entries []ChangelogEntry) (string, error) {
    var sb strings.Builder

    // Group by version
    versions := groupByVersion(entries)

    for i, version := range versions {
        sb.WriteString(fmt.Sprintf("## [%s] - %s\n", version.Version,
            version.Date.Format("2006-01-02")))

        // Group by category
        byCategory := groupByCategory(version.Entries)

        for cat, entries := range byCategory {
            sb.WriteString(fmt.Sprintf("\n### %s\n", toTitle(cat)))
            for _, entry := range entries {
                prefix := ""
                if entry.Breaking {
                    prefix = "BREAKING: "
                }
                sb.WriteString(fmt.Sprintf("- %s%s ([#%s])\n",
                    prefix, entry.Message, entry.PR))
            }
        }

        sb.WriteString("\n")
    }

    return sb.String(), nil
}
```

### 3.6 SKILL.md 定义

```markdown
---
name: "doc-generator"
version: "1.0.0"
description: "Generate documentation, API specs, and architecture diagrams from code"
author: "cicd-ai-toolkit"
license: "MIT"

options:
  thinking:
    budget_tokens: 6144
  temperature: 0.1

tools:
  allow: ["read", "grep", "ls", "bash"]

inputs:
  - name: target_path
    type: string
    description: "Path to generate documentation for"
  - name: doc_type
    type: string
    description: "Type of documentation (api, architecture, changelog, all)"
  - name: format
    type: string
    description: "Output format (openapi, mermaid, markdown)"
  - name: output_path
    type: string
    description: "Output path for generated documentation"
---

# Doc Generator

You are an expert technical writer. Generate comprehensive documentation from code.

## Analysis Steps

1. **Language Detection**
   - Identify the programming language
   - Detect frameworks and libraries used

2. **API Documentation** (for doc_type="api")
   - Parse HTTP handlers and endpoints
   - Extract request/response schemas
   - Generate OpenAPI specification

3. **Architecture Diagrams** (for doc_type="architecture")
   - Analyze package/module structure
   - Build import dependency graph
   - Generate Mermaid diagram

4. **Changelog** (for doc_type="changelog")
   - Analyze git commit history
   - Categorize changes (features, bugfixes, breaking)
   - Generate Keep A Changelog format

## Output Format

Output in XML-wrapped JSON:

```xml
<json>
{
  "documentation": [
    {
      "type": "api|architecture|changelog",
      "format": "openapi|mermaid|markdown",
      "content": "Generated documentation content",
      "file": "path/to/output/file",
      "language": "go|python|typescript|java"
    }
  ],
  "summary": {
    "total_files": 3,
    "endpoints_documented": 15,
    "diagrams_generated": 2
  }
}
</json>
```
```

### 3.7 执行流程

```go
func (dg *DocGenerator) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResult, error) {
    // 1. Detect language
    language := detectLanguage(req.TargetPath)

    // 2. Get parser for language
    parser := dg.parsers[language]
    if parser == nil {
        return nil, fmt.Errorf("unsupported language: %s", language)
    }

    // 3. Parse source files
    var docs []Documentation

    switch req.DocType {
    case "api", "all":
        apiSpec, _ := parser.ParseAPISpec(req.Source)
        docs = append(docs, Documentation{
            Type: "api",
            Format: "openapi",
            Content: dg.formatters["openapi"].Format(apiSpec),
            File: filepath.Join(req.OutputPath, "openapi.json"),
        })

    case "architecture", "all":
        graph, _ := parser.ParseImports(req.Source)
        docs = append(docs, Documentation{
            Type: "architecture",
            Format: "mermaid",
            Content: dg.formatters["mermaid"].FormatDiagram(graph),
            File: filepath.Join(req.OutputPath, "architecture.mmd"),
        })

    case "changelog", "all":
        entries, _ := dg.gitClient.GetCommitHistory(req.Since)
        content, _ := dg.formatters["keepachangelog"].Format(entries)
        docs = append(docs, Documentation{
            Type: "changelog",
            Format: "markdown",
            Content: content,
            File: filepath.Join(req.OutputPath, "CHANGELOG.md"),
        })
    }

    // 4. Write files
    for _, doc := range docs {
        os.WriteFile(doc.File, []byte(doc.Content), 0644)
    }

    return &GenerateResult{Documents: docs}, nil
}
```

### 3.8 集成示例

```yaml
# .github/workflows/generate-docs.yml
name: Generate Documentation

on:
  push:
    branches: [main]

jobs:
  docs:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Generate API Documentation
        uses: cicd-ai-toolkit/action@v1
        with:
          run_skills: "doc-generator"
          doc_type: "api"
          format: "openapi"
          output_path: "./docs/api"

      - name: Generate Architecture Diagram
        uses: cicd-ai-toolkit/action@v1
        with:
          run_skills: "doc-generator"
          doc_type: "architecture"
          format: "mermaid"
          output_path: "./docs/architecture"

      - name: Update Changelog
        uses: cicd-ai-toolkit/action@v1
        with:
          run_skills: "doc-generator"
          doc_type: "changelog"
          output_path: "./CHANGELOG.md"
```

## 4. 依赖关系 (Dependencies)

- **Related**: [SPEC-SKILL-01](./SPEC-SKILL-01-Skill_Definition.md) - Skill 定义标准
- **Related**: [SPEC-LIB-01](./SPEC-LIB-01-Standard_Skills.md) - Standard Skills 库

## 5. 验收标准 (Acceptance Criteria)

1. **Go API 文档**: 能解析 Go HTTP handler 并生成有效的 OpenAPI 3.0 规范
2. **Python FastAPI**: 能识别 FastAPI 装饰器并提取路由定义
3. **架构图**: 能生成包含节点和边的 Mermaid 流程图
4. **Changelog**: 能按 Keep A Changelog 格式生成变更日志
5. **多语言**: 支持 Go、Python、TypeScript、Java 中的至少 2 种
6. **输出验证**: 生成的 OpenAPI 文档能通过 `swagger-cli validate`
7. **Mermaid 渲染**: 生成的 Mermaid 图能在 Mermaid Live Editor 中正确显示
