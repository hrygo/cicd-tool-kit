# SPEC-PLAT-04: Jenkins Plugin

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24
**Covers**: PRD 2.1 (Legacy Support)

## 1. 概述 (Overview)

Jenkins 是传统的 CI/CD 平台，许多企业仍大量使用。与 GitHub/GitLab 不同，Jenkins 采用 Plugin 机制而非 Webhook。本 Spec 定义 Jenkins 插件的实现方案。

## 2. 核心职责 (Core Responsibilities)

- **Plugin Development**: 开发 Jenkins Plugin 封装 `cicd-ai-toolkit`
- **SCM Integration**: 与 Git SCM 集成获取变更内容
- **Build Trigger**: 在构建阶段触发 AI 分析
- **Result Publishing**: 将分析结果发布到 Jenkins UI 和 Git 平台

## 3. 详细设计 (Detailed Design)

### 3.1 架构方案 (Architecture)

由于 Jenkins Plugin 开发成本高（Java 生态），采用 **Hybrid 架构**：

```
┌─────────────────────────────────────────────────────────┐
│                    Jenkins Master                        │
│  ┌──────────────────────────────────────────────────┐   │
│  │         cicd-ai-toolkit Jenkins Plugin           │   │
│  │  - BuildWrapper (环境准备)                       │   │
│  │  - BuildStep (触发 Runner)                       │   │
│  │  - Post-Build Action (发布结果)                  │   │
│  └──────────────────────────────────────────────────┘   │
│                         │                                 │
│                         ▼                                 │
│  ┌──────────────────────────────────────────────────┐   │
│  │         cicd-runner (Go Binary)                   │   │
│  │  - 在 Agent 上运行                                 │   │
│  │  - 读取 Jenkins 环境变量                           │   │
│  │  - 调用 Claude Code                                │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### 3.2 Plugin 组件 (Plugin Components)

#### 3.2.1 BuildWrapper

```java
public class CicdAiBuildWrapper extends SimpleBuildWrapper {
    // 准备 AI 分析环境
    @Override
    public void setUp(Context context, Run<?,?> build, FilePath workspace, Launcher launcher, TaskListener listener) throws IOException, InterruptedException {

        // 1. 获取 SCM 变更
        SCMRevisionAction revisionAction = build.getAction(SCMRevisionAction.class);
        if (revisionAction != null) {
            // 获取变更内容，写入临时文件供 Runner 使用
            List<ChangeLogSet<? extends ChangeLogSet.Entry>> changeSets = build.getChangeSets();
            writeDiffToFile(workspace, changeSets);
        }

        // 2. 设置环境变量
        EnvVars env = build.getEnvironment(listener);
        env.put("CICD_PLATFORM", "jenkins");
        env.put("CICD_BUILD_URL", build.getAbsoluteUrl());
        env.put("CICD_PROJECT_NAME", build.getParent().getFullName());

        // 3. 注入必要的环境变量到构建环境
        context.env.put("CICD_AI_ENABLED", "true");
    }
}
```

#### 3.2.2 Builder (BuildStep)

```java
public class CicdAiBuilder extends Builder implements SimpleBuildStep {

    private final String skills;
    private final String configPath;
    private final boolean failOnError;

    @DataBoundConstructor
    public CicdAiBuilder(String skills, String configPath, boolean failOnError) {
        this.skills = skills;
        this.configPath = configPath;
        this.failOnError = failOnError;
    }

    @Override
    public void perform(Run<?,?> build, FilePath workspace, Launcher launcher, TaskListener listener) throws InterruptedException, IOException {

        listener.getLogger().println("Starting cicd-ai-toolkit analysis...");

        // 构建 Runner 命令
        ArgumentListBuilder args = new ArgumentListBuilder();
        args.add("cicd-runner");
        args.add("--platform", "jenkins");
        args.add("--skills", skills);

        if (configPath != null && !configPath.isEmpty()) {
            args.add("--config", workspace.child(configPath).getRemote());
        }

        // 在 workspace 中执行
        Proc proc = launcher.launch().cmds(args).pwd(workspace).stdout(listener).join();

        int exitCode = proc.join();
        if (exitCode != 0 && failOnError) {
            listener.error("cicd-ai-toolkit failed with exit code: " + exitCode);
            build.setResult(Result.FAILURE);
        }
    }

    // UI 配置支持
    @Symbol("cicdAi")
    @Extension
    public static final class DescriptorImpl extends BuildStepDescriptor<Builder> {
        @Override
        public boolean isApplicable(Class<? extends AbstractProject> jobType) {
            return true;
        }

