// Package webhook handles incoming webhooks from CI/CD platforms
package webhook

import (
	"encoding/json"
	"fmt"
	"time"
)

// Event represents a normalized webhook event
type Event struct {
	// Platform that sent the event
	Platform Platform

	// Type of event
	Type EventType

	// PR/MR identifier
	PRID int

	// Repository information
	Repo     string
	RepoID   int
	Owner    string
	FullName string

	// Commit information
	SHA     string
	BaseRef string
	HeadRef string

	// PR/MR metadata
	Title       string
	Description string
	Author      string

	// Raw payload for debugging
	RawPayload json.RawMessage

	// When the event was received
	Timestamp time.Time
}

// Platform identifies the source platform
type Platform string

const (
	PlatformGitHub    Platform = "github"
	PlatformGitLab    Platform = "gitlab"
	PlatformGitee     Platform = "gitee"
	MaxRawPayloadSize          = 10 * 1024 * 1024 // 10MB limit for raw payload storage
)

// EventType represents webhook event types
type EventType string

const (
	EventPROpened      EventType = "opened"
	EventPRSynchronize EventType = "synchronize"
	EventPRReopened    EventType = "reopened"
	EventPRClosed      EventType = "closed"
	EventPRMerged      EventType = "merged"
	EventPROpenedGL    EventType = "open"   // GitLab
	EventPRUpdatedGL   EventType = "update" // GitLab
)

// String returns the string representation of the event type
func (e EventType) String() string {
	return string(e)
}

// ShouldTriggerReview returns true if this event should trigger a review
func (e *Event) ShouldTriggerReview() bool {
	switch e.Type {
	case EventPROpened, EventPRSynchronize, EventPRReopened, EventPROpenedGL, EventPRUpdatedGL:
		return true
	default:
		return false
	}
}

// GitHubWebhook represents a GitHub webhook event payload
type GitHubWebhook struct {
	// Action is the action that triggered the event
	Action string `json:"action"`

	// Number is the PR number
	Number int `json:"number"`

	// PullRequest contains PR details
	PullRequest struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		Body    string `json:"body"`
		State   string `json:"state"`
		HTMLURL string `json:"html_url"`
		User    struct {
			Login string `json:"login"`
		} `json:"user"`
		Head struct {
			Ref  string `json:"ref"`
			SHA  string `json:"sha"`
			Repo struct {
				Name     string `json:"name"`
				FullName string `json:"full_name"`
				Owner    struct {
					Login string `json:"login"`
				} `json:"owner"`
			} `json:"repo"`
		} `json:"head"`
		Base struct {
			Ref  string `json:"ref"`
			SHA  string `json:"sha"`
			Repo struct {
				Name     string `json:"name"`
				FullName string `json:"full_name"`
				Owner    struct {
					Login string `json:"login"`
				} `json:"owner"`
			} `json:"repo"`
		} `json:"base"`
	} `json:"pull_request"`

	// Repository contains repo details
	Repository struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		Owner    struct {
			Login string `json:"login"`
		} `json:"owner"`
		Private bool `json:"private"`
	} `json:"repository"`

	// Sender contains the user who triggered the event
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`

	// Installation contains installation details (for GitHub Apps)
	Installation *struct {
		ID int64 `json:"id"`
	} `json:"installation"`
}

// ParseGitHubEvent parses a GitHub webhook event
func ParseGitHubEvent(data []byte, eventType string) (*Event, error) {
	// Handle ping event
	if eventType == "ping" {
		return nil, nil // Ignore ping events
	}

	var payload GitHubWebhook
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub payload: %w", err)
	}

	// Only process pull_request events
	if eventType != "pull_request" {
		return nil, nil
	}

	// Map GitHub action to our event type
	var evtType EventType
	switch payload.Action {
	case "opened", "reopened", "synchronize", "edited":
		evtType = EventPROpened
		// For synchronize, use the correct type
		if payload.Action == "synchronize" {
			evtType = EventPRSynchronize
		}
		if payload.Action == "reopened" {
			evtType = EventPRReopened
		}
	default:
		// Ignore other actions like closed, merged, assigned, etc.
		return nil, nil
	}

	// Validate required fields
	if payload.PullRequest.Number <= 0 {
		return nil, fmt.Errorf("invalid PR number: %d", payload.PullRequest.Number)
	}

	// Limit raw payload size to prevent memory issues
	rawPayload := data
	if len(data) > MaxRawPayloadSize {
		rawPayload = data[:MaxRawPayloadSize]
	}

	event := &Event{
		Platform:    PlatformGitHub,
		Type:        evtType,
		PRID:        payload.PullRequest.Number,
		Repo:        nonEmptyString(payload.Repository.Name, "unknown"),
		RepoID:      int(payload.Repository.ID),
		Owner:       nonEmptyString(payload.Repository.Owner.Login, "unknown"),
		FullName:    nonEmptyString(payload.Repository.FullName, "unknown"),
		SHA:         payload.PullRequest.Head.SHA,
		BaseRef:     payload.PullRequest.Base.Ref,
		HeadRef:     payload.PullRequest.Head.Ref,
		Title:       nonEmptyString(payload.PullRequest.Title, "Untitled"),
		Description: payload.PullRequest.Body, // Body can be empty
		Author:      nonEmptyString(payload.PullRequest.User.Login, "unknown"),
		RawPayload:  rawPayload,
	}

	return event, nil
}

// nonEmptyString returns the string if non-empty, otherwise returns the default value
func nonEmptyString(s, defaultValue string) string {
	if s == "" {
		return defaultValue
	}
	return s
}

// GitLabWebhook represents a GitLab webhook event payload
type GitLabWebhook struct {
	// ObjectKind is the type of event
	ObjectKind string `json:"object_kind"`

	// EventType is the specific event type
	EventType string `json:"event_type"`

	// User who triggered the event
	User struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"user"`

	// Project details
	Project struct {
		ID                int64  `json:"id"`
		Name              string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
		WebURL            string `json:"web_url"`
	} `json:"project"`

	// ObjectAttributes contains MR details
	ObjectAttributes struct {
		ID           int64  `json:"id"`
		IID          int    `json:"iid"` // Merge Request IID (user-facing number)
		Title        string `json:"title"`
		Description  string `json:"description"`
		State        string `json:"state"`
		Action       string `json:"action"`
		SourceBranch string `json:"source_branch"`
		TargetBranch string `json:"target_branch"`
		Source       struct {
			ID       int64  `json:"id"`
			Name     string `json:"name"`
			FullName string `json:"full_name"`
		} `json:"source"`
		Target struct {
			ID       int64  `json:"id"`
			Name     string `json:"name"`
			FullName string `json:"full_name"`
		} `json:"target"`
		LastCommit struct {
			ID string `json:"id"`
		} `json:"last_commit"`
		HeadPipelineID string `json:"head_pipeline_id"`
		URL            string `json:"url"`
	} `json:"object_attributes"`

	// Labels assigned to the MR
	Labels []struct {
		ID       int64  `json:"id"`
		Title    string `json:"title"`
		Color    string `json:"color"`
		Template bool   `json:"template"`
	} `json:"labels"`
}

