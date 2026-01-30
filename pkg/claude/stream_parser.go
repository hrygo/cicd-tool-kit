// Package claude provides streaming JSON output parser for Claude CLI
// Based on production best practices from docs/BEST_PRACTICE_CLI_AGENT.md section 7.3
package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// StreamEventType represents the type of stream event
type StreamEventType string

const (
	// EventTypeMessage is a complete message
	EventTypeMessage StreamEventType = "message"
	// EventTypeContentBlockDelta is streaming text content
	EventTypeContentBlockDelta StreamEventType = "content_block_delta"
	// EventTypeToolUse indicates a tool is being used
	EventTypeToolUse StreamEventType = "tool_use"
	// EventTypeResult is the final result
	EventTypeResult StreamEventType = "result"
	// EventTypeError indicates an error occurred
	EventTypeError StreamEventType = "error"
	// EventTypeThinking is the thinking block
	EventTypeThinking StreamEventType = "thinking"
)

// StreamEvent represents a single event in the stream
type StreamEvent struct {
	Type      StreamEventType     `json:"type"`
	Timestamp string              `json:"timestamp,omitempty"`
	Data      json.RawMessage     `json:"data,omitempty"`
	Error     string              `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// StreamParser parses streaming JSON output from Claude CLI
// Implements stream-json parsing from docs/BEST_PRACTICE_CLI_AGENT.md section 7.3
type StreamParser struct {
	scanner *bufio.Scanner
	handler EventHandler
	errors  []string
}

// EventHandler handles stream events as they are parsed
type EventHandler interface {
	// OnMessage is called when a complete message is received
	OnMessage(event StreamEvent)

	// OnContentDelta is called when streaming content is received
	OnContentDelta(event StreamEvent)

	// OnToolUse is called when a tool is being used
	OnToolUse(event StreamEvent)

	// OnResult is called when the final result is received
	OnResult(event StreamEvent)

	// OnError is called when an error is encountered
	OnError(event StreamEvent)

	// OnThinking is called when thinking block content is received
	OnThinking(event StreamEvent)
}

// DefaultEventHandler provides a no-op implementation of EventHandler
type DefaultEventHandler struct{}

func (h *DefaultEventHandler) OnMessage(event StreamEvent)    {}
func (h *DefaultEventHandler) OnContentDelta(event StreamEvent) {}
func (h *DefaultEventHandler) OnToolUse(event StreamEvent)     {}
func (h *DefaultEventHandler) OnResult(event StreamEvent)      {}
func (h *DefaultEventHandler) OnError(event StreamEvent)       {}
func (h *DefaultEventHandler) OnThinking(event StreamEvent)    {}

// NewStreamParser creates a new stream parser
func NewStreamParser(r io.Reader, handler EventHandler) *StreamParser {
	if handler == nil {
		handler = &DefaultEventHandler{}
	}

	return &StreamParser{
		scanner: bufio.NewScanner(r),
		handler: handler,
		errors:  make([]string, 0),
	}
}

// Parse reads and parses the stream until EOF
// Returns any errors encountered during parsing
func (p *StreamParser) Parse() error {
	for p.scanner.Scan() {
		line := p.scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Try to parse as JSON
		var event StreamEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Not a JSON line, could be raw text output
			// Store as potential error or continue
			if p.isErrorLine(line) {
				p.errors = append(p.errors, line)
				p.handler.OnError(StreamEvent{
					Type:  EventTypeError,
					Error: line,
				})
			}
			continue
		}

		// Dispatch event based on type
		p.dispatch(event)
	}

	if err := p.scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

// dispatch routes the event to the appropriate handler
func (p *StreamParser) dispatch(event StreamEvent) {
	switch event.Type {
	case EventTypeMessage:
		p.handler.OnMessage(event)
	case EventTypeContentBlockDelta:
		p.handler.OnContentDelta(event)
	case EventTypeToolUse:
		p.handler.OnToolUse(event)
	case EventTypeResult:
		p.handler.OnResult(event)
	case EventTypeError:
		p.handler.OnError(event)
	case EventTypeThinking:
		p.handler.OnThinking(event)
	default:
		// Unknown event type, treat as message
		p.handler.OnMessage(event)
	}
}

// isErrorLine checks if a line appears to be an error message
func (p *StreamParser) isErrorLine(line string) bool {
	lower := strings.ToLower(line)
	errorIndicators := []string{
		"error", "failed", "exception", "cannot", "unable",
		"fatal", "panic", "denied", "forbidden",
	}

	for _, indicator := range errorIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}

	return false
}

// GetErrors returns any errors encountered during parsing
func (p *StreamParser) GetErrors() []string {
	return p.errors
}

// HasErrors returns true if any errors were encountered
func (p *StreamParser) HasErrors() bool {
	return len(p.errors) > 0
}

// BufferedEventHandler buffers all events for later processing
type BufferedEventHandler struct {
	events []StreamEvent
}

// NewBufferedEventHandler creates a new buffered event handler
func NewBufferedEventHandler() *BufferedEventHandler {
	return &BufferedEventHandler{
		events: make([]StreamEvent, 0),
	}
}

func (h *BufferedEventHandler) record(event StreamEvent) {
	h.events = append(h.events, event)
}

func (h *BufferedEventHandler) OnMessage(event StreamEvent)    { h.record(event) }
func (h *BufferedEventHandler) OnContentDelta(event StreamEvent) { h.record(event) }
func (h *BufferedEventHandler) OnToolUse(event StreamEvent)     { h.record(event) }
func (h *BufferedEventHandler) OnResult(event StreamEvent)      { h.record(event) }
func (h *BufferedEventHandler) OnError(event StreamEvent)       { h.record(event) }
func (h *BufferedEventHandler) OnThinking(event StreamEvent)    { h.record(event) }

// GetEvents returns all buffered events
func (h *BufferedEventHandler) GetEvents() []StreamEvent {
	return h.events
}

// GetEventsByType returns events filtered by type
func (h *BufferedEventHandler) GetEventsByType(eventType StreamEventType) []StreamEvent {
	result := make([]StreamEvent, 0)
	for _, event := range h.events {
		if event.Type == eventType {
			result = append(result, event)
		}
	}
	return result
}

// GetErrorEvents returns all error events
func (h *BufferedEventHandler) GetErrorEvents() []StreamEvent {
	return h.GetEventsByType(EventTypeError)
}

// HasErrors returns true if any error events were recorded
func (h *BufferedEventHandler) HasErrors() bool {
	return len(h.GetErrorEvents()) > 0
}

// Clear removes all buffered events
func (h *BufferedEventHandler) Clear() {
	h.events = make([]StreamEvent, 0)
}

// ParseStreamJSON parses stream-json format output from a string
// This is a convenience function for one-shot parsing
func ParseStreamJSON(output string) (*BufferedEventHandler, error) {
	handler := NewBufferedEventHandler()
	parser := NewStreamParser(strings.NewReader(output), handler)

	if err := parser.Parse(); err != nil {
		return handler, err
	}

	return handler, nil
}

// CollectIssuesFromStream extracts issues from stream events
// Useful for code review workflows
func CollectIssuesFromStream(events []StreamEvent) []Issue {
	issues := make([]Issue, 0)

	for _, event := range events {
		if event.Type == EventTypeResult && len(event.Data) > 0 {
			var result struct {
				Issues []Issue `json:"issues"`
			}
			if err := json.Unmarshal(event.Data, &result); err == nil {
				issues = append(issues, result.Issues...)
			}
		}
	}

	return issues
}

// ExtractTextFromContentDeltas extracts all text from content block delta events
func ExtractTextFromContentDeltas(events []StreamEvent) string {
	var sb strings.Builder

	for _, event := range events {
		if event.Type == EventTypeContentBlockDelta && len(event.Data) > 0 {
			var delta struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal(event.Data, &delta); err == nil {
				sb.WriteString(delta.Text)
			}
		}
	}

	return sb.String()
}
