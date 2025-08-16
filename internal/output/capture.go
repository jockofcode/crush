package output

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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
	
	started   bool
	closed    bool
	startTime time.Time
}

// NewOutputCapture creates a new output capture instance
func NewOutputCapture() OutputCapture {
	return &DefaultOutputCapture{
		messages:  make([]*message.Message, 0),
		toolCalls: make([]interface{}, 0),
	}
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
			content += msg.Content().String()
		}
	}
	return content
}

// writeTextOutput writes plain text output
func (oc *DefaultOutputCapture) writeTextOutput(writer io.Writer) error {
	for _, msg := range oc.messages {
		if msg.Role == message.Assistant {
			content := msg.Content().String()
			if _, err := writer.Write([]byte(content)); err != nil {
				return fmt.Errorf("failed to write text output: %w", err)
			}
		}
	}
	return nil
}

// writeJSONOutput writes JSON formatted output
func (oc *DefaultOutputCapture) writeJSONOutput(writer io.Writer) error {
	output := ConversationOutput{
		SessionID:   oc.sessionID,
		Messages:    oc.messages,
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