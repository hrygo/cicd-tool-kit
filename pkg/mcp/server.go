// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Server provides an MCP (Model Context Protocol) server.
// Implements SPEC-MCP-02: MCP Server
type Server struct {
	address     string
	resources   map[string]*ResourceHandler
	prompts     map[string]*PromptHandler
	tools       map[string]*ToolHandler
	mu          sync.RWMutex
	capabilities *Capabilities
	serverInfo  *ServerInfo
	httpServer  *http.Server
}

// ResourceHandler handles resource requests.
type ResourceHandler struct {
	Resource    *Resource
	ReadFunc    func(ctx context.Context, uri string) (*ResourceContent, error)
	ListFunc    func(ctx context.Context) ([]*Resource, error)
}

// PromptHandler handles prompt requests.
type PromptHandler struct {
	Prompt    *Prompt
	GetFunc   func(ctx context.Context, name string, args map[string]any) (*PromptMessage, error)
	ListFunc  func(ctx context.Context) ([]*Prompt, error)
}

// ToolHandler handles tool requests.
type ToolHandler struct {
	Tool     *Tool
	CallFunc func(ctx context.Context, name string, args map[string]any) (*ToolResult, error)
	ListFunc func(ctx context.Context) ([]*Tool, error)
}

// NewServer creates a new MCP server.
func NewServer(address string) *Server {
	return &Server{
		address: address,
		resources: make(map[string]*ResourceHandler),
		prompts: make(map[string]*PromptHandler),
		tools: make(map[string]*ToolHandler),
		capabilities: &Capabilities{
			Resources: true,
			Prompts:   true,
			Tools:     true,
		},
		serverInfo: &ServerInfo{
			Name:    "cicd-ai-toolkit",
			Version: "1.0.0",
		},
	}
}

// SetServerInfo sets the server information.
func (s *Server) SetServerInfo(name, version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.serverInfo.Name = name
	s.serverInfo.Version = version
}

// SetCapabilities sets server capabilities.
func (s *Server) SetCapabilities(cap *Capabilities) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.capabilities = cap
}

// RegisterResource registers a resource handler.
func (s *Server) RegisterResource(handler *ResourceHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resources[handler.Resource.URI] = handler
}

// RegisterPrompt registers a prompt handler.
func (s *Server) RegisterPrompt(handler *PromptHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prompts[handler.Prompt.Name] = handler
}

// RegisterTool registers a tool handler.
func (s *Server) RegisterTool(handler *ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[handler.Tool.Name] = handler
}

// Start starts the MCP server.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1", s.handleInfo)
	mux.HandleFunc("/v1/resources/list", s.handleResourceList)
	mux.HandleFunc("/v1/resources/read", s.handleResourceRead)
	mux.HandleFunc("/v1/prompts/list", s.handlePromptList)
	mux.HandleFunc("/v1/prompts/get", s.handlePromptGet)
	mux.HandleFunc("/v1/tools/list", s.handleToolList)
	mux.HandleFunc("/v1/tools/call", s.handleToolCall)

	s.httpServer = &http.Server{
		Addr:    s.address,
		Handler: mux,
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- s.httpServer.ListenAndServe()
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return s.httpServer.Shutdown(context.Background())
	}
}

// Stop stops the MCP server.
func (s *Server) Stop() error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(context.Background())
	}
	return nil
}

// handleInfo handles server info requests.
func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.writeResponse(w, map[string]any{
		"name":         s.serverInfo.Name,
		"version":      s.serverInfo.Version,
		"capabilities": s.capabilities,
	})
}

// handleResourceList handles resource list requests.
func (s *Server) handleResourceList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	resources := make([]*Resource, 0, len(s.resources))
	for _, handler := range s.resources {
		resources = append(resources, handler.Resource)
	}

	s.writeResponse(w, map[string]any{
		"resources": resources,
	})
}

