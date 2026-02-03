// Package mcp provides MCP (Model Context Protocol) server implementation
// This exposes platform API tools to Claude Code as MCP tools
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/platform"
)

// Server implements an MCP server over stdio or HTTP
type Server struct {
	platform platform.Platform
	tools    []Tool
	mu       sync.RWMutex
	logger   *slog.Logger
}

// Tool represents an MCP tool
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Handler     ToolHandler    `json:"-"`
}

// ToolHandler handles tool execution
type ToolHandler func(ctx context.Context, args map[string]any) (map[string]any, error)

// NewServer creates a new MCP server
func NewServer(p platform.Platform, logger *slog.Logger) *Server {
	s := &Server{
		platform: p,
		logger:   logger,
	}
	s.registerDefaultTools()
	return s
}

// registerDefaultTools registers platform-specific tools
func (s *Server) registerDefaultTools() {
	s.tools = []Tool{
		{
			Name:        "get_pr_info",
			Description: "Get pull/merge request information including title, author, branches, and metadata",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pr_id": map[string]any{
						"type":        "integer",
						"description": "Pull/Merge request number",
					},
				},
				"required": []string{"pr_id"},
			},
			Handler: s.handleGetPRInfo,
		},
		{
			Name:        "get_pr_diff",
			Description: "Get the diff for a pull/merge request. Returns unified diff format.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pr_id": map[string]any{
						"type":        "integer",
						"description": "Pull/Merge request number",
					},
				},
				"required": []string{"pr_id"},
			},
			Handler: s.handleGetPRDiff,
		},
		{
			Name:        "get_file_content",
			Description: "Get file content at a specific revision/commit",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "File path in the repository",
					},
					"ref": map[string]any{
						"type":        "string",
						"description": "Git ref (branch, tag, or commit SHA)",
					},
				},
				"required": []string{"path", "ref"},
			},
			Handler: s.handleGetFileContent,
		},
		{
			Name:        "post_review_comment",
			Description: "Post a review comment to a pull/merge request. Use this to deliver code review results.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pr_id": map[string]any{
						"type":        "integer",
						"description": "Pull/Merge request number",
					},
					"body": map[string]any{
						"type":        "string",
						"description": "Comment body in markdown format",
					},
					"as_review": map[string]any{
						"type":        "boolean",
						"description": "Post as a review comment (default: false)",
					},
				},
				"required": []string{"pr_id", "body"},
			},
			Handler: s.handlePostReviewComment,
		},
		{
			Name:        "list_files",
			Description: "List files in the repository at a specific path and ref",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Directory path (use '.' for root)",
					},
					"ref": map[string]any{
						"type":        "string",
						"description": "Git ref (branch, tag, or commit SHA)",
					},
				},
				"required": []string{"path", "ref"},
			},
			Handler: s.handleListFiles,
		},
	}
}

// RegisterTool registers a custom tool
func (s *Server) RegisterTool(tool Tool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools = append(s.tools, tool)
}

// ListTools returns all available tools (for MCP tools/list response)
func (s *Server) ListTools() []Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tools
}

// CallTool executes a tool by name
func (s *Server) CallTool(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, tool := range s.tools {
		if tool.Name == name {
			s.logger.Info("calling tool", "tool", name, "args", args)
			result, err := tool.Handler(ctx, args)
			if err != nil {
				s.logger.Error("tool error", "tool", name, "error", err)
				return nil, err
			}
			return result, nil
		}
	}

	return nil, fmt.Errorf("tool not found: %s", name)
}

// Tool handlers

func (s *Server) handleGetPRInfo(ctx context.Context, args map[string]any) (map[string]any, error) {
	prID, ok := args["pr_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid pr_id: must be integer")
	}

	info, err := s.platform.GetPRInfo(ctx, int(prID))
	if err != nil {
		return nil, fmt.Errorf("failed to get PR info: %w", err)
	}

	return map[string]any{
		"number":      info.Number,
		"title":       info.Title,
		"description": info.Description,
		"author":      info.Author,
		"sha":         info.SHA,
		"base_branch": info.BaseBranch,
		"head_branch": info.HeadBranch,
		"source_repo": info.SourceRepo,
		"labels":      info.Labels,
		"platform":    s.platform.Name(),
	}, nil
}

func (s *Server) handleGetPRDiff(ctx context.Context, args map[string]any) (map[string]any, error) {
	prID, ok := args["pr_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid pr_id: must be integer")
	}

	diff, err := s.platform.GetDiff(ctx, int(prID))
	if err != nil {
		return nil, fmt.Errorf("failed to get PR diff: %w", err)
	}

	return map[string]any{
		"pr_id":  int(prID),
		"diff":   diff,
		"length": len(diff),
	}, nil
}

