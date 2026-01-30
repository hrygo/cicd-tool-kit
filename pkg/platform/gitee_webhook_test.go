// Package platform tests for Gitee webhook functionality
package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultWebhookConfig(t *testing.T) {
	config := DefaultWebhookConfig()

	if config.Address != ":8080" {
		t.Errorf("Address = %s, want :8080", config.Address)
	}
	if config.Path != "/webhook" {
		t.Errorf("Path = %s, want /webhook", config.Path)
	}
	if config.ReadTimeout != 10*time.Second {
		t.Errorf("ReadTimeout = %v, want 10s", config.ReadTimeout)
	}
	if config.WriteTimeout != 10*time.Second {
		t.Errorf("WriteTimeout = %v, want 10s", config.WriteTimeout)
	}
}

func TestNewWebhookServer(t *testing.T) {
	config := WebhookConfig{
		Address: ":9090",
		Secret:  "test-secret",
		Path:    "/hook",
	}

	server := NewWebhookServer(config)

	if server == nil {
		t.Fatal("NewWebhookServer returned nil")
	}
	if server.secret != "test-secret" {
		t.Errorf("secret = %s, want test-secret", server.secret)
	}
	if server.handlers == nil {
		t.Error("handlers map is nil")
	}
}

func TestWebhookServerRegisterHandler(t *testing.T) {
	config := WebhookConfig{}
	server := NewWebhookServer(config)

	handler := func(ctx context.Context, event *GiteeWebhookEvent) error {
		return nil
	}

	server.RegisterHandler(GiteeEventPush, handler)

	if len(server.handlers) != 1 {
		t.Errorf("Expected 1 handler, got %d", len(server.handlers))
	}

	server.UnregisterHandler(GiteeEventPush)

	if len(server.handlers) != 0 {
		t.Errorf("Expected 0 handlers after unregister, got %d", len(server.handlers))
	}
}

func TestWebhookEventTypeConstants(t *testing.T) {
	tests := []struct {
		event GiteeEventType
		value string
	}{
		{GiteeEventPush, "push_hooks"},
		{GiteeEventMergeRequest, "merge_request_hooks"},
		{GiteeEventNote, "note_hooks"},
		{GiteeEventIssue, "issue_hooks"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if string(tt.event) != tt.value {
				t.Errorf("GiteeEventType = %s, want %s", tt.event, tt.value)
			}
		})
	}
}

func TestWebhookServerHandleRequest(t *testing.T) {
	tests := []struct {
		name           string
		secret         string
		method         string
		eventHeader    string
		signature      string
		body           []byte
		expectedStatus int
	}{
		{
			name:           "invalid method",
			method:         "GET",
			secret:         "",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "missing event header (no secret)",
			method:         "POST",
			secret:         "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing signature with secret set",
			method:         "POST",
			secret:         "test-secret",
			eventHeader:    "push_hooks",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewWebhookServer(WebhookConfig{
				Secret: tt.secret,
			})

			// Register a handler
			server.RegisterHandler(GiteeEventPush, func(ctx context.Context, event *GiteeWebhookEvent) error {
				return nil
			})

			req := httptest.NewRequest(tt.method, "/webhook", nil)
			if tt.eventHeader != "" {
				req.Header.Set("X-Gitee-Event", tt.eventHeader)
			}
			if tt.signature != "" {
				req.Header.Set("X-Gitee-Token", tt.signature)
			}
			w := httptest.NewRecorder()

			server.handleWebhook(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestValidateGiteeWebhook(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		eventHeader string
		secret      string
		sigHeader   string
		body        []byte
		wantErr     bool
		errContains string
	}{
		{
			name:        "invalid method",
			method:      "GET",
			wantErr:     true,
			errContains: "invalid method",
		},
		{
			name:        "missing event header",
			method:      "POST",
			wantErr:     true,
			errContains: "missing event type header",
		},
		{
			name:        "missing signature with secret",
			method:      "POST",
			eventHeader: "push_hooks",
			secret:      "test-secret",
			wantErr:     true,
			errContains: "missing signature header",
		},
		{
			name:        "valid request no secret",
			method:      "POST",
			eventHeader: "push_hooks",
			body:        []byte(`{"test": "data"}`),
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				req = httptest.NewRequest(tt.method, "/webhook", bytes.NewReader(tt.body))
			} else {
				req = httptest.NewRequest(tt.method, "/webhook", nil)
			}
			if tt.eventHeader != "" {
				req.Header.Set("X-Gitee-Event", tt.eventHeader)
			}
			if tt.sigHeader != "" {
				req.Header.Set("X-Gitee-Token", tt.sigHeader)
			}

			_, err := ValidateGiteeWebhook(req, tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGiteeWebhook() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errContains, err.Error())
				}
			}
		})
	}
}

