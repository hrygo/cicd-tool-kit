# SPEC-LIB-01: Standard Skills Library

**Version**: 1.1
**Status**: Draft
**Date**: 2026-01-24
**Changelog**:
- v1.1: Added 3.5 Issue Triage skill definition

## 1. 概述 (Overview)
为了保证开箱即用的体验，`cicd-ai-toolkit` 必须内置一组高质量的 Standard Skills。这些 Skill 经过精心调优，覆盖最常见的 CI 场景。

## 2. 核心职责 (Core Responsibilities)
- **High Quality**: 经过大量 Real-world PR 验证的 Prompts。
- **Robustness**: 包含详细的边界情况处理指令。
- **Standardization**: 作为社区开发 Custom Skill 的参考模版。

## 3. 详细设计 (Detailed Design)

### 3.1 Code Reviewer (`code-reviewer`)
*   **Goal**: 发现 Security, Logic, Performance 问题（忽略 Style/Lint 问题）。
*   **Key Heuristics**:
    *   IF modification involves SQL -> Check for Injection.
    *   IF modification involves Loop -> Check for Bounds/Termination.
    *   IF modification involves Refactor -> Check for Behavioral Changes.
*   **Severity Definition**:
    *   `Critical`: Crash, Data Loss, Security Breach.
    *   `High`: Business Logic Error, Major Perf Regression.

### 3.2 Change Analyzer (`change-analyzer`)
*   **Goal**: 为 PR 生成摘要，并更新 Release Notes。
*   **Prompt Strategy**:
    *   "Summarize the *Intent* (Why), not just the *Mechanics* (What)."
    *   Identify breaking changes explicitly.
*   **Output**:
    *   `summary`: Markdown paragraph.
    *   `labels`: Suggested labels (e.g., `bug`, `feature`, `breaking`).

### 3.3 Test Generator (`test-generator`)

**Covers**: PRD 1.3 (测试用例生成), 2.2 (Phase 1)

#### 概述
Test Generator Skill 自动为新增代码生成单元测试，通过分析现有测试模式、代码结构和业务逻辑，生成高质量、可维护的测试用例。

#### 目标
- 为新增函数/方法自动生成单元测试
- 检测未测试的代码路径
- 匹配项目现有测试风格
- 生成边界条件和异常用例

#### 架构设计

```go
// TestGenerator Core Logic
type TestGenerator struct {
    analyzer     *CodeAnalyzer
    patternMatch *TestPatternMatcher
    framework    TestFramework
    templates    map[string]*TestTemplate
}

type TestFramework interface {
    Name() string
    Detect() bool
    GenerateTests(spec *TestSpec) (string, error)
    GetImports() []string
    AssertFormat() string
}

// Supported Frameworks
type GoTestFramework struct{}
type JestFramework struct{}
type PytestFramework struct{}
type JUnitFramework struct{}
```

#### 测试框架检测策略

| 语言 | 检测文件/配置 | 框架优先级 |
|------|--------------|------------|
| **Go** | `*_test.go` 文件 | testing (标准库) |
| **JavaScript/TypeScript** | `jest.config.js`, `package.json` (jest) | Jest > Mocha > Vitest |
| **Python** | `conftest.py`, `pytest.ini` | pytest > unittest |
| **Java** | `pom.xml` (JUnit), `build.gradle` | JUnit 5 > TestNG |
| **Rust** | `Cargo.toml` (dev-dependencies) | rust-testing |

```go
func (tg *TestGenerator) DetectFramework(language string) TestFramework {
    switch language {
    case "go":
        return &GoTestFramework{}
    case "javascript", "typescript":
        if fileExists("jest.config.js") || hasDevDependency("jest") {
            return &JestFramework{}
        }
        return &JestFramework{} // Default to Jest
    case "python":
        if fileExists("pytest.ini") || fileExists("conftest.py") {
            return &PytestFramework{}
        }
        return &PytestFramework{} // Default to pytest
    // ... more languages
    }
    return nil
}
```

#### 代码分析策略

**1. 函数签名分析**

