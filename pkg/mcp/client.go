// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client implements an MCP (Model Context Protocol) client.
// Implements SPEC-MCP-01: MCP Client
type Client struct {
	httpClient *http.Client
	baseURL    string
	headers    map[string]string
	mu         sync.RWMutex
}

// NewClient creates a new MCP client.
func NewClient(baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
		headers: make(map[string]string),
	}
}

// SetTimeout sets the HTTP timeout.
func (c *Client) SetTimeout(timeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.httpClient.Timeout = timeout
}

// SetHeader sets a default header.
func (c *Client) SetHeader(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.headers[key] = value
}

// Call invokes an MCP tool.
func (c *Client) Call(ctx context.Context, tool string, input map[string]any) (map[string]any, error) {
	request := &ToolRequest{
		Tool:  tool,
		Input: input,
	}
	
	response, err := c.doRequest(ctx, "/tools/call", request)
	if err != nil {
		return nil, err
	}
	
	return response.Result, nil
}

// ListTools lists available MCP tools.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	response, err := c.doRequest(ctx, "/tools/list", nil)
	if err != nil {
		return nil, err
	}
	
	var tools []Tool
	if data, err := json.Marshal(response.Result); err == nil {
		json.Unmarshal(data, &tools)
	}
	
	return tools, nil
}

// GetResource gets an MCP resource.
func (c *Client) GetResource(ctx context.Context, uri string) (map[string]any, error) {
	request := map[string]any{
		"uri": uri,
	}
	
	response, err := c.doRequest(ctx, "/resources/get", request)
	if err != nil {
		return nil, err
	}
	
	return response.Result, nil
}

// ListResources lists available resources.
func (c *Client) ListResources(ctx context.Context) ([]Resource, error) {
	response, err := c.doRequest(ctx, "/resources/list", nil)
	if err != nil {
		return nil, err
	}
	
	var resources []Resource
	if data, err := json.Marshal(response.Result); err == nil {
		json.Unmarshal(data, &resources)
	}
	
	return resources, nil
}

// doRequest makes an HTTP request to the MCP server.
func (c *Client) doRequest(ctx context.Context, path string, body any) (*ToolResponse, error) {
	c.mu.RLock()
	baseURL := c.baseURL
	c.mu.RUnlock()
	
	url := baseURL + path
	
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bodyReader)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	c.mu.RLock()
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	c.mu.RUnlock()
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MCP request failed: %s", resp.Status)
	}
	
	var response ToolResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	
	return &response, nil
}

// ToolRequest represents an MCP tool invocation.
type ToolRequest struct {
	Tool  string                 `json:"tool"`
	Input map[string]any         `json:"input"`
	Meta  map[string]string      `json:"meta,omitempty"`
}

// ToolResponse represents an MCP tool response.
type ToolResponse struct {
	Result    map[string]any `json:"result"`
	Error     *string        `json:"error,omitempty"`
	Meta      map[string]any `json:"meta,omitempty"`
}

// Tool represents an available MCP tool.
type Tool struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	InputSchema json.RawMessage  `json:"inputSchema"`
}

// Resource represents an MCP resource.
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MIMEType    string `json:"mimeType"`
}
