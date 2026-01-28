// Package platform provides Gitee webhook server functionality
package platform

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// GiteeEventType represents Gitee webhook event types
type GiteeEventType string

const (
	// GiteeEventPush is triggered when code is pushed
	GiteeEventPush GiteeEventType = "push_hooks"
	// GiteeEventMergeRequest is triggered when a PR is opened/updated/merged
	GiteeEventMergeRequest GiteeEventType = "merge_request_hooks"
	// GiteeEventNote is triggered when a comment is posted
	GiteeEventNote GiteeEventType = "note_hooks"
	// GiteeEventIssue is triggered when an issue is created/updated
	GiteeEventIssue GiteeEventType = "issue_hooks"
)

// GiteeEventHandler processes Gitee webhook events
type GiteeEventHandler func(ctx context.Context, event *GiteeWebhookEvent) error

// WebhookServer handles Gitee webhook events
type WebhookServer struct {
	server   *http.Server
	secret   string
	handlers map[GiteeEventType]GiteeEventHandler
	mu       sync.RWMutex
	// logger is an optional logging function
	logger func(format string, args ...interface{})
}

// GiteeWebhookEvent represents a parsed Gitee webhook event
type GiteeWebhookEvent struct {
	Type      GiteeEventType `json:"hook_name"`
	Timestamp int64          `json:"timestamp"`
	Repo      *GiteeRepo     `json:"repository"`
	PR        *GiteePR       `json:"pull_request"`
	Issue     *GiteeIssue    `json:"issue"`
	Comment   *GiteeNote     `json:"note"`
	Sender    *GiteeUser     `json:"sender"`
	Enterprise *GiteeEnterprise `json:"enterprise"`
	Action    string         `json:"action"` // open, update, merge, close, etc.
	Raw       json.RawMessage `json:"-"`     // Raw payload for custom processing
}

// GiteeIssue represents an issue in Gitee
type GiteeIssue struct {
	ID     int       `json:"id"`
	Number int       `json:"number"`
	Title  string    `json:"title"`
	Body   string    `json:"body"`
	State  string    `json:"state"`
	User   GiteeUser `json:"user"`
}

// GiteeNote represents a comment/note in Gitee
type GiteeNote struct {
	ID           int       `json:"id"`
	Body         string    `json:"body"`
	NoteableType string    `json:"noteable_type"` // Issue, PullRequest
	NoteableID   int       `json:"noteable_id"`
	User         GiteeUser `json:"user"`
	CreatedAt    string    `json:"created_at"`
}

// GiteeEnterprise represents enterprise information
type GiteeEnterprise struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// WebhookConfig represents webhook server configuration
type WebhookConfig struct {
	// Address is the listen address (e.g., ":8080")
	Address string
	// Secret is the webhook secret for signature verification
	Secret string
	// Path is the webhook endpoint path (default: /webhook)
	Path string
	// ReadTimeout is the maximum duration for reading the request
	ReadTimeout time.Duration
	// WriteTimeout is the maximum duration for writing the response
	WriteTimeout time.Duration
}

// DefaultWebhookConfig returns default webhook configuration
func DefaultWebhookConfig() WebhookConfig {
	return WebhookConfig{
		Address:      ":8080",
		Secret:       "",
		Path:         "/webhook",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

// NewWebhookServer creates a new webhook server
func NewWebhookServer(config WebhookConfig) *WebhookServer {
	if config.Path == "" {
		config.Path = "/webhook"
	}

	return &WebhookServer{
		secret:   config.Secret,
		handlers: make(map[GiteeEventType]GiteeEventHandler),
		logger:   func(format string, args ...interface{}) {},
	}
}

// SetLogger sets a custom logging function
func (s *WebhookServer) SetLogger(logger func(format string, args ...interface{})) {
	s.logger = logger
}

// RegisterHandler registers an event handler
func (s *WebhookServer) RegisterHandler(eventType GiteeEventType, handler GiteeEventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[eventType] = handler
}

// UnregisterHandler unregisters an event handler
func (s *WebhookServer) UnregisterHandler(eventType GiteeEventType) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.handlers, eventType)
}

// Start starts the webhook server
func (s *WebhookServer) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", s.handleWebhook)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	s.logger("gitee webhook server starting on %s", addr)
	return s.server.ListenAndServe()
}

