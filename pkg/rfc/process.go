// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package rfc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Process manages the RFC (Request for Comments) workflow.
// Implements SPEC-RFC-01: RFC Process
type Process struct {
	mu        sync.RWMutex
	rootDir   string
	rfcs      map[string]*RFC
	stateDir  string
}

// RFC represents a Request for Comments.
type RFC struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Author      string       `json:"author"`
	Status      RFCStatus    `json:"status"`
	Type        RFCType      `json:"type"`
	Priority    Priority     `json:"priority"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	DiscussedAt time.Time    `json:"discussed_at,omitempty"`
	AcceptedAt  time.Time    `json:"accepted_at,omitempty"`
	RejectedAt  time.Time    `json:"rejected_at,omitempty"`
	FilePath    string       `json:"file_path"`
	Comments    []*Comment   `json:"comments,omitempty"`
	Votes       []*Vote      `json:"votes,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
}

// RFCStatus represents the status of an RFC.
type RFCStatus string

const (
	StatusDraft      RFCStatus = "draft"
	StatusProposed   RFCStatus = "proposed"
	StatusDiscussed  RFCStatus = "discussed"
	StatusAccepted   RFCStatus = "accepted"
	StatusRejected   RFCStatus = "rejected"
	StatusSuperseded RFCStatus = "superseded"
	StatusWithdrawn  RFCStatus = "withdrawn"
)

// RFCType represents the type of RFC.
type RFCType string

const (
	TypeFeature    RFCType = "feature"
	TypeRefactor   RFCType = "refactor"
	TypeProcess    RFCType = "process"
	TypePolicy     RFCType = "policy"
	TypeDeprecation RFCType = "deprecation"
)

// Priority represents the priority level.
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityCritical Priority = "critical"
)

// Comment represents a comment on an RFC.
type Comment struct {
	ID        string    `json:"id"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Vote represents a vote on an RFC.
type Vote struct {
	Author    string       `json:"author"`
	Decision  VoteDecision `json:"decision"`
	Reason    string       `json:"reason,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

// VoteDecision represents a vote decision.
type VoteDecision string

const (
	VoteApprove VoteDecision = "approve"
	VoteReject  VoteDecision = "reject"
	VoteAbstain VoteDecision = "abstain"
)

// NewProcess creates a new RFC process manager.
func NewProcess(rootDir string) (*Process, error) {
	if rootDir == "" {
		rootDir = "./rfcs"
	}

	stateDir := filepath.Join(rootDir, ".state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	p := &Process{
		rootDir:  rootDir,
		stateDir: stateDir,
		rfcs:     make(map[string]*RFC),
	}

	// Load existing RFCs
	if err := p.Load(); err != nil {
		return nil, err
	}

	return p, nil
}

// Create creates a new RFC.
func (p *Process) Create(title, description, author string, rfcType RFCType) (*RFC, error) {
	id := generateRFCID()

	rfc := &RFC{
		ID:          id,
		Title:       title,
		Description: description,
		Author:      author,
		Status:      StatusDraft,
		Type:        rfcType,
		Priority:    PriorityMedium,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Comments:    make([]*Comment, 0),
		Votes:       make([]*Vote, 0),
		Tags:        make([]string, 0),
	}

	p.mu.Lock()
	p.rfcs[id] = rfc
	p.mu.Unlock()

	// Write RFC file
	if err := p.Write(rfc); err != nil {
		return nil, err
	}

	return rfc, nil
}

// Get retrieves an RFC by ID.
func (p *Process) Get(id string) (*RFC, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	rfc, ok := p.rfcs[id]
	return rfc, ok
}

// List lists all RFCs, optionally filtered.
func (p *Process) List(filter func(*RFC) bool) []*RFC {
	p.mu.RLock()
	defer p.mu.RUnlock()

	results := make([]*RFC, 0)
	for _, rfc := range p.rfcs {
		if filter == nil || filter(rfc) {
			results = append(results, rfc)
		}
	}
	return results
}

// Update updates an RFC.
func (p *Process) Update(rfc *RFC) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	rfc.UpdatedAt = time.Now()
	p.rfcs[rfc.ID] = rfc

	return p.Write(rfc)
}

// Propose moves an RFC from draft to proposed.
func (p *Process) Propose(id string) error {
	rfc, ok := p.Get(id)
	if !ok {
		return fmt.Errorf("RFC not found: %s", id)
	}

	if rfc.Status != StatusDraft {
		return fmt.Errorf("RFC is not in draft status: %s", rfc.Status)
	}

	rfc.Status = StatusProposed
	return p.Update(rfc)
}

