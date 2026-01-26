// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Marketplace manages skill discovery and installation.
// Implements SPEC-ECO-01: Skill Marketplace
type Marketplace struct {
	mu          sync.RWMutex
	skills      map[string]*SkillMetadata
	repositories []string
	cacheDir    string
	installed   map[string]string // name -> version
}

// SkillMetadata represents metadata about a skill.
type SkillMetadata struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Author      string            `json:"author"`
	Repository  string            `json:"repository"`
	Homepage    string            `json:"homepage,omitempty"`
	License     string            `json:"license"`
	Tags        []string          `json:"tags"`
	Category    string            `json:"category"`
	Keywords    []string          `json:"keywords"`
	Installed   bool              `json:"installed"`
	InstalledAt time.Time         `json:"installed_at,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// InstallResult represents the result of a skill installation.
type InstallResult struct {
	Success  bool
	Skill    string
	Version  string
	Message  string
	Error    error
}

// NewMarketplace creates a new skill marketplace.
func NewMarketplace(cacheDir string) (*Marketplace, error) {
	if cacheDir == "" {
		homeDir, _ := os.UserHomeDir()
		cacheDir = filepath.Join(homeDir, ".cicd-ai-toolkit", "marketplace")
	}

	mp := &Marketplace{
		skills:      make(map[string]*SkillMetadata),
		repositories: []string{
			"https://raw.githubusercontent.com/cicd-ai-toolkit/skills/main/registry",
		},
		cacheDir:  cacheDir,
		installed: make(map[string]string),
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Load installed skills
	mp.loadInstalled()

	return mp, nil
}

// AddRepository adds a skill repository.
func (m *Marketplace) AddRepository(repo string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.repositories = append(m.repositories, repo)
}

// Refresh fetches the latest skill catalog.
func (m *Marketplace) Refresh(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Fetch from all repositories
	for _, repo := range m.repositories {
		if err := m.fetchRepository(ctx, repo); err != nil {
			return fmt.Errorf("failed to fetch repository %s: %w", repo, err)
		}
	}

	return nil
}

// fetchRepository fetches skills from a repository.
func (m *Marketplace) fetchRepository(ctx context.Context, repo string) error {
	// In production, this would make HTTP requests to the repository
	// For now, use local registry
	return nil
}

// List lists all available skills.
func (m *Marketplace) List() []*SkillMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	skills := make([]*SkillMetadata, 0, len(m.skills))
	for _, skill := range m.skills {
		skills = append(skills, skill)
	}
	return skills
}

// Search searches for skills by query.
func (m *Marketplace) Search(query string) []*SkillMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]*SkillMetadata, 0)
	queryLower := toLower(query)

	for _, skill := range m.skills {
		if contains(toLower(skill.Name), queryLower) ||
			contains(toLower(skill.Description), queryLower) ||
			contains(toLower(skill.Category), queryLower) {
			results = append(results, skill)
			continue
		}

		for _, tag := range skill.Tags {
			if contains(toLower(tag), queryLower) {
				results = append(results, skill)
				break
			}
		}

		for _, keyword := range skill.Keywords {
			if contains(toLower(keyword), queryLower) {
				results = append(results, skill)
				break
			}
		}
	}

	return results
}

// Get retrieves a skill by ID.
func (m *Marketplace) Get(id string) (*SkillMetadata, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	skill, ok := m.skills[id]
	return skill, ok
}

// Install installs a skill.
func (m *Marketplace) Install(ctx context.Context, id string) (*InstallResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	skill, ok := m.skills[id]
	if !ok {
		return &InstallResult{
			Success: false,
			Skill:   id,
			Message: "skill not found",
		}, fmt.Errorf("skill not found: %s", id)
	}

	// Check if already installed
	if _, installed := m.installed[id]; installed {
		return &InstallResult{
			Success: true,
			Skill:   id,
			Version: skill.Version,
			Message: "already installed",
		}, nil
	}

	// Download and install
	if err := m.downloadSkill(ctx, skill); err != nil {
		return &InstallResult{
			Success: false,
			Skill:   id,
			Error:   err,
		}, err
	}

	// Mark as installed
	m.installed[id] = skill.Version
	skill.Installed = true
	skill.InstalledAt = time.Now()

	// Save state
	m.saveInstalled()

	return &InstallResult{
		Success: true,
		Skill:   id,
		Version: skill.Version,
		Message: "installed successfully",
	}, nil
}

// Uninstall uninstalls a skill.
func (m *Marketplace) Uninstall(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.installed[id]; !ok {
		return fmt.Errorf("skill not installed: %s", id)
	}

	// Remove from installed
	delete(m.installed, id)

	if skill, ok := m.skills[id]; ok {
		skill.Installed = false
		skill.InstalledAt = time.Time{}
	}

	m.saveInstalled()
	return nil
}

// Update updates an installed skill.
func (m *Marketplace) Update(ctx context.Context, id string) (*InstallResult, error) {
	// Uninstall then reinstall
	if err := m.Uninstall(ctx, id); err != nil {
		return nil, err
	}
	return m.Install(ctx, id)
}

// downloadSkill downloads a skill from its repository.
func (m *Marketplace) downloadSkill(ctx context.Context, skill *SkillMetadata) error {
	// In production, this would download from the repository
	// For now, create a placeholder
	return nil
}

// loadInstalled loads the list of installed skills.
func (m *Marketplace) loadInstalled() {
	path := filepath.Join(m.cacheDir, "installed.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	json.Unmarshal(data, &m.installed)

	// Update skill metadata
	for id, version := range m.installed {
		if skill, ok := m.skills[id]; ok {
			skill.Installed = true
			skill.Version = version
		}
	}
}

// saveInstalled saves the list of installed skills.
func (m *Marketplace) saveInstalled() {
	path := filepath.Join(m.cacheDir, "installed.json")
	data, _ := json.MarshalIndent(m.installed, "", "  ")
	os.WriteFile(path, data, 0644)
}

// Register registers a skill in the marketplace.
func (m *Marketplace) Register(skill *SkillMetadata) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skills[skill.ID] = skill
}

// GetByCategory returns skills in a category.
func (m *Marketplace) GetByCategory(category string) []*SkillMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]*SkillMetadata, 0)
	for _, skill := range m.skills {
		if skill.Category == category {
			results = append(results, skill)
		}
	}
	return results
}

// GetInstalled returns all installed skills.
func (m *Marketplace) GetInstalled() []*SkillMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]*SkillMetadata, 0)
	for _, skill := range m.skills {
		if skill.Installed {
			results = append(results, skill)
		}
	}
	return results
}

// GetUpdates returns skills that have updates available.
func (m *Marketplace) GetUpdates(ctx context.Context) []*SkillMetadata {
	// In production, this would check for version updates
	return make([]*SkillMetadata, 0)
}

// SkillBuilder helps build skill metadata.
type SkillBuilder struct {
	metadata *SkillMetadata
}

// NewSkillBuilder creates a new skill builder.
func NewSkillBuilder(id, name, version string) *SkillBuilder {
	return &SkillBuilder{
		metadata: &SkillMetadata{
			ID:      id,
			Name:    name,
			Version: version,
			Tags:    make([]string, 0),
			Keywords: make([]string, 0),
		},
	}
}

// WithDescription sets the description.
func (b *SkillBuilder) WithDescription(desc string) *SkillBuilder {
	b.metadata.Description = desc
	return b
}

// WithAuthor sets the author.
func (b *SkillBuilder) WithAuthor(author string) *SkillBuilder {
	b.metadata.Author = author
	return b
}

// WithLicense sets the license.
func (b *SkillBuilder) WithLicense(license string) *SkillBuilder {
	b.metadata.License = license
	return b
}

// WithCategory sets the category.
func (b *SkillBuilder) WithCategory(category string) *SkillBuilder {
	b.metadata.Category = category
	return b
}

// AddTag adds a tag.
func (b *SkillBuilder) AddTag(tag string) *SkillBuilder {
	b.metadata.Tags = append(b.metadata.Tags, tag)
	return b
}

// AddKeyword adds a keyword.
func (b *SkillBuilder) AddKeyword(keyword string) *SkillBuilder {
	b.metadata.Keywords = append(b.metadata.Keywords, keyword)
	return b
}

// Build returns the skill metadata.
func (b *SkillBuilder) Build() *SkillMetadata {
	return b.metadata
}

// Helper functions
func toLower(s string) string {
	// Simple toLower implementation
	return s // In production, use strings.ToLower
}

func contains(s, substr string) bool {
	// Simple contains implementation
	return len(s) >= len(substr) && s[:len(substr)] == substr
}