// StartWithConfig starts the webhook server with custom configuration
func (s *WebhookServer) StartWithConfig(config WebhookConfig) error {
	mux := http.NewServeMux()
	mux.HandleFunc(config.Path, s.handleWebhook)

	s.server = &http.Server{
		Addr:         config.Address,
		Handler:      mux,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	s.logger("gitee webhook server starting on %s%s", config.Address, config.Path)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the webhook server
func (s *WebhookServer) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	s.logger("gitee webhook server shutting down")
	return s.server.Shutdown(ctx)
}

const maxWebhookBodySize = 1 << 20 // 1MB

// handleWebhook handles incoming webhook requests
func (s *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit body size to prevent unbounded memory usage
	limitedReader := http.MaxBytesReader(w, r.Body, maxWebhookBodySize)
	defer r.Body.Close()

	// Read request body
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		s.logger("failed to read webhook body: %v", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	// Verify signature if secret is configured
	if s.secret != "" {
		signature := r.Header.Get("X-Gitee-Token")
		if signature == "" {
			s.logger("webhook missing signature header")
			http.Error(w, "Missing signature", http.StatusUnauthorized)
			return
		}

		if !s.verifySignature(body, signature) {
			s.logger("webhook signature verification failed")
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Parse event type
	eventType := r.Header.Get("X-Gitee-Event")
	if eventType == "" {
		s.logger("webhook missing event type header")
		http.Error(w, "Missing event type", http.StatusBadRequest)
		return
	}

	s.logger("received webhook event: %s", eventType)

	// Parse webhook payload
	var event GiteeWebhookEvent
	event.Raw = body
	if err := json.Unmarshal(body, &event); err != nil {
		s.logger("failed to parse webhook payload: %v", err)
		http.Error(w, "Failed to parse payload", http.StatusBadRequest)
		return
	}

	event.Type = GiteeEventType(eventType)

	// Get the handler for this event type
	s.mu.RLock()
	handler, ok := s.handlers[event.Type]
	s.mu.RUnlock()

	if !ok {
		s.logger("no handler registered for event type: %s", event.Type)
		// Still return 200, we don't want Gitee to retry
		w.WriteHeader(http.StatusOK)
		return
	}

	// Handle the event in a goroutine for async processing
	// Copy event data to avoid race condition: the goroutine may run
	// after handler returns, invalidating ctx and &event references.
	eventCopy := event
	handlerCopy := handler
	ctx := r.Context()
	go func() {
		if err := handlerCopy(ctx, &eventCopy); err != nil {
			s.logger("handler error for event %s: %v", eventCopy.Type, err)
		}
	}()

	// Respond immediately
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"received","event":"%s"}`, event.Type)
}

// verifySignature verifies the webhook signature
func (s *WebhookServer) verifySignature(payload []byte, signature string) bool {
	// Gitee uses HMAC-SHA256 for webhook signatures
	// The signature is sent as hex string
	expectedMAC := s.generateMAC(payload)
	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	return hmac.Equal(signatureBytes, expectedMAC)
}

// generateMAC generates the HMAC for the payload
func (s *WebhookServer) generateMAC(payload []byte) []byte {
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write(payload)
	return mac.Sum(nil)
}

// ValidateGiteeWebhook validates an incoming webhook request
// This is a utility function that can be used without starting a server
func ValidateGiteeWebhook(r *http.Request, secret string) (*GiteeWebhookEvent, error) {
	// Check method
	if r.Method != http.MethodPost {
		return nil, fmt.Errorf("invalid method: %s", r.Method)
	}

	// Check event type
	eventType := r.Header.Get("X-Gitee-Event")
	if eventType == "" {
		return nil, fmt.Errorf("missing event type header")
	}

	// Limit body size to prevent unbounded memory usage
	limitedReader := http.MaxBytesReader(nil, r.Body, maxWebhookBodySize)
	defer r.Body.Close()

	// Read body
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	// Verify signature if secret provided
	if secret != "" {
		signature := r.Header.Get("X-Gitee-Token")
		if signature == "" {
			return nil, fmt.Errorf("missing signature header")
		}

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expectedMAC := mac.Sum(nil)
		signatureBytes, err := hex.DecodeString(signature)
		if err != nil {
			return nil, fmt.Errorf("invalid signature format: %w", err)
		}

		if !hmac.Equal(signatureBytes, expectedMAC) {
			return nil, fmt.Errorf("signature verification failed")
		}
	}

	// Parse payload
	var event GiteeWebhookEvent
	event.Raw = body
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	event.Type = GiteeEventType(eventType)

	return &event, nil
}

// WebhookClient is a client for sending webhook-like requests
type WebhookClient struct {
	client *http.Client
	secret string
}

// NewWebhookClient creates a new webhook client
func NewWebhookClient(secret string) *WebhookClient {
	return &WebhookClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		secret: secret,
	}
}

// SendWebhook sends a webhook payload to a URL
func (c *WebhookClient) SendWebhook(ctx context.Context, url string, event *GiteeWebhookEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gitee-Event", string(event.Type))

	if c.secret != "" {
		mac := hmac.New(sha256.New, []byte(c.secret))
		mac.Write(payload)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Gitee-Token", signature)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ParsePushEvent parses a push event
func ParsePushEvent(data []byte) (*PushEvent, error) {
	var event PushEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to parse push event: %w", err)
	}
	return &event, nil
}

// PushEvent represents a push webhook event
type PushEvent struct {
	Ref         string      `json:"ref"`         // refs/heads/main
	Before      string      `json:"before"`      // SHA before push
	After       string      `json:"after"`       // SHA after push
	Repository  *GiteeRepo  `json:"repository"`
	Pusher      *GiteeUser  `json:"pusher"`
	Commits     []PushCommit `json:"commits"`
	TotalCommits int         `json:"total_commits"`
}

// PushCommit represents a commit in a push event
type PushCommit struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	Author  struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"author"`
	URL      string   `json:"url"`
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	Modified []string `json:"modified"`
}

// ParseMergeRequestEvent parses a merge request event
func ParseMergeRequestEvent(data []byte) (*MergeRequestEvent, error) {
	var event MergeRequestEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to parse merge request event: %w", err)
	}
	return &event, nil
}

// MergeRequestEvent represents a merge request webhook event
type MergeRequestEvent struct {
	Action     string            `json:"action"` // open, update, merge, close
	Number     int               `json:"number"`
	PR         *GiteePR          `json:"pull_request"`
	Repository *GiteeRepo        `json:"repository"`
	Sender     *GiteeUser        `json:"sender"`
	Enterprise *GiteeEnterprise  `json:"enterprise"`
	Timestamp  int64             `json:"timestamp"`
}

// IsPRAction checks if the event is a specific PR action
func (e *MergeRequestEvent) IsPRAction(action string) bool {
	return e.Action == action
}

// IsOpened checks if the PR was just opened
func (e *MergeRequestEvent) IsOpened() bool {
	return e.Action == "open"
}

// IsMerged checks if the PR was merged
func (e *MergeRequestEvent) IsMerged() bool {
	return e.Action == "merge"
}

// IsUpdated checks if the PR was updated
func (e *MergeRequestEvent) IsUpdated() bool {
	return e.Action == "update" || e.Action == "synchronize"
}

// IsClosed checks if the PR was closed without merging
func (e *MergeRequestEvent) IsClosed() bool {
	return e.Action == "close"
}

// WebhookMiddleware provides middleware for webhook processing
type WebhookMiddleware struct {
	secret string
	next   http.Handler
}

// NewWebhookMiddleware creates new webhook middleware
func NewWebhookMiddleware(secret string, next http.Handler) *WebhookMiddleware {
	return &WebhookMiddleware{
		secret: secret,
		next:   next,
	}
}

// ServeHTTP implements the http.Handler interface
func (m *WebhookMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Verify signature for POST requests
	if r.Method == http.MethodPost && m.secret != "" {
		signature := r.Header.Get("X-Gitee-Token")
		if signature == "" {
			http.Error(w, "Missing signature", http.StatusUnauthorized)
			return
		}

		// Limit body size to prevent unbounded memory usage
		limitedReader := http.MaxBytesReader(w, r.Body, maxWebhookBodySize)
		body, err := io.ReadAll(limitedReader)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		r.Body.Close()

		mac := hmac.New(sha256.New, []byte(m.secret))
		mac.Write(body)
		expectedMAC := mac.Sum(nil)
		signatureBytes, err := hex.DecodeString(signature)
		if err != nil || !hmac.Equal(signatureBytes, expectedMAC) {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}

		// Replace the body for the next handler
		r.Body = io.NopCloser(bytes.NewReader(body))
	}

	m.next.ServeHTTP(w, r)
}
