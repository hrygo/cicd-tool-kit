# SPEC-PLAT-02: Async Execution & Callback Mechanism

**Version**: 1.0
**Status**: Draft
**Date**: 2026-01-24

## 1. 概述 (Overview)
深度 AI 分析可能耗时较长（>60s）。为了不阻塞 CI Runner（或者在 Serverless 场景下运行），Runner 需支持 "异步处理" 模式。

## 2. 核心职责 (Core Responsibilities)
- **State Management**: 维护任务状态 (Pending -> Processing -> Completed)。
- **Feedback Loop**: 及时告知用户 "AI 正在思考"，避免误以为卡死。

## 3. 详细设计 (Detailed Design)

### 3.1 状态机流转 (State Transition)
1.  **Init (Pending)**:
    *   Runner 启动，立即调用 Platform API 创建一个 Check Run，状态设为 `queued` 或 `in_progress`。
    *   Title: "AI Review / Initializing".
2.  **Running (Processing)**:
    *   每隔 10s (Heartbeat) 更新 Check Run 的 Summary，追加日志或进度条。
    *   *目的*: 告知平台 Runner 依然存活，防止 CI Timeout。
3.  **Finalize (Completed)**:
    *   分析结束。更新 Check Run 为 `completed`，结论为 `success` 或 `failure`。
    *   Payload: 写入 Markdown 格式的详细报告。

### 3.2 两种部署模式 (Deployment Modes)

#### Mode A: CI Native (Current MVP)
Runner 在 CI 容器内运行。
*   **实现**: Go Runner 启动 Goroutine 执行 Claude 分析，主线程负责 Ticker 更新 Heartbeat。
*   **限制**: 必须在 CI Job 超时前完成。

#### Mode B: Remote Worker (Future)
Runner 仅作为 Trigger。
*   **实现**:
    1.  Runner 发送 Webhook 到 `ai-worker-service`。
    2.  Runner 退出（标记 Check 为 Pending）。
    3.  User 查看 PR 页面，Check 状态为 Pending。
    4.  Remote Worker 完成分析，调用 GitHub API 回写结果。
*   **优势**: 不消耗 CI 分钟数。

### 3.3 MVP 实现 (Mode A)
```go
func (r *Runner) ListenAndServe() {
    // Start Heartbeat
    go func() {
        for range time.Tick(10 * time.Second) {
            r.Platform.UpdateCheckRun("Still analyzing...")
        }
    }()
    
    // Blocking Call
    result := r.Claude.Execute()
    
    // Final Update
    r.Platform.CompleteCheckRun(result)
}
```

## 4. 依赖关系 (Dependencies)
- **Deps**: [SPEC-PLAT-01](./SPEC-PLAT-01-Platform_Adapter.md) 用于更新 Check 状态。

## 5. 验收标准 (Acceptance Criteria)
1.  **Immediate Feedback**: Runner 启动后 5s 内，Git 平台上应出现 "AI Review" 的 Check Run，状态为黄色 (Running)。
2.  **Heartbeat**: 模拟耗时 60s 的分析，Check Run 不应超时或消失。
3.  **Completion**: 分析完成后，Check Run 变绿 (Success) 或红 (Failure)，并显示 Result。