        @Override
        public String getDisplayName() {
            return "AI Code Review";
        }

        // jelly 配置表单
    }
}
```

### 3.3 Jenkins Adapter 实现

```go
// JenkinsAdapter implements Platform for Jenkins
type JenkinsAdapter struct {
    client    *http.Client
    baseURL   string
    username  string
    token     string
    buildURL  string
    project   string
    buildNum  int
}

func NewJenkinsAdapter() (*JenkinsAdapter, error) {
    baseURL := os.Getenv("JENKINS_URL")
    if baseURL == "" {
        return nil, fmt.Errorf("JENKINS_URL not set")
    }

    // Jenkins uses username:token or API token
    username := os.Getenv("JENKINS_USERNAME")
    token := os.Getenv("JENKINS_TOKEN")

    buildURL := os.Getenv("CICD_BUILD_URL")
    project := os.Getenv("CICD_PROJECT_NAME")
    buildNum, _ := strconv.Atoi(os.Getenv("BUILD_ID"))

    return &JenkinsAdapter{
        client:   &http.Client{},
        baseURL:  strings.TrimSuffix(baseURL, "/"),
        username: username,
        token:    token,
        buildURL: buildURL,
        project:  project,
        buildNum: buildNum,
    }, nil
}

func (j *JenkinsAdapter) GetPullRequest(ctx context.Context, id string) (*UnifiedPR, error) {
    // Jenkins SCM plugin can detect branch/PR info
    // Use change log API to get details

    url := fmt.Sprintf("%s/job/%s/%d/api/json", j.baseURL, j.project, j.buildNum)
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    req.SetBasicAuth(j.username, j.token)

    resp, err := j.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var buildInfo struct {
        Actions []struct {
            Causes []struct {
                ShortDescription string `json:"shortDescription"`
            } `json:"causes"`
        } `json:"actions"`
        ChangeSet struct {
            Items []struct {
                CommitID string `json:"commitId"`
                Msg      string `json:"msg"`
                Author   struct {
                    FullName string `json:"fullName"`
                } `json:"author"`
            } `json:"items"`
        } `json:"changeSet"`
    }

    json.NewDecoder(resp.Body).Decode(&buildInfo)

    return &UnifiedPR{
        ID:          fmt.Sprintf("%d", j.buildNum),
        Title:       fmt.Sprintf("Build #%d", j.buildNum),
        Description: extractDescription(buildInfo.Actions),
        SHA:         buildInfo.ChangeSet.Items[0].CommitID,
        Author:      buildInfo.ChangeSet.Items[0].Author.FullName,
    }, nil
}

func (j *JenkinsAdapter) GetDiff(ctx context.Context, id string) ([]byte, error) {
    // Jenkins changelog.xml
    url := fmt.Sprintf("%s/job/%s/%d/changes", j.baseURL, j.project, j.buildNum)
    // ... fetch and parse
    return nil, nil
}

func (j *JenkinsAdapter) PostComment(ctx context.Context, id string, body string) error {
    // Add to build description or use Markdown Plugin
    url := fmt.Sprintf("%s/job/%s/%d/submitDescription", j.baseURL, j.project, j.buildNum)

    form := url.Values{}
    form.Add("description", body)

    req, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(form.Encode()))
    req.SetBasicAuth(j.username, j.token)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    _, err := j.client.Do(req)
    return err
}

func (j *JenkinsAdapter) CreateCheckRun(ctx context.Context, sha string, name string) (string, error) {
    // Jenkins doesn't have Check Runs equivalent
    // Use Build Description or Console Note
    return j.buildURL, nil
}

func (j *JenkinsAdapter) UpdateCheckRun(ctx context.Context, runID string, status string, output CheckOutput) error {
    // Update build description
    return j.PostComment(ctx, "", formatOutput(output))
}
```

### 3.4 Pipeline 支持 (Jenkinsfile)

```groovy
// Declarative Pipeline
pipeline {
    agent any

    stages {
        stage('AI Code Review') {
            steps {
                cicdAi(
                    skills: 'code-reviewer,change-analyzer',
                    config: '.cicd-ai-toolkit.yaml',
                    failOnError: false
                )
            }
        }
    }

    post {
        always {
            // AI 结果会自动添加到构建描述
        }
    }
}

