// Package webhook handles incoming webhooks from CI/CD platforms
// This file contains basic tests to establish test coverage

package webhook

import (
	"encoding/json"
	"testing"
	"time"
)

// TestPlatformConstants verifies platform constants are properly defined
func TestPlatformConstants(t *testing.T) {
	platforms := []Platform{
		PlatformGitHub,
		PlatformGitLab,
		PlatformGitee,
	}

	expected := []string{"github", "gitlab", "gitee"}
	for i, p := range platforms {
		if string(p) != expected[i] {
			t.Errorf("Platform[%d] = %s, want %s", i, p, expected[i])
		}
	}
}

// TestEventTypeString verifies EventType String method
func TestEventTypeString(t *testing.T) {
	events := []EventType{
		EventPROpened,
		EventPRSynchronize,
		EventPRReopened,
		EventPRClosed,
		EventPRMerged,
		EventPROpenedGL,
		EventPRUpdatedGL,
	}

	// Just verify String() doesn't panic
	for _, e := range events {
		_ = e.String()
	}
}

// TestShouldTriggerReview verifies review trigger logic
func TestShouldTriggerReview(t *testing.T) {
	testCases := []struct {
		name     string
		event    EventType
		expected bool
	}{
		{"GitHub opened", EventPROpened, true},
		{"GitHub synchronize", EventPRSynchronize, true},
		{"GitHub reopened", EventPRReopened, true},
		{"GitHub closed", EventPRClosed, false},
		{"GitHub merged", EventPRMerged, false},
		{"GitLab opened", EventPROpenedGL, true},
		{"GitLab updated", EventPRUpdatedGL, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := &Event{Type: tc.event}
			if event.ShouldTriggerReview() != tc.expected {
				t.Errorf("ShouldTriggerReview() = %v, want %v for %s",
					event.ShouldTriggerReview(), tc.expected, tc.name)
			}
		})
	}
}

// TestEventJSONMarshal verifies Event can be marshaled
func TestEventJSONMarshal(t *testing.T) {
	event := &Event{
		Platform:    PlatformGitHub,
		Type:        EventPROpened,
		PRID:        123,
		Repo:        "test/repo",
		RepoID:      456,
		Owner:       "testowner",
		FullName:    "testowner/test/repo",
		SHA:         "abc123",
		BaseRef:     "main",
		HeadRef:     "feature",
		Title:       "Test PR",
		Description: "Test description",
		Author:      "testuser",
		RawPayload:  json.RawMessage(`{"test": true}`),
		Timestamp:   time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Marshaled data should not be empty")
	}

	// Verify it can be unmarshaled back
	var unmarshaled Event
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if unmarshaled.Platform != event.Platform {
		t.Errorf("Platform = %s, want %s", unmarshaled.Platform, event.Platform)
	}
	if unmarshaled.PRID != event.PRID {
		t.Errorf("PRID = %d, want %d", unmarshaled.PRID, event.PRID)
	}
}

// TestParseGitHubEventValidInput verifies GitHub event parsing with valid input
func TestParseGitHubEventValidInput(t *testing.T) {
	validPayload := []byte(`{
		"action": "opened",
		"number": 42,
		"pull_request": {
			"number": 42,
			"title": "Test PR",
			"body": "Test body",
			"state": "open",
			"user": {
				"login": "testuser"
			},
			"head": {
				"sha": "abc123",
				"ref": "feature",
				"repo": {
					"name": "test-repo",
					"full_name": "owner/test-repo",
					"owner": {
						"login": "owner"
					}
				}
			},
			"base": {
				"ref": "main",
				"repo": {
					"name": "test-repo",
					"full_name": "owner/test-repo",
					"owner": {
						"login": "owner"
					}
				}
			}
		},
		"repository": {
			"id": 12345,
			"name": "test-repo",
			"full_name": "owner/test-repo",
			"owner": {
				"login": "owner"
			},
			"private": false
		}
	}`)

	event, err := ParseGitHubEvent(validPayload, "pull_request")
	if err != nil {
		t.Fatalf("ParseGitHubEvent failed: %v", err)
	}

	if event == nil {
		t.Fatal("Event should not be nil")
	}

	if event.Platform != PlatformGitHub {
		t.Errorf("Platform = %s, want %s", event.Platform, PlatformGitHub)
	}
	if event.Type != EventPROpened {
		t.Errorf("Type = %s, want %s", event.Type, EventPROpened)
	}
	if event.PRID != 42 {
		t.Errorf("PRID = %d, want 42", event.PRID)
	}
	if event.Repo != "test-repo" {
		t.Errorf("Repo = %s, want test-repo", event.Repo)
	}
	if event.Owner != "owner" {
		t.Errorf("Owner = %s, want owner", event.Owner)
	}
	if event.Title != "Test PR" {
		t.Errorf("Title = %s, want Test PR", event.Title)
	}
	if event.Author != "testuser" {
		t.Errorf("Author = %s, want testuser", event.Author)
	}
}