func TestParsePushEvent(t *testing.T) {
	data := []byte(`{
		"ref": "refs/heads/main",
		"before": "old123",
		"after": "new123",
		"repository": {
			"id": 123,
			"full_name": "owner/repo"
		},
		"pusher": {
			"login": "user",
			"name": "User"
		},
		"commits": [],
		"total_commits": 0
	}`)

	event, err := ParsePushEvent(data)
	if err != nil {
		t.Fatalf("ParsePushEvent() error = %v", err)
	}

	if event.Ref != "refs/heads/main" {
		t.Errorf("Ref = %s, want refs/heads/main", event.Ref)
	}
	if event.Before != "old123" {
		t.Errorf("Before = %s, want old123", event.Before)
	}
	if event.After != "new123" {
		t.Errorf("After = %s, want new123", event.After)
	}
	if event.Repository == nil {
		t.Error("Repository is nil")
	}
}

func TestParseMergeRequestEvent(t *testing.T) {
	data := []byte(`{
		"action": "open",
		"number": 123,
		"pull_request": {
			"id": 456,
			"number": 123,
			"title": "Test PR"
		},
		"repository": {
			"id": 789,
			"full_name": "owner/repo"
		},
		"sender": {
			"login": "user"
		},
		"timestamp": 1234567890
	}`)

	event, err := ParseMergeRequestEvent(data)
	if err != nil {
		t.Fatalf("ParseMergeRequestEvent() error = %v", err)
	}

	if event.Action != "open" {
		t.Errorf("Action = %s, want open", event.Action)
	}
	if event.Number != 123 {
		t.Errorf("Number = %d, want 123", event.Number)
	}
}

func TestMergeRequestEventIsMethods(t *testing.T) {
	tests := []struct {
		name   string
		action string
		want   map[string]bool
	}{
		{
			name:   "open action",
			action: "open",
			want:   map[string]bool{"IsOpened": true, "IsMerged": false, "IsUpdated": false, "IsClosed": false},
		},
		{
			name:   "merge action",
			action: "merge",
			want:   map[string]bool{"IsOpened": false, "IsMerged": true, "IsUpdated": false, "IsClosed": false},
		},
		{
			name:   "update action",
			action: "update",
			want:   map[string]bool{"IsOpened": false, "IsMerged": false, "IsUpdated": true, "IsClosed": false},
		},
		{
			name:   "synchronize action",
			action: "synchronize",
			want:   map[string]bool{"IsOpened": false, "IsMerged": false, "IsUpdated": true, "IsClosed": false},
		},
		{
			name:   "close action",
			action: "close",
			want:   map[string]bool{"IsOpened": false, "IsMerged": false, "IsUpdated": false, "IsClosed": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := MergeRequestEvent{Action: tt.action}
			if event.IsOpened() != tt.want["IsOpened"] {
				t.Errorf("IsOpened() = %v, want %v", event.IsOpened(), tt.want["IsOpened"])
			}
			if event.IsMerged() != tt.want["IsMerged"] {
				t.Errorf("IsMerged() = %v, want %v", event.IsMerged(), tt.want["IsMerged"])
			}
			if event.IsUpdated() != tt.want["IsUpdated"] {
				t.Errorf("IsUpdated() = %v, want %v", event.IsUpdated(), tt.want["IsUpdated"])
			}
			if event.IsClosed() != tt.want["IsClosed"] {
				t.Errorf("IsClosed() = %v, want %v", event.IsClosed(), tt.want["IsClosed"])
			}
		})
	}
}