```go
type FunctionSignature struct {
    Name       string
    Package    string
    Receiver   string
    Parameters []Parameter
    Returns    []Type
    Visibility string
    IsMethod   bool
}

type Parameter struct {
    Name     string
    Type     Type
    IsVariadic bool
}

func (a *CodeAnalyzer) AnalyzeFunction(ast *ast.Node) *FunctionSignature {
    // Parse function signature
    // Extract parameter types
    // Extract return types
    // Determine visibility (public/private)
}
```

**2. 测试点生成算法**

```go
type TestPoint struct {
    Description string
    Input       map[string]interface{}
    Expected    interface{}
    Setup       string
    Teardown    string
    Category    string // "happy", "edge", "error", "boundary"
}

func (tg *TestGenerator) GenerateTestPoints(fn *FunctionSignature) []TestPoint {
    var points []TestPoint

    // Happy path test
    points = append(points, TestPoint{
        Description: fmt.Sprintf("%s should succeed with valid input", fn.Name),
        Category:    "happy",
        Input:       tg.generateValidInput(fn),
        Expected:    tg.generateExpectedOutput(fn),
    })

    // Edge cases for each parameter
    for _, param := range fn.Parameters {
        points = append(points, tg.generateEdgeCases(param)...)
    }

    // Error cases (if function returns error)
    if tg.returnsError(fn) {
        points = append(points, tg.generateErrorCases(fn)...)
    }

    // Boundary cases
    for _, param := range fn.Parameters {
        if tg.isNumericType(param.Type) {
            points = append(points, tg.generateBoundaryCases(param)...)
        }
    }

    return points
}
```

**3. 现有测试模式匹配**

```go
type TestPattern struct {
    FilePattern    string
    NamePattern    string
    Structure      string // "table-driven", "BDD", "AAA"
    Imports        []string
    CommonSetup    string
    CommonTeardown string
}

func (pm *TestPatternMatcher) MatchExisting() *TestPattern {
    // Scan for existing test files
    testFiles := findFiles("*_test.go", "*.test.ts", "*_test.py")

    if len(testFiles) == 0 {
        return pm.getDefaultPattern()
    }

    // Analyze structure
    pattern := &TestPattern{}

    for _, file := range testFiles {
        content := readFile(file)

        // Detect structure
        if containsTableDriven(content) {
            pattern.Structure = "table-driven"
        } else if containsBDD(content) {
            pattern.Structure = "BDD"
        }

        // Extract common imports
        pattern.Imports = extractImports(content)

        // Extract common setup/teardown
        pattern.CommonSetup = extractSetupBlock(content)
        pattern.CommonTeardown = extractTeardownBlock(content)
    }

    return pattern
}
```

#### Go 测试生成详细设计

**Table-Driven Pattern**

```go
// Go Table-Driven Test Template
const goTableDrivenTemplate = `func Test{{.FunctionName}}(t *testing.T) {
{{if .HasSetup}}{{.SetupCode}}{{end}}

    tests := []struct {
        name    string
{{range .Params}}        {{.Name}} {{.Type}}
{{end}}        want    {{.ReturnType}}
        wantErr bool
    }{
{{range .TestCases}}        {
            name: "{{.Description}}",
{{range .Inputs}}            {{.ParamName}}: {{.Value}},
{{end}}            want:    {{.Expected}},
            wantErr: {{.ExpectError}},
        },
{{end}}    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
{{if .HasPreparation}}            // Setup
{{.PreparationCode}}

{{end}}            got, err := {{.FunctionCall}}
            if (err != nil) != tt.wantErr {
                t.Errorf("{{.FunctionName}}() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("{{.FunctionName}}() = %v, want %v", got, tt.want)
            }
        })
    }
{{if .HasTeardown}}{{.TeardownCode}}{{end}}
}
`
```

**Mock 生成策略**