// handleResourceRead handles resource read requests.
func (s *Server) handleResourceRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URI string `json:"uri"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, -32700, "parse error")
		return
	}

	s.mu.RLock()
	handler, ok := s.resources[req.URI]
	s.mu.RUnlock()

	if !ok {
		s.writeError(w, -32601, "resource not found")
		return
	}

	if handler.ReadFunc != nil {
		content, err := handler.ReadFunc(r.Context(), req.URI)
		if err != nil {
			s.writeError(w, -32603, fmt.Sprintf("read failed: %v", err))
			return
		}
		s.writeResponse(w, content)
		return
	}

	s.writeResponse(w, &ResourceContent{
		URI:  req.URI,
		Text: "",
	})
}

// handlePromptList handles prompt list requests.
func (s *Server) handlePromptList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	prompts := make([]*Prompt, 0, len(s.prompts))
	for _, handler := range s.prompts {
		prompts = append(prompts, handler.Prompt)
	}

	s.writeResponse(w, map[string]any{
		"prompts": prompts,
	})
}

// handlePromptGet handles prompt get requests.
func (s *Server) handlePromptGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, -32700, "parse error")
		return
	}

	s.mu.RLock()
	handler, ok := s.prompts[req.Name]
	s.mu.RUnlock()

	if !ok {
		s.writeError(w, -32601, "prompt not found")
		return
	}

	if handler.GetFunc != nil {
		message, err := handler.GetFunc(r.Context(), req.Name, req.Arguments)
		if err != nil {
			s.writeError(w, -32603, fmt.Sprintf("get failed: %v", err))
			return
		}
		s.writeResponse(w, message)
		return
	}

	s.writeResponse(w, &PromptMessage{
		Role:    "user",
		Content: "",
	})
}

// handleToolList handles tool list requests.
func (s *Server) handleToolList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]*Tool, 0, len(s.tools))
	for _, handler := range s.tools {
		tools = append(tools, handler.Tool)
	}

	s.writeResponse(w, map[string]any{
		"tools": tools,
	})
}

// handleToolCall handles tool call requests.
func (s *Server) handleToolCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, -32700, "parse error")
		return
	}

	s.mu.RLock()
	handler, ok := s.tools[req.Name]
	s.mu.RUnlock()

	if !ok {
		s.writeError(w, -32601, "tool not found")
		return
	}

	if handler.CallFunc != nil {
		result, err := handler.CallFunc(r.Context(), req.Name, req.Arguments)
		if err != nil {
			s.writeError(w, -32603, fmt.Sprintf("call failed: %v", err))
			return
		}
		s.writeResponse(w, result)
		return
	}

	s.writeResponse(w, &ToolResult{
		Content: []any{},
	})
}

// writeResponse writes a JSON-RPC response.
func (s *Server) writeResponse(w http.ResponseWriter, result any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"result":  result,
	})
}

// writeError writes a JSON-RPC error response.
func (s *Server) writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

// ResourceBuilder helps build resource handlers.
type ResourceBuilder struct {
	uri     string
	name    string
	readFn  func(ctx context.Context, uri string) (*ResourceContent, error)
}

// NewResourceBuilder creates a new resource builder.
func NewResourceBuilder(uri, name string) *ResourceBuilder {
	return &ResourceBuilder{
		uri:  uri,
		name: name,
	}
}

// WithRead sets the read function.
func (b *ResourceBuilder) WithRead(fn func(ctx context.Context, uri string) (*ResourceContent, error)) *ResourceBuilder {
	b.readFn = fn
	return b
}

// Build creates the resource handler.
func (b *ResourceBuilder) Build() *ResourceHandler {
	return &ResourceHandler{
		Resource: &Resource{
			URI:  b.uri,
			Name: b.name,
		},
		ReadFunc: b.readFn,
	}
}

// ToolBuilder helps build tool handlers.
type ToolBuilder struct {
	name        string
	description string
	callFn      func(ctx context.Context, name string, args map[string]any) (*ToolResult, error)
}

// NewToolBuilder creates a new tool builder.
func NewToolBuilder(name, description string) *ToolBuilder {
	return &ToolBuilder{
		name:        name,
		description: description,
	}
}

// WithCall sets the call function.
func (b *ToolBuilder) WithCall(fn func(ctx context.Context, name string, args map[string]any) (*ToolResult, error)) *ToolBuilder {
	b.callFn = fn
	return b
}

// Build creates the tool handler.
func (b *ToolBuilder) Build() *ToolHandler {
	return &ToolHandler{
		Tool: &Tool{
			Name:        b.name,
			Description: b.description,
		},
		CallFunc: b.callFn,
	}
}

// Capabilities defines server capabilities.
type Capabilities struct {
	Resources bool `json:"resources"`
	Prompts   bool `json:"prompts"`
	Tools     bool `json:"tools"`
}

// ServerInfo holds server metadata.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ResourceContent represents resource content.
type ResourceContent struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Blob     []byte `json:"blob,omitempty"`
}

// Prompt represents a prompt template.
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Arguments   []PromptArgument `json:"arguments"`
}

// PromptArgument represents a prompt argument.
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// PromptMessage represents a prompt message.
type PromptMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// ToolResult represents a tool execution result.
type ToolResult struct {
	Content []any `json:"content"`
	IsError bool   `json:"isError"`
}