func TestWebhookClient(t *testing.T) {
	client := NewWebhookClient("test-secret")

	if client == nil {
		t.Fatal("NewWebhookClient returned nil")
	}
	if client.secret != "test-secret" {
		t.Errorf("secret = %s, want test-secret", client.secret)
	}
	if client.client == nil {
		t.Error("HTTP client is nil")
	}
}

func TestWebhookServerSetLogger(t *testing.T) {
	server := NewWebhookServer(WebhookConfig{})

	var logged string
	server.SetLogger(func(format string, args ...interface{}) {
		logged = format
	})

	server.logger("test %s", "message")

	if logged != "test %s" {
		t.Errorf("logged = %s", logged)
	}
}

func TestGiteeWebhookEventFields(t *testing.T) {
	event := &GiteeWebhookEvent{
		Type:      GiteeEventPush,
		Timestamp: 1234567890,
		Action:    "open",
		Raw:       json.RawMessage(`{"test": "data"}`),
	}

	if event.Type != GiteeEventPush {
		t.Errorf("Type = %s, want push_hooks", event.Type)
	}
	if event.Timestamp != 1234567890 {
		t.Errorf("Timestamp = %d, want 1234567890", event.Timestamp)
	}
	if event.Action != "open" {
		t.Errorf("Action = %s, want open", event.Action)
	}
	if len(event.Raw) == 0 {
		t.Error("Raw is empty")
	}
}

func TestGiteeIssueFields(t *testing.T) {
	issue := GiteeIssue{
		ID:     123,
		Number: 456,
		Title:  "Test Issue",
		Body:   "Issue body",
		State:  "open",
		User: GiteeUser{
			Login: "user",
			Name:  "User",
		},
	}

	if issue.ID != 123 {
		t.Errorf("ID = %d, want 123", issue.ID)
	}
	if issue.Number != 456 {
		t.Errorf("Number = %d, want 456", issue.Number)
	}
	if issue.Title != "Test Issue" {
		t.Errorf("Title = %s, want Test Issue", issue.Title)
	}
}

func TestGiteeNoteFields(t *testing.T) {
	note := GiteeNote{
		ID:           789,
		Body:         "Comment",
		NoteableType: "PullRequest",
		NoteableID:   123,
		User: GiteeUser{
			Login: "user",
		},
		CreatedAt: "2026-01-28T00:00:00Z",
	}

	if note.ID != 789 {
		t.Errorf("ID = %d, want 789", note.ID)
	}
	if note.Body != "Comment" {
		t.Errorf("Body = %s, want Comment", note.Body)
	}
	if note.NoteableType != "PullRequest" {
		t.Errorf("NoteableType = %s, want PullRequest", note.NoteableType)
	}
}

func TestGiteeEnterpriseFields(t *testing.T) {
	ent := GiteeEnterprise{
		ID:   123,
		Name: "Test Enterprise",
		Slug: "test-enterprise",
	}

	if ent.ID != 123 {
		t.Errorf("ID = %d, want 123", ent.ID)
	}
	if ent.Name != "Test Enterprise" {
		t.Errorf("Name = %s, want Test Enterprise", ent.Name)
	}
	if ent.Slug != "test-enterprise" {
		t.Errorf("Slug = %s, want test-enterprise", ent.Slug)
	}
}

func TestPushCommitFields(t *testing.T) {
	commit := PushCommit{
		ID:       "abc123",
		Message:  "Test commit",
		Added:    []string{"file1.go"},
		Removed:  []string{"file2.go"},
		Modified: []string{"file3.go"},
	}

	commit.Author.Name = "Test Author"
	commit.Author.Email = "test@example.com"

	if commit.ID != "abc123" {
		t.Errorf("ID = %s, want abc123", commit.ID)
	}
	if commit.Message != "Test commit" {
		t.Errorf("Message = %s, want Test commit", commit.Message)
	}
	if len(commit.Added) != 1 {
		t.Errorf("Added length = %d, want 1", len(commit.Added))
	}
	if commit.Author.Name != "Test Author" {
		t.Errorf("Author.Name = %s, want Test Author", commit.Author.Name)
	}
}