```go
type MockGenerator struct {
    interfaces map[string]*InterfaceDefinition
}

type InterfaceDefinition struct {
    Name       string
    Methods    []MethodSignature
    Package    string
}

func (mg *MockGenerator) GenerateMock(intf *InterfaceDefinition) string {
    // Using gomock or mockgen pattern
    template := `// Code generated by mockgen; DO NOT EDIT.

type mock{{.InterfaceName}} struct {
    ctrl     *gomock.Controller
    recorder *mock{{.InterfaceName}}MockRecorder
}

func NewMock{{.InterfaceName}}(ctrl *gomock.Controller) *mock{{.InterfaceName}} {
    mock := &mock{{.InterfaceName}}{ctrl: ctrl}
    mock.recorder = &mock{{.InterfaceName}}MockRecorder{mock}
    return mock
}
`
    // ... generate methods
    return template
}
```

#### JavaScript/TypeScript (Jest) 测试生成

**Jest Pattern**

```typescript
// Jest Test Template
const jestTemplate = `describe('{{.FunctionName}}', () => {
{{if .HasMocks}}  let {{.MockVariables}};

  beforeEach(() => {
{{.MockSetup}}
  });

  afterEach(() => {
{{.MockTeardown}}
  });

{{end}}{{range .TestCases}}  it('{{.Description}}', async () => {
{{if .Setup}}    // Arrange
{{.Setup}}

{{end}}    // Act
    const result = await {{.FunctionCall}};

    // Assert
{{if .HasExpectations}}{{.Expectations}}{{end}}
  });

{{end}}});
`;
```

#### Python (Pytest) 测试生成

**Pytest Pattern**

```python
# Pytest Test Template
pytest_template = """
import pytest
{{if .HasImports}}{{.Imports}}

{{end}}class Test{{.ClassName}}:
{{if .HasFixtures}}    @pytest.fixture
    def {{.FixtureName}}(self):
        {{.FixtureCode}}

{{end}}{{range .TestCases}}    def test_{{.TestCaseName}}(self{{if .NeedsFixture}}, {{.FixtureParam}}{{end}}):
{{if .HasArrange}}        # Arrange
{{.Arrange}}

{{end}}        # Act
        result = {{.FunctionCall}}

        # Assert
{{.Assert}}
{{end}}
"""
```

#### 测试点生成规则

| 场景 | 生成的测试点 |
|------|-------------|
| **字符串参数** | 空字符串、长字符串、特殊字符、nil/null |
| **数值参数** | 0、负数、最大值、最小值、边界值 |
| **切片/数组** | 空、单项、多项 |
| **Map/Struct** | 空、单项、嵌套 |
| **错误返回** | nil error、各种 error 类型 |
| **布尔值** | true、false 两分支 |
| **枚举** | 每个枚举值 |

```go
func (tg *TestGenerator) generateEdgeCases(param Parameter) []TestPoint {
    var points []TestPoint

    switch param.Type.Kind {
    case reflect.String:
        points = append(points,
            TestPoint{Description: "with empty string", Input: map[string]interface{}{param.Name: ""}, Category: "edge"},
            TestPoint{Description: "with long string", Input: map[string]interface{}{param.Name: strings.Repeat("a", 10000)}, Category: "boundary"},
        )

    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        points = append(points,
            TestPoint{Description: "with zero", Input: map[string]interface{}{param.Name: 0}, Category: "edge"},
            TestPoint{Description: "with negative", Input: map[string]interface{}{param.Name: -1}, Category: "edge"},
            TestPoint{Description: "with max value", Input: map[string]interface{}{param.Name: math.MaxInt64}, Category: "boundary"},
        )

    case reflect.Slice, reflect.Array:
        points = append(points,
            TestPoint{Description: "with empty slice", Input: map[string]interface{}{param.Name: []interface{}{}}, Category: "edge"},
            TestPoint{Description: "with single item", Input: map[string]interface{}{param.Name: []interface{}{mockItem()}}, Category: "edge"},
        )

    case reflect.Interface, reflect.Pointer:
        points = append(points,
            TestPoint{Description: "with nil", Input: map[string]interface{}{param.Name: nil}, Category: "edge"},
        )
    }

    return points
}
```