// Scripted Pipeline
node {
    checkout scm

    // 包装整个构建
    wrap([$class: 'CicdAiBuildWrapper']) {
        sh 'make build'
        sh 'make test'

        // 或单独执行
        step([$class: 'CicdAiBuilder',
              skills: 'code-reviewer',
              config: '.cicd-ai-toolkit.yaml'])
    }
}
```

### 3.5 SCM 兼容性 (SCM Compatibility)

| SCM | 状态 | 实现方式 |
|-----|------|----------|
| **Git** | ✅ Full | Git Plugin / GitLab Branch Source |
| **Subversion** | ⚠️ Partial | SVN Plugin (diff 格式不同，需适配) |
| **Mercurial** | ⚠️ Partial | Mercurial Plugin |
| **Perforce** | ❌ Not Planned | 企业级，需单独评估 |

### 3.6 环境变量映射 (Environment Variables)

| Jenkins Env | Mapping | cicd-ai-toolkit Env |
|-------------|---------|---------------------|
| `BUILD_URL` | → | `CICD_BUILD_URL` |
| `JOB_NAME` | → | `CICD_PROJECT_NAME` |
| `BUILD_ID` | → | `CICD_BUILD_NUMBER` |
| `CHANGE_ID` | PR/MR ID | `CICD_PR_ID` |
| `CHANGE_TARGET` | Target branch | `CICD_TARGET_BRANCH` |
| `CHANGE_SOURCE` | Source branch | `CICD_SOURCE_BRANCH` |
| `GIT_COMMIT` | Commit SHA | `CICD_COMMIT_SHA` |

### 3.7 与 Git 平台集成 (Git Platform Integration)

Jenkins 可能连接到 GitHub/GitLab。Runner 需要额外与这些平台通信以发布评论：

```yaml
# .cicd-ai-toolkit.yaml (Jenkins 场景)
platform:
  jenkins:
    # Jenkins 基础信息（自动检测）
    base_url: "${JENKINS_URL}"
    username: "${JENKINS_USERNAME}"
    token: "${JENKINS_TOKEN}"

  # 如果要在 GitHub/GitLab 发表评论
  github:
    enabled: true
    token_env: "GITHUB_TOKEN"
    post_comment: true
```

## 4. Plugin 分发 (Distribution)

### 4.1 HPI 文件

编译生成 `.hpi` (Jenkins Plugin) 文件：

```bash
mvn package
# target/cicd-ai-toolkit.hpi
```

### 4.2 安装方式

1. **手动上传**: Jenkins → Plugin Manager → Advanced → Upload
2. **Update Center**: 发布到官方 Jenkins Update Center (需申请)
3. **Docker**: 预装 Plugin 的自定义镜像

```dockerfile
FROM jenkins/jenkins:lts

# Install cicd-ai-toolkit
RUN curl -fsSL https://get.cicd-toolkit.com | bash

# Install Plugin (copy .hpi to plugins dir)
COPY cicd-ai-toolkit.hpi /var/jenkins_home/ref/plugins/
```

## 5. 依赖关系 (Dependencies)

- **Plugin SDK**: `org.jenkins-ci.plugins:structs` (数据绑定)
- **Go Runner**: Runner 必须在 Jenkins Agent 上可用
- **Git Plugin**: 获取变更内容

## 6. 验收标准 (Acceptance Criteria)

1. **Freestyle Job**: 能在 Freestyle Project 中添加 "AI Code Review" 构建步骤
2. **Pipeline Job**: 能在 Jenkinsfile 中使用 `cicdAi` 步骤
3. **SCM 检测**: 能正确检测到 Git 变更并传递给 Runner
4. **结果发布**: AI 分析结果显示在构建描述中
5. **错误处理**: Runner 失败时根据配置决定是否使构建失败
6. **多分支支持**: 在多分支 Pipeline 中能正常工作
7. **环境隔离**: 不同 Job 的配置互不干扰

## 7. 限制与已知问题 (Limitations)

1. **Java 依赖**: Plugin 需要编译，更新较慢
2. **UI 简陋**: Jenkins UI 不如 GitHub Actions 现代
3. **权限模型**: Jenkins 权限复杂，需要额外适配
4. **Master/Agent**: Runner 必须在执行的 Agent 上可用
5. **异步模式**: Jenkins 不太适合异步分析（Job 会等待）

## 8. 优先级评估

由于 Jenkins 是 Legacy 支持，且维护成本高，建议：

| 场景 | 优先级 | 说明 |
|------|--------|------|
| **PoC 实现** | P2 | 验证可行性，确认需求 |
| **完整 Plugin** | P3 | 根据社区反馈决定是否投入 |
| **持续维护** | P4 | 可考虑由社区接手 |