func (s *Server) handleGetFileContent(ctx context.Context, args map[string]any) (map[string]any, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid path: must be string")
	}
	ref, ok := args["ref"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid ref: must be string")
	}

	content, err := s.platform.GetFile(ctx, path, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %w", err)
	}

	return map[string]any{
		"path":    path,
		"ref":     ref,
		"content": content,
		"length":  len(content),
	}, nil
}

func (s *Server) handlePostReviewComment(ctx context.Context, args map[string]any) (map[string]any, error) {
	prID, ok := args["pr_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid pr_id: must be integer")
	}
	body, ok := args["body"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid body: must be string")
	}
	asReview, _ := args["as_review"].(bool)

	opts := platform.CommentOptions{
		PRID:     int(prID),
		Body:     body,
		AsReview: asReview,
	}

	err := s.platform.PostComment(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to post comment: %w", err)
	}

	return map[string]any{
		"success": true,
		"pr_id":   int(prID),
		"message": "Comment posted successfully",
	}, nil
}

func (s *Server) handleListFiles(ctx context.Context, args map[string]any) (map[string]any, error) {
	// This is a placeholder - actual implementation would require platform-specific APIs
	// For now, return a note about using Claude Code's native ls tool
	return map[string]any{
		"message": "Use Claude Code's native ls tool for local file listing",
	}, nil
}

// MCP Protocol types for JSON-RPC

// MCPRequest represents an incoming JSON-RPC request
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse represents a JSON-RPC response
type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

// MCPError represents a JSON-RPC error
type MCPError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// InitializeParams represents the initialize request params
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]any         `json:"capabilities"`
	ClientInfo      map[string]string      `json:"clientInfo"`
	Meta            map[string]interface{} `json:"meta,omitempty"`
}

// InitializeResult represents the initialize response
type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]any         `json:"capabilities"`
	ServerInfo      map[string]string      `json:"serverInfo"`
	Meta            map[string]interface{} `json:"meta,omitempty"`
}

// HandleRequest handles an incoming MCP JSON-RPC request
func (s *Server) HandleRequest(ctx context.Context, req MCPRequest) MCPResponse {
	var result any
	var err error

	switch req.Method {
	case "initialize":
		var params InitializeParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return s.errorResponse(req.ID, -32700, "Parse error", nil)
		}
		result = s.handleInitialize(params)

	case "tools/list":
		result = s.handleListTools()

	case "tools/call":
		result, err = s.handleToolsCall(ctx, req.Params)
		if err != nil {
			return s.errorResponse(req.ID, -32603, "Internal error", []byte(err.Error()))
		}

	default:
		return s.errorResponse(req.ID, -32601, "Method not found", nil)
	}

	if result != nil {
		data, _ := json.Marshal(result)
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(data),
		}
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}
}

func (s *Server) handleInitialize(params InitializeParams) InitializeResult {
	s.logger.Info("MCP server initialized", "client", params.ClientInfo["name"])

	return InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: map[string]any{
			"tools": map[string]any{},
		},
		ServerInfo: map[string]string{
			"name":    "cicd-toolkit",
			"version": "1.0.0",
		},
	}
}

func (s *Server) handleListTools() map[string]any {
	tools := make([]map[string]any, len(s.tools))
	for i, tool := range s.tools {
		tools[i] = map[string]any{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		}
	}
	return map[string]any{
		"tools": tools,
	}
}

func (s *Server) handleToolsCall(ctx context.Context, params json.RawMessage) (map[string]any, error) {
	var p struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments,omitempty"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return s.CallTool(ctx, p.Name, p.Arguments)
}

func (s *Server) errorResponse(id any, code int, message string, data json.RawMessage) MCPResponse {
	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// ServeHTTP handles HTTP requests (for SSE/HTTP transport)
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, nil, -32700, "Parse error", nil)
		return
	}

	resp := s.HandleRequest(r.Context(), req)
	json.NewEncoder(w).Encode(resp)
}

// ServeStdio handles stdio transport (for direct Claude Code integration)
func (s *Server) ServeStdio(ctx context.Context, in io.Reader, out io.Writer) error {
	decoder := json.NewDecoder(in)
	encoder := json.NewEncoder(out)

	for {
		var req MCPRequest
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("decode error: %w", err)
		}

		resp := s.HandleRequest(ctx, req)
		if err := encoder.Encode(resp); err != nil {
			return fmt.Errorf("encode error: %w", err)
		}
	}
}

func (s *Server) writeError(w http.ResponseWriter, id any, code int, message string, data json.RawMessage) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	json.NewEncoder(w).Encode(resp)
}