#### 完整 SKILL.md 定义

```markdown
---
name: "test-generator"
version: "1.2.0"
description: "Generate unit tests based on code changes"
author: "cicd-ai-team"
license: "MIT"

options:
  thinking:
    budget_tokens: 8192
  temperature: 0.1

tools:
  allow: ["read", "grep", "ls", "bash"]

inputs:
  - name: target_path
    type: string
    description: "Path to generate tests for (default: all changes)"
  - name: test_framework
    type: string
    description: "Test framework to use (auto, jest, pytest, go-test, junit)"
  - name: coverage_mode
    type: string
    description: "Coverage strategy (all, untracked, changed)"
  - name: generate_mocks
    type: boolean
    description: "Generate mock objects for interfaces"
  - name: output_path
    type: string
    description: "Output path for generated tests"
---

# Test Generator

You are an expert test engineer. Your goal is to generate comprehensive, maintainable unit tests for code changes.

## Analysis Steps

1. **Language & Framework Detection**
   - Identify the programming language from file extensions
   - Detect the test framework used in the project
   - Analyze existing test files for patterns

2. **Code Analysis**
   - Parse function/method signatures
   - Identify parameters, return types, error handling
   - Map dependencies and interfaces

3. **Test Planning**
   - Generate happy path test cases
   - Generate edge cases (empty, nil, boundaries)
   - Generate error cases
   - Identify need for mocks/fixtures

4. **Test Generation**
   - Follow existing project conventions
   - Use appropriate assertions
   - Include descriptive test names
   - Add setup/teardown if needed

## Test Generation Rules

### Go
- Use table-driven tests for multiple cases
- Use `t.Run()` for subtests
- Use `require` for fatal errors, `assert` for non-fatal
- Follow `Test<FunctionName>` naming

### JavaScript/TypeScript (Jest)
- Use `describe`/`it` or `test` structure
- Use descriptive test names
- Mock external dependencies
- Use `async`/`await` for async tests

### Python (Pytest)
- Use class-based tests for related tests
- Use fixtures for setup/teardown
- Use descriptive test names
- Follow `test_<function>` naming

## Output Format

You must output in XML-wrapped JSON:

\`\`\`xml
<json>
{
  "tests": [
    {
      "file": "path/to/test_file.ext",
      "content": "// Full test file content",
      "framework": "go-test|jest|pytest|junit",
      "language": "go|javascript|python|java",
      "imports": ["import1", "import2"],
      "functions_tested": ["Function1", "Function2"],
      "mocks_required": [
        {
          "interface": "Repository",
          "file": "mocks/mock_repository.go"
        }
      ],
      "coverage_estimate": {
        "functions": 5,
        "branches": 8,
        "statements": 25
      }
    }
  ],
  "summary": {
    "total_tests": 15,
    "files_created": 3,
    "framework": "jest",
    "confidence": 0.9
  }
}
</json>
\`\`\`
```

#### 执行流程

```go
func (tg *TestGenerator) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResult, error) {
    // 1. Detect language and framework
    language := detectLanguage(req.TargetPath)
    framework := tg.DetectFramework(language)

    // 2. Parse source files
    sourceFiles := findSourceFiles(req.TargetPath, language)
    var functions []FunctionSignature
    for _, file := range sourceFiles {
        funcs, _ := tg.analyzer.ParseFile(file)
        functions = append(functions, funcs...)
    }

    // 3. Match existing test patterns
    pattern := tg.patternMatch.MatchExisting()

    // 4. Generate test specs
    var tests []*TestSpec
    for _, fn := range functions {
        testPoints := tg.GenerateTestPoints(&fn)
        spec := &TestSpec{
            Function:    fn,
            Framework:   framework,
            Pattern:     pattern,
            TestPoints:  testPoints,
        }
        tests = append(tests, spec)
    }

    // 5. Generate test code
    var generatedTests []*GeneratedTest
    for _, spec := range tests {
        code, _ := framework.GenerateTests(spec)
        generatedTests = append(generatedTests, &GeneratedTest{
            File:    tg.getTestFilePath(spec.Function),
            Content: code,
        })
    }

    // 6. Generate mocks if needed
    if req.GenerateMocks {
        mocks := tg.generateMocks(tests)
        generatedTests = append(generatedTests, mocks...)
    }

    return &GenerateResult{Tests: generatedTests}, nil
}
```