// Discuss marks an RFC as discussed.
func (p *Process) Discuss(id string) error {
	rfc, ok := p.Get(id)
	if !ok {
		return fmt.Errorf("RFC not found: %s", id)
	}

	if rfc.Status != StatusProposed {
		return fmt.Errorf("RFC is not in proposed status: %s", rfc.Status)
	}

	rfc.Status = StatusDiscussed
	rfc.DiscussedAt = time.Now()
	return p.Update(rfc)
}

// Accept accepts an RFC.
func (p *Process) Accept(id string) error {
	rfc, ok := p.Get(id)
	if !ok {
		return fmt.Errorf("RFC not found: %s", id)
	}

	if rfc.Status != StatusDiscussed && rfc.Status != StatusProposed {
		return fmt.Errorf("RFC cannot be accepted from status: %s", rfc.Status)
	}

	rfc.Status = StatusAccepted
	rfc.AcceptedAt = time.Now()
	return p.Update(rfc)
}

// Reject rejects an RFC.
func (p *Process) Reject(id string) error {
	rfc, ok := p.Get(id)
	if !ok {
		return fmt.Errorf("RFC not found: %s", id)
	}

	rfc.Status = StatusRejected
	rfc.RejectedAt = time.Now()
	return p.Update(rfc)
}

// Withdraw withdraws an RFC.
func (p *Process) Withdraw(id string) error {
	rfc, ok := p.Get(id)
	if !ok {
		return fmt.Errorf("RFC not found: %s", id)
	}

	rfc.Status = StatusWithdrawn
	return p.Update(rfc)
}

// AddComment adds a comment to an RFC.
func (p *Process) AddComment(rfcID, author, content string) error {
	rfc, ok := p.Get(rfcID)
	if !ok {
		return fmt.Errorf("RFC not found: %s", rfcID)
	}

	comment := &Comment{
		ID:        generateCommentID(),
		Author:    author,
		Content:   content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	rfc.Comments = append(rfc.Comments, comment)
	return p.Update(rfc)
}

// AddVote adds a vote to an RFC.
func (p *Process) AddVote(rfcID, author string, decision VoteDecision) error {
	rfc, ok := p.Get(rfcID)
	if !ok {
		return fmt.Errorf("RFC not found: %s", rfcID)
	}

	// Remove existing vote from author
	var filtered []*Vote
	for _, v := range rfc.Votes {
		if v.Author != author {
			filtered = append(filtered, v)
		}
	}
	rfc.Votes = filtered

	// Add new vote
	rfc.Votes = append(rfc.Votes, &Vote{
		Author:    author,
		Decision:  decision,
		CreatedAt: time.Now(),
	})

	return p.Update(rfc)
}

// Write writes an RFC to its file.
func (p *Process) Write(rfc *RFC) error {
	if rfc.FilePath == "" {
		rfc.FilePath = filepath.Join(p.rootDir, rfc.ID+".md")
	}

	var content strings.Builder

	content.WriteString("---\n")
	content.WriteString(fmt.Sprintf("id: %s\n", rfc.ID))
	content.WriteString(fmt.Sprintf("title: %s\n", rfc.Title))
	content.WriteString(fmt.Sprintf("author: %s\n", rfc.Author))
	content.WriteString(fmt.Sprintf("status: %s\n", rfc.Status))
	content.WriteString(fmt.Sprintf("type: %s\n", rfc.Type))
	content.WriteString(fmt.Sprintf("priority: %s\n", rfc.Priority))
	content.WriteString(fmt.Sprintf("created_at: %s\n", rfc.CreatedAt.Format(time.RFC3339)))
	if !rfc.DiscussedAt.IsZero() {
		content.WriteString(fmt.Sprintf("discussed_at: %s\n", rfc.DiscussedAt.Format(time.RFC3339)))
	}
	if !rfc.AcceptedAt.IsZero() {
		content.WriteString(fmt.Sprintf("accepted_at: %s\n", rfc.AcceptedAt.Format(time.RFC3339)))
	}
	if len(rfc.Tags) > 0 {
		content.WriteString(fmt.Sprintf("tags: [%s]\n", strings.Join(rfc.Tags, ", ")))
	}
	content.WriteString("---\n\n")

	content.WriteString("# ")
	content.WriteString(rfc.Title)
	content.WriteString("\n\n")

	content.WriteString(rfc.Description)
	content.WriteString("\n\n")

	// Add comments section
	if len(rfc.Comments) > 0 {
		content.WriteString("## Discussion\n\n")
		for _, c := range rfc.Comments {
			content.WriteString(fmt.Sprintf("### %s (%s)\n\n", c.Author, c.CreatedAt.Format("2006-01-02")))
			content.WriteString(c.Content)
			content.WriteString("\n\n")
		}
	}

	// Add voting section
	if len(rfc.Votes) > 0 {
		content.WriteString("## Voting\n\n")
		approvals := 0
		rejections := 0
		for _, v := range rfc.Votes {
			if v.Decision == VoteApprove {
				approvals++
			} else if v.Decision == VoteReject {
				rejections++
			}
			content.WriteString(fmt.Sprintf("- %s: %s\n", v.Author, v.Decision))
		}
		content.WriteString(fmt.Sprintf("\n**Result:** %d approve, %d reject\n", approvals, rejections))
	}

	return os.WriteFile(rfc.FilePath, []byte(content.String()), 0644)
}

// Load loads all RFCs from the directory.
func (p *Process) Load() error {
	entries, err := os.ReadDir(p.rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(p.rootDir, entry.Name())
		rfc, err := p.parseRFC(path)
		if err != nil {
			continue
		}

		p.rfcs[rfc.ID] = rfc
	}

	return nil
}

// parseRFC parses an RFC from a file.
func (p *Process) parseRFC(path string) (*RFC, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Simple parser - in production, use proper frontmatter parser
	lines := strings.Split(string(content), "\n")

	rfc := &RFC{
		FilePath:  path,
		Comments:  make([]*Comment, 0),
		Votes:     make([]*Vote, 0),
		Tags:      make([]string, 0),
	}

	inFrontmatter := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "---" {
			inFrontmatter = !inFrontmatter
			continue
		}

		if inFrontmatter {
			if strings.HasPrefix(line, "id:") {
				rfc.ID = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
			} else if strings.HasPrefix(line, "title:") {
				rfc.Title = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
			} else if strings.HasPrefix(line, "author:") {
				rfc.Author = strings.TrimSpace(strings.TrimPrefix(line, "author:"))
			} else if strings.HasPrefix(line, "status:") {
				rfc.Status = RFCStatus(strings.TrimSpace(strings.TrimPrefix(line, "status:")))
			} else if strings.HasPrefix(line, "type:") {
				rfc.Type = RFCType(strings.TrimSpace(strings.TrimPrefix(line, "type:")))
			}
		}
	}

	return rfc, nil
}

