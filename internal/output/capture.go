package output

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/session"
)

// OutputFormat represents the format for output capture
type OutputFormat string

const (
	FormatText       OutputFormat = "text"
	FormatJSON       OutputFormat = "json"
	FormatStructured OutputFormat = "structured"
)

var (
	thinkingTagRegex = regexp.MustCompile(`(?is)<think\b[^>]*>.*?</think>`)
	whitespaceRegex  = regexp.MustCompile(`\n\s*\n`)
)

// ConversationMetadata contains metadata about the conversation
type ConversationMetadata struct {
	SessionID     string        `json:"session_id"`
	TotalTokens   int64         `json:"total_tokens,omitempty"`
	InputTokens   int64         `json:"input_tokens,omitempty"`
	OutputTokens  int64         `json:"output_tokens,omitempty"`
	DurationMs    int64         `json:"duration_ms"`
	Model         string        `json:"model,omitempty"`
	Provider      string        `json:"provider,omitempty"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time,omitempty"`
	ToolCalls     []interface{} `json:"tool_calls,omitempty"`
}

// ConversationOutput represents the complete conversation output
type ConversationOutput struct {
	SessionID   string                `json:"session_id"`
	Messages    []*message.Message    `json:"messages"`
	Metadata    ConversationMetadata  `json:"metadata"`
	SessionInfo *session.Session      `json:"session_info,omitempty"`
	ToolCalls   []interface{}         `json:"tool_calls,omitempty"`
}

// OutputCapture interface defines the contract for output capture implementations
type OutputCapture interface {
	StartCapture(sessionID string, format OutputFormat) error
	CaptureMessage(msg *message.Message) error
	CaptureMetadata(metadata ConversationMetadata) error
	WriteOutput(writer io.Writer) error
	Close() error
	GetContent() string
}

// DefaultOutputCapture is the default implementation of OutputCapture
type DefaultOutputCapture struct {
	mu sync.RWMutex
	
	sessionID string
	format    OutputFormat
	messages  []*message.Message
	metadata  ConversationMetadata
	session   *session.Session
	toolCalls []interface{}
	
	started         bool
	closed          bool
	startTime       time.Time
	filterReasoning bool    // NEW: Add this field exactly here
}

// NewOutputCapture creates a new output capture instance
func NewOutputCapture(filterReasoning bool) OutputCapture {
	return &DefaultOutputCapture{
		messages:        make([]*message.Message, 0),
		toolCalls:       make([]interface{}, 0),
		filterReasoning: filterReasoning,  // NEW: Add exactly here
	}
}