#### 验收标准

1. **框架检测**: 能正确识别项目的测试框架
2. **模式匹配**: 生成的测试风格与现有测试一致
3. **测试覆盖**: 生成的测试覆盖正常路径、边界条件和错误情况
4. **可编译性**: 生成的测试代码能通过编译（无语法错误）
5. **Go 测试**: 生成的 Go 测试能通过 `go test`（无断言失败）
6. **Jest 测试**: 生成的 Jest 测试语法正确
7. **Mock 生成**: 为接口生成正确的 mock 代码
8. **测试独立性**: 每个测试用例独立运行

### 3.4 Log Analyzer (`log-analyzer`)
*   **Goal**: 分析 CI/CD 日志，定位构建失败或运行时错误的根因。
*   **Strategy**:
    1.  **Pattern Recognition**: 识别常见的 Stack Trace 格式 (Java, Python, Go).
    2.  **Noise Filtering**: 忽略常规的 Info/Debug 日志，聚焦 Error/Fatal.
    3.  **Context Mapping**: 尝试将日志报错行号映射回源代码变更 (Diff).
*   **Output**:
    *   `root_cause`: 一句话总结错误原因。
    *   `fix_suggestion`: 基于错误的修复建议。

### 3.5 Issue Triage (`issue-triage`)

**Covers**: PRD 1.3 (自动化 Issue 分类), 2.2

#### 概述
Issue Triage Skill 自动分析新创建的 Issue，提取关键信息并添加标签、优先级和分配建议。这有助于维护团队快速响应和分类 Issue。

#### 目标
- 自动识别 Issue 类别 (bug/feature/improvement/documentation)
- 推荐优先级 (P0-P3)
- 建议分配给合适的团队/成员
- 识别重复 Issue

#### Prompt Strategy

```
You are an Issue Triage Bot. Analyze the provided Issue and extract structured information.

## Analysis Dimensions

1. **Category Classification**
   - `bug`: Something is broken or not working as expected
   - `feature`: Request for new functionality
   - `improvement`: Enhancement to existing functionality
   - `documentation`: Docs, README, API reference issues
   - `question`: User asking for help
   - `performance`: Slowdown, memory, efficiency issues
   - `security`: Security vulnerabilities or concerns

2. **Priority Assessment**
   - `P0 (Critical)`: Production down, data loss, security breach
   - `P1 (High)`: Major feature broken, significant user impact
   - `P2 (Medium)`: Minor issues, workaround exists
   - `P3 (Low)`: Nice to have, cosmetic, edge cases

3. **Component Tagging**
   Based on keywords and file paths, suggest component labels:
   - `auth`, `database`, `api`, `ui`, `ci/cd`, `infra`, etc.

4. **Duplicate Detection**
   Check if this Issue appears similar to known patterns or recent issues.
```

#### 输入格式

```yaml
inputs:
  - name: issue_title
    type: string
    description: "The issue title"
  - name: issue_body
    type: string
    description: "The issue description/body"
  - name: issue_author
    type: string
    description: "Who created the issue (optional)"
  - name: recent_issues
    type: array
    description: "Recent issues for duplicate detection (optional)"
```

#### 输出格式

```json
{
  "analysis": {
    "category": "bug|feature|improvement|documentation|question|performance|security",
    "confidence": 0.95,
    "reasoning": "Brief explanation of classification"
  },
  "priority": {
    "level": "P0|P1|P2|P3",
    "reasoning": "Why this priority was chosen"
  },
  "labels": [
    "bug",
    "database",
    "high-severity"
  ],
  "assignee_suggestion": {
    "team": "backend",
    "reasoning": "Issue involves database queries"
  },
  "duplicate_check": {
    "is_duplicate": false,
    "similar_issues": [
      {
        "id": "123",
        "similarity": 0.85,
        "reason": "Both describe login timeout issues"
      }
    ]
  },
  "summary": "One-line summary of the issue",
  "suggested_response": {
    "should_respond": true,
    "template": "acknowledgment|clarification|estimate",
    "message": "Draft response to the issue author"
  }
}
```