// ParseGitLabEvent parses a GitLab webhook event
func ParseGitLabEvent(data []byte, eventType string) (*Event, error) {
	var payload GitLabWebhook
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse GitLab payload: %w", err)
	}

	// Only process merge_request events
	if payload.ObjectKind != "merge_request" {
		return nil, nil
	}

	// Map GitLab action to our event type
	var evtType EventType
	switch payload.ObjectAttributes.Action {
	case "open", "reopen", "update", "merge":
		evtType = EventPROpenedGL
		if payload.ObjectAttributes.Action == "update" {
			// For GitLab, update can mean many things
			// We'll treat it as a potential review trigger
			evtType = EventPRUpdatedGL
		}
	default:
		// Ignore other actions like close, approved, etc.
		return nil, nil
	}

	// Extract owner/repo from path_with_namespace
	fullName := payload.Project.PathWithNamespace

	// Validate required fields
	if payload.ObjectAttributes.IID <= 0 {
		return nil, fmt.Errorf("invalid MR number: %d", payload.ObjectAttributes.IID)
	}

	// Limit raw payload size to prevent memory issues
	rawPayload := data
	if len(data) > MaxRawPayloadSize {
		rawPayload = data[:MaxRawPayloadSize]
	}

	event := &Event{
		Platform:    PlatformGitLab,
		Type:        evtType,
		PRID:        payload.ObjectAttributes.IID,
		Repo:        nonEmptyString(payload.Project.Name, "unknown"),
		RepoID:      int(payload.Project.ID),
		Owner:       nonEmptyString(payload.User.Username, "unknown"),
		FullName:    nonEmptyString(fullName, "unknown"),
		SHA:         payload.ObjectAttributes.LastCommit.ID,
		BaseRef:     payload.ObjectAttributes.TargetBranch,
		HeadRef:     payload.ObjectAttributes.SourceBranch,
		Title:       nonEmptyString(payload.ObjectAttributes.Title, "Untitled"),
		Description: payload.ObjectAttributes.Description,
		Author:      nonEmptyString(payload.User.Username, "unknown"),
		RawPayload:  rawPayload,
	}

	return event, nil
}