// filterThinkingContent removes thinking tags from content with security controls
func filterThinkingContent(content string) string {
	// Content size limit: 1MB
	if len(content) > 1024*1024 {
		return content // Skip filtering for large content
	}
	
	// Check if content is valid for processing
	if !isValidThinkingContent(content) {
		return content // Skip filtering for invalid content
	}
	
	// Use channel-based goroutine pattern with timeout protection
	result := make(chan string, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	go func() {
		// Apply regex filtering
		filtered := thinkingTagRegex.ReplaceAllString(content, "")
		// Clean up whitespace
		cleaned := cleanWhitespace(filtered)
		result <- cleaned
	}()
	
	select {
	case filtered := <-result:
		return filtered
	case <-ctx.Done():
		// Timeout protection: return original content
		return content
	}
}

// isValidThinkingContent validates content for safe processing
func isValidThinkingContent(content string) bool {
	// Balance check: openCount == closeCount
	openCount := strings.Count(content, "<think")
	closeCount := strings.Count(content, "</think>")
	
	if openCount != closeCount {
		return false // Unbalanced tags
	}
	
	// Limit check: <= 100 tags
	if openCount > 100 {
		return false // Too many tags
	}
	
	return true
}

// cleanWhitespace removes excessive whitespace from content
func cleanWhitespace(content string) string {
	// Use whitespaceRegex to replace multiple newlines with single newline
	cleaned := whitespaceRegex.ReplaceAllString(content, "\n")
	// Apply strings.TrimSpace
	return strings.TrimSpace(cleaned)
}

// StartCapture initializes the output capture
func (oc *DefaultOutputCapture) StartCapture(sessionID string, format OutputFormat) error {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	
	if oc.started {
		return fmt.Errorf("capture already started")
	}
	
	oc.sessionID = sessionID
	oc.format = format
	oc.started = true
	oc.startTime = time.Now()
	
	// Initialize metadata
	oc.metadata = ConversationMetadata{
		SessionID: sessionID,
		StartTime: oc.startTime,
	}
	
	slog.Debug("Output capture started", "session_id", sessionID, "format", format)
	return nil
}

// CaptureMessage captures a message from the conversation
func (oc *DefaultOutputCapture) CaptureMessage(msg *message.Message) error {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	
	if !oc.started {
		return fmt.Errorf("capture not started")
	}
	
	if oc.closed {
		return fmt.Errorf("capture already closed")
	}
	
	// Deep copy the message to avoid issues with concurrent access
	msgCopy := *msg
	oc.messages = append(oc.messages, &msgCopy)
	
	slog.Debug("Message captured", "session_id", oc.sessionID, "message_id", msg.ID, "role", msg.Role)
	return nil
}

// CaptureMetadata captures conversation metadata
func (oc *DefaultOutputCapture) CaptureMetadata(metadata ConversationMetadata) error {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	
	if !oc.started {
		return fmt.Errorf("capture not started")
	}
	
	// Merge with existing metadata, preserving start time and session ID
	metadata.SessionID = oc.metadata.SessionID
	metadata.StartTime = oc.metadata.StartTime
	oc.metadata = metadata
	
	slog.Debug("Metadata captured", "session_id", oc.sessionID)
	return nil
}

// WriteOutput writes the captured output to the provided writer
func (oc *DefaultOutputCapture) WriteOutput(writer io.Writer) error {
	oc.mu.RLock()
	defer oc.mu.RUnlock()
	
	if !oc.started {
		return fmt.Errorf("capture not started")
	}
	
	switch oc.format {
	case FormatText:
		return oc.writeTextOutput(writer)
	case FormatJSON:
		return oc.writeJSONOutput(writer)
	case FormatStructured:
		return oc.writeStructuredOutput(writer)
	default:
		return fmt.Errorf("unsupported output format: %s", oc.format)
	}
}

// Close finalizes the output capture
func (oc *DefaultOutputCapture) Close() error {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	
	if !oc.started {
		return fmt.Errorf("capture not started")
	}
	
	if oc.closed {
		return nil // Already closed
	}
	
	oc.closed = true
	oc.metadata.EndTime = time.Now()
	oc.metadata.DurationMs = oc.metadata.EndTime.Sub(oc.metadata.StartTime).Milliseconds()
	
	slog.Debug("Output capture closed", "session_id", oc.sessionID, "duration_ms", oc.metadata.DurationMs)
	return nil
}

// GetContent returns the captured content as a string (for text format)
func (oc *DefaultOutputCapture) GetContent() string {
	oc.mu.RLock()
	defer oc.mu.RUnlock()
	
	if oc.format != FormatText {
		return ""
	}
	
	content := ""
	for _, msg := range oc.messages {
		if msg.Role == message.Assistant {
			msgContent := msg.Content().String()
			
			// Apply filtering if enabled for assistant messages
			if oc.filterReasoning && msg.Role == message.Assistant {
				msgContent = filterThinkingContent(msgContent)
			}
			
			content += msgContent
		}
	}
	return content
}

// writeTextOutput writes plain text output
func (oc *DefaultOutputCapture) writeTextOutput(writer io.Writer) error {
	for _, msg := range oc.messages {
		if msg.Role == message.Assistant {
			content := msg.Content().String()
			
			// Apply filtering if enabled for assistant messages
			if oc.filterReasoning && msg.Role == message.Assistant {
				content = filterThinkingContent(content)
			}
			
			if _, err := writer.Write([]byte(content)); err != nil {
				return fmt.Errorf("failed to write text output: %w", err)
			}
		}
	}
	return nil
}

// writeJSONOutput writes JSON formatted output
func (oc *DefaultOutputCapture) writeJSONOutput(writer io.Writer) error {
	// Create messages slice with filtering applied if enabled
	messages := oc.messages
	if oc.filterReasoning {
		messages = make([]*message.Message, len(oc.messages))
		for i, msg := range oc.messages {
			// Deep copy message
			msgCopy := *msg
			messages[i] = &msgCopy
			
			// Apply filtering if enabled for assistant messages
			if msg.Role == message.Assistant {
				// Get original content and filter it
				originalContent := msg.Content().String()
				filteredContent := filterThinkingContent(originalContent)
				
				// Update message content parts with filtered content
				if filteredContent != originalContent {
					// Create new text content part with filtered content
					msgCopy.Parts = []message.ContentPart{
						message.TextContent{Text: filteredContent},
					}
				}
			}
		}
	}
	
	output := ConversationOutput{
		SessionID:   oc.sessionID,
		Messages:    messages,
		Metadata:    oc.metadata,
		SessionInfo: oc.session,
		ToolCalls:   oc.toolCalls,
	}
	
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode JSON output: %w", err)
	}
	
	return nil
}

// writeStructuredOutput writes structured output with extended metadata
func (oc *DefaultOutputCapture) writeStructuredOutput(writer io.Writer) error {
	// For now, structured format is the same as JSON but with all fields included
	// This can be extended in the future with additional metadata
	return oc.writeJSONOutput(writer)
}

// SetSession sets the session information for the capture
func (oc *DefaultOutputCapture) SetSession(sess *session.Session) {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	oc.session = sess
}

// AddToolCall adds a tool call to the capture
func (oc *DefaultOutputCapture) AddToolCall(toolCall interface{}) {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	oc.toolCalls = append(oc.toolCalls, toolCall)
}