// TestParseGitHubEventIgnoresNonPullRequest verifies non-pull_request events are ignored
func TestParseGitHubEventIgnoresNonPullRequest(t *testing.T) {
	payload := []byte(`{"action": "created"}`)

	event, err := ParseGitHubEvent(payload, "pull_request")
	if err != nil {
		t.Fatalf("ParseGitHubEvent failed: %v", err)
	}

	if event != nil {
		t.Error("Non-pull_request event should return nil event")
	}
}

// TestParseGitHubEventIgnoresPing verifies ping events are ignored
func TestParseGitHubEventIgnoresPing(t *testing.T) {
	event, err := ParseGitHubEvent([]byte("{}"), "ping")
	if err != nil {
		t.Fatalf("ParseGitHubEvent failed: %v", err)
	}

	if event != nil {
		t.Error("Ping event should return nil event")
	}
}

// TestParseGitLabEventValidInput verifies GitLab event parsing with valid input
func TestParseGitLabEventValidInput(t *testing.T) {
	validPayload := []byte(`{
		"object_kind": "merge_request",
		"event_type": "merge_request",
		"user": {
			"id": 1,
			"name": "Test User",
			"username": "testuser",
			"email": "test@example.com"
		},
		"project": {
			"id": 123,
			"name": "test-project",
			"path_with_namespace": "owner/test-project",
			"web_url": "https://gitee.com/owner/test-project"
		},
		"object_attributes": {
			"id": 456,
			"iid": 789,
			"title": "Test MR",
			"description": "Test description",
			"state": "opened",
			"action": "open",
			"source_branch": "feature",
			"target_branch": "main",
			"source": {
				"id": 1,
				"name": "test-project",
				"full_name": "owner/test-project"
			},
			"target": {
				"id": 2,
				"name": "test-project",
				"full_name": "owner/test-project"
			},
			"last_commit": {
				"id": "abc123"
			}
		}
	}`)

	event, err := ParseGitLabEvent(validPayload, "")
	if err != nil {
		t.Fatalf("ParseGitLabEvent failed: %v", err)
	}

	if event == nil {
		t.Fatal("Event should not be nil")
	}

	if event.Platform != PlatformGitLab {
		t.Errorf("Platform = %s, want %s", event.Platform, PlatformGitLab)
	}
	if event.Type != EventPROpenedGL {
		t.Errorf("Type = %s, want %s", event.Type, EventPROpenedGL)
	}
	if event.PRID != 789 {
		t.Errorf("PRID = %d, want 789 (IID)", event.PRID)
	}
	if event.Repo != "test-project" {
		t.Errorf("Repo = %s, want test-project", event.Repo)
	}
	if event.Owner != "testuser" {
		t.Errorf("Owner = %s, want testuser", event.Owner)
	}
}

// TestParseGitLabEventIgnoresNonMergeRequest verifies non-MR events are ignored
func TestParseGitLabEventIgnoresNonMergeRequest(t *testing.T) {
	payload := []byte(`{"object_kind": "push"}`)

	event, err := ParseGitLabEvent(payload, "")
	if err != nil {
		t.Fatalf("ParseGitLabEvent failed: %v", err)
	}

	if event != nil {
		t.Error("Non-merge_request event should return nil event")
	}
}
