// Package mcp provides tests for the MCP server
package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/platform"
)

// mockPlatform is a mock implementation of platform.Platform for testing
type mockPlatform struct{}

func (m *mockPlatform) Name() string {
	return "mock"
}

func (m *mockPlatform) PostComment(ctx context.Context, opts platform.CommentOptions) error {
	return nil
}

func (m *mockPlatform) GetDiff(ctx context.Context, prID int) (string, error) {
	return "diff --git a/file.go b/file.go\n+new line", nil
}

func (m *mockPlatform) GetFile(ctx context.Context, path, ref string) (string, error) {
	return "package main\n\nfunc main() {}", nil
}

func (m *mockPlatform) GetPRInfo(ctx context.Context, prID int) (*platform.PRInfo, error) {
	return &platform.PRInfo{
		Number:      prID,
		Title:       "Test PR",
		Description: "Test description",
		Author:      "testuser",
		SHA:         "abc123",
		BaseBranch:  "main",
		HeadBranch:  "feature",
		SourceRepo:  "test/repo",
		Labels:      []string{"test"},
	}, nil
}

func (m *mockPlatform) Health(ctx context.Context) error {
	return nil
}

// TestNewServer verifies server creation
func TestNewServer(t *testing.T) {
	mock := &mockPlatform{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer(mock, logger)

	if server == nil {
		t.Fatal("NewServer should not return nil")
	}

	tools := server.ListTools()
	if len(tools) == 0 {
		t.Error("Server should register default tools")
	}
}

// TestListTools verifies tools are properly listed
func TestListTools(t *testing.T) {
	mock := &mockPlatform{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	server := NewServer(mock, logger)

	tools := server.ListTools()

	expectedTools := []string{
		"get_pr_info",
		"get_pr_diff",
		"get_file_content",
		"post_review_comment",
		"list_files",
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Expected tool %s not found", expected)
		}
	}
}

// TestCallToolGetPRInfo verifies get_pr_info tool execution
func TestCallToolGetPRInfo(t *testing.T) {
	mock := &mockPlatform{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	server := NewServer(mock, logger)
	ctx := context.Background()

	args := map[string]any{
		"pr_id": float64(123),
	}

	result, err := server.CallTool(ctx, "get_pr_info", args)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result["number"].(int) != 123 {
		t.Errorf("Expected PR number 123, got %v", result["number"])
	}

	if result["title"].(string) != "Test PR" {
		t.Errorf("Expected title 'Test PR', got %v", result["title"])
	}
}

// TestCallToolGetPRDiff verifies get_pr_diff tool execution
func TestCallToolGetPRDiff(t *testing.T) {
	mock := &mockPlatform{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	server := NewServer(mock, logger)
	ctx := context.Background()

	args := map[string]any{
		"pr_id": float64(123),
	}

	result, err := server.CallTool(ctx, "get_pr_diff", args)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result["pr_id"].(int) != 123 {
		t.Errorf("Expected PR ID 123, got %v", result["pr_id"])
	}

	diff, ok := result["diff"].(string)
	if !ok || diff == "" {
		t.Error("Expected non-empty diff")
	}
}

// TestCallToolGetFileContent verifies get_file_content tool execution
func TestCallToolGetFileContent(t *testing.T) {
	mock := &mockPlatform{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	server := NewServer(mock, logger)
	ctx := context.Background()

	args := map[string]any{
		"path": "main.go",
		"ref":  "main",
	}

	result, err := server.CallTool(ctx, "get_file_content", args)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result["path"].(string) != "main.go" {
		t.Errorf("Expected path 'main.go', got %v", result["path"])
	}

	content, ok := result["content"].(string)
	if !ok || content == "" {
		t.Error("Expected non-empty content")
	}
}

// TestCallToolPostReviewComment verifies post_review_comment tool execution
func TestCallToolPostReviewComment(t *testing.T) {
	mock := &mockPlatform{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	server := NewServer(mock, logger)
	ctx := context.Background()

	args := map[string]any{
		"pr_id": float64(123),
		"body":  "Test comment",
	}

	result, err := server.CallTool(ctx, "post_review_comment", args)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result["success"].(bool) != true {
		t.Error("Expected success to be true")
	}
}

// TestCallToolNotFound verifies error handling for unknown tools
func TestCallToolNotFound(t *testing.T) {
	mock := &mockPlatform{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	server := NewServer(mock, logger)
	ctx := context.Background()

	_, err := server.CallTool(ctx, "unknown_tool", nil)
	if err == nil {
		t.Error("Expected error for unknown tool")
	}
}

// TestHandleRequestInitialize verifies initialize request handling
func TestHandleRequestInitialize(t *testing.T) {
	mock := &mockPlatform{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	server := NewServer(mock, logger)
	ctx := context.Background()

	params := `{
		"protocolVersion": "2024-11-05",
		"capabilities": {},
		"clientInfo": {"name": "test-client", "version": "1.0.0"}
	}`

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      "test-id",
		Method:  "initialize",
		Params:  json.RawMessage(params),
	}

	resp := server.HandleRequest(ctx, req)

	if resp.Error != nil {
		t.Fatalf("HandleRequest failed: %v", resp.Error)
	}

	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if result.ProtocolVersion != "2024-11-05" {
		t.Errorf("Expected protocol version 2024-11-05, got %s", result.ProtocolVersion)
	}

	if result.ServerInfo["name"] != "cicd-toolkit" {
		t.Errorf("Expected server name 'cicd-toolkit', got %s", result.ServerInfo["name"])
	}
}

// TestHandleRequestToolsList verifies tools/list request handling
func TestHandleRequestToolsList(t *testing.T) {
	mock := &mockPlatform{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	server := NewServer(mock, logger)
	ctx := context.Background()

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      "test-id",
		Method:  "tools/list",
	}

	resp := server.HandleRequest(ctx, req)

	if resp.Error != nil {
		t.Fatalf("HandleRequest failed: %v", resp.Error)
	}

	var result map[string]any
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	tools, ok := result["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Error("Expected non-empty tools list")
	}
}

// TestHandleRequestToolsCall verifies tools/call request handling
func TestHandleRequestToolsCall(t *testing.T) {
	mock := &mockPlatform{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	server := NewServer(mock, logger)
	ctx := context.Background()

	params := `{
		"name": "get_pr_info",
		"arguments": {"pr_id": 123}
	}`

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      "test-id",
		Method:  "tools/call",
		Params:  json.RawMessage(params),
	}

	resp := server.HandleRequest(ctx, req)

	if resp.Error != nil {
		t.Fatalf("HandleRequest failed: %v", resp.Error)
	}

	var result map[string]any
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// JSON numbers are unmarshaled as float64
	if int(result["number"].(float64)) != 123 {
		t.Errorf("Expected PR number 123, got %v", result["number"])
	}
}

// TestHandleRequestMethodNotFound verifies error handling for unknown methods
func TestHandleRequestMethodNotFound(t *testing.T) {
	mock := &mockPlatform{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	server := NewServer(mock, logger)
	ctx := context.Background()

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      "test-id",
		Method:  "unknown/method",
	}

	resp := server.HandleRequest(ctx, req)

	if resp.Error == nil {
		t.Fatal("Expected error for unknown method")
	}

	if resp.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", resp.Error.Code)
	}
}

// TestToolInputSchemas verifies tool input schemas are valid
func TestToolInputSchemas(t *testing.T) {
	mock := &mockPlatform{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	server := NewServer(mock, logger)

	tools := server.ListTools()

	for _, tool := range tools {
		schema := tool.InputSchema

		// Check required schema fields
		if schema["type"] != "object" {
			t.Errorf("Tool %s: schema type should be 'object', got %v", tool.Name, schema["type"])
		}

		// Verify properties exist
		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			continue
		}

		required, _ := schema["required"].([]string)
		for _, req := range required {
			if _, exists := properties[req]; !exists {
				t.Errorf("Tool %s: required property '%s' not found in properties", tool.Name, req)
			}
		}
	}
}