#### SKILL.md 模板

```markdown
---
name: "issue-triage"
version: "1.0.0"
description: "Automatically categorize and prioritize issues"
author: "cicd-ai-team"
license: "MIT"

options:
  thinking:
    budget_tokens: 2048
  temperature: 0.1  # Low temperature for consistent classification

tools:
  allow: ["grep", "cat"]  # Read-only for checking similar issues
---

# Issue Triage Bot

You are an intelligent issue triage assistant. Your goal is to analyze incoming issues and provide structured classification.

## Classification Guidelines

### Categories

| Category | When to Use |
|----------|-------------|
| `bug` | Error, crash, unexpected behavior |
| `feature` | New capability request |
| `improvement` | Enhancement to existing feature |
| `documentation` | Docs, guides, README issues |
| `question` | User asking for help/guidance |
| `performance` | Slowdown, memory, efficiency |
| `security` | Vulnerability, exploit, auth issue |

### Priority Matrix

| Severity | User Impact | Workaround | Priority |
|----------|-------------|------------|----------|
| Complete loss | All users | None | P0 |
| Major feature | Many users | Difficult | P1 |
| Minor feature | Some users | Easy | P2 |
| Cosmetic | Few users | N/A | P3 |

## Output Format

You must output in XML-wrapped JSON:

```xml
<json>
{
  "analysis": {...},
  "priority": {...},
  "labels": [...],
  "assignee_suggestion": {...},
  "duplicate_check": {...},
  "summary": "...",
  "suggested_response": {...}
}
</json>
```
```

#### 触发方式

GitHub Actions / GitLab CI 集成：

```yaml
name: Issue Triage
on:
  issues:
    types: [opened, edited]

jobs:
  triage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: AI Issue Triage
        uses: cicd-ai-toolkit/action@v1
        with:
          run_skills: "issue-triage"
          issue_title: ${{ github.event.issue.title }}
          issue_body: ${{ github.event.issue.body }}
          github_token: ${{ secrets.GITHUB_TOKEN }}
```

#### 验收标准

1. **分类准确**: 提供明确描述 Bug 的 Issue，应正确分类为 `category: "bug"`
2. **优先级合理**: 生产环境崩溃问题应标记为 `P0`，拼写问题应为 `P3`
3. **标签相关性**: 数据库相关 Issue 应包含 `database` 标签
4. **重复检测**: 与已存在 Issue 相似度 >80% 时，应标记为 `is_duplicate: true`
5. **响应质量**: 对不清楚的 Issue，应生成合理的澄清问题

## 4. 依赖关系 (Dependencies)
- **Schema**: 必须符合 [SPEC-SKILL-01](./SPEC-SKILL-01-Skill_Definition.md) 定义。

## 5. 验收标准 (Acceptance Criteria)
1.  **Review Quality**: 对含有人为注入的 SQL 注入漏洞代码运行 `code-reviewer`，必须检出 Critical Issue。
2.  **Summary Relevance**: 对重构代码运行 `change-analyzer`，摘要应指出 "Refactoring for readability" 而非罗列 "Changed line 10, Changed line 20"。
3.  **Test Validity**: `test-generator` 生成的 Go 测试代码能通过 `go test`（假设无复杂外部依赖）。
4.  **Issue Classification**: 对明确描述 Bug 的 Issue 运行 `issue-triage`，应正确分类为 `category: "bug"` 并推荐合理的优先级。
5.  **Label Accuracy**: 数据库相关的 Issue 应包含 `database` 标签。
6.  **Duplicate Detection**: 提交与已有 Issue 高度相似的 Issue，应标记 `is_duplicate: true`。