// GetByStatus returns RFCs with the given status.
func (p *Process) GetByStatus(status RFCStatus) []*RFC {
	return p.List(func(rfc *RFC) bool {
		return rfc.Status == status
	})
}

// GetByType returns RFCs of the given type.
func (p *Process) GetByType(rfcType RFCType) []*RFC {
	return p.List(func(rfc *RFC) bool {
		return rfc.Type == rfcType
	})
}

// GetByAuthor returns RFCs by the given author.
func (p *Process) GetByAuthor(author string) []*RFC {
	return p.List(func(rfc *RFC) bool {
		return rfc.Author == author
	})
}

// generateRFCID generates a unique RFC ID.
func generateRFCID() string {
	return fmt.Sprintf("RFC-%04d", time.Now().Unix()%10000)
}

// generateCommentID generates a unique comment ID.
func generateCommentID() string {
	return fmt.Sprintf("cmt-%d", time.Now().UnixNano())
}

// Template represents an RFC template.
type Template struct {
	Name        string
	Description string
	Type        RFCType
	Content     string
}

// GetTemplates returns available RFC templates.
func GetTemplates() []*Template {
	return []*Template{
		{
			Name:        "Feature Request",
			Description: "Template for proposing new features",
			Type:        TypeFeature,
			Content: `# Feature Title

## Summary
Brief description of the feature.

## Motivation
Why is this feature needed?

## Proposal
Detailed description of the proposed feature.

## Alternatives
What alternative approaches were considered?
`,
		},
		{
			Name:        "Process Change",
			Description: "Template for proposing process changes",
			Type:        TypeProcess,
			Content: `# Process Change Title

## Current State
Description of the current process.

## Problem
What problem does the current process have?

## Proposed Change
Description of the proposed change.

## Impact
What impact will this change have?
`,
		},
	}
}
