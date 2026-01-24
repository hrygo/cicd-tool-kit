// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

// Package claude provides Claude AI integration.
package claude

import (
	"context"
)

// Client is the Claude API client.
// This will be fully implemented in SPEC-CORE-01.
type Client struct {
	// TODO: Add API client fields
	APIKey string
}

// NewClient creates a new Claude client.
func NewClient(apiKey string) *Client {
	return &Client{
		APIKey: apiKey,
	}
}

// Send sends a message to Claude.
func (c *Client) Send(ctx context.Context, req *Request) (*Response, error) {
	// TODO: Implement per SPEC-CORE-01
	return nil, nil
}

// Request represents a Claude API request.
type Request struct {
	Prompt    string
	MaxTokens int
	System    string
}

// Response represents a Claude API response.
type Response struct {
	Content  string
	Tokens   int
	Finished bool
}
