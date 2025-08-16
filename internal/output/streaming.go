package output

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/charmbracelet/crush/internal/message"
)

// StreamingOutputCapture provides real-time output capture during conversation
type StreamingOutputCapture struct {
	mu sync.RWMutex
	
	sessionID string
	format    OutputFormat
	writer    io.Writer
	
	// Streaming state
	started      bool
	closed       bool
	startTime    time.Time
	bufferSize   int
	
	// JSON streaming state
	jsonEncoder  *json.Encoder
	firstMessage bool
	
	// Content buffering for streaming
	messageBuffer []byte
	contentOffset int
}

// NewStreamingOutputCapture creates a new streaming output capture
func NewStreamingOutputCapture(writer io.Writer, bufferSize int) *StreamingOutputCapture {
	if bufferSize <= 0 {
		bufferSize = 4096 // Default buffer size
	}
	
	return &StreamingOutputCapture{
		writer:        writer,
		bufferSize:    bufferSize,
		messageBuffer: make([]byte, 0, bufferSize),
		firstMessage:  true,
	}
}

// StartCapture initializes streaming capture
func (soc *StreamingOutputCapture) StartCapture(sessionID string, format OutputFormat) error {
	soc.mu.Lock()
	defer soc.mu.Unlock()
	
	if soc.started {
		return fmt.Errorf("streaming capture already started")
	}
	
	soc.sessionID = sessionID
	soc.format = format
	soc.started = true
	soc.startTime = time.Now()
	
	// Initialize format-specific streaming
	switch format {
	case FormatJSON, FormatStructured:
		if err := soc.initJSONStreaming(); err != nil {
			return fmt.Errorf("failed to initialize JSON streaming: %w", err)
		}
	case FormatText:
		// Text format doesn't need special initialization
	default:
		return fmt.Errorf("unsupported streaming format: %s", format)
	}
	
	slog.Debug("Streaming output capture started", "session_id", sessionID, "format", format)
	return nil
}

// CaptureMessage streams a message immediately as it's received
func (soc *StreamingOutputCapture) CaptureMessage(msg *message.Message) error {
	soc.mu.Lock()
	defer soc.mu.Unlock()
	
	if !soc.started {
		return fmt.Errorf("streaming capture not started")
	}
	
	if soc.closed {
		return fmt.Errorf("streaming capture already closed")
	}
	
	switch soc.format {
	case FormatText:
		return soc.streamTextMessage(msg)
	case FormatJSON, FormatStructured:
		return soc.streamJSONMessage(msg)
	default:
		return fmt.Errorf("unsupported streaming format: %s", soc.format)
	}
}

// CaptureMetadata streams metadata (for structured formats)
func (soc *StreamingOutputCapture) CaptureMetadata(metadata ConversationMetadata) error {
	soc.mu.Lock()
	defer soc.mu.Unlock()
	
	if !soc.started {
		return fmt.Errorf("streaming capture not started")
	}
	
	// For streaming, we might want to update metadata in real-time
	// This is more complex for JSON streaming, so we'll handle it in Close()
	return nil
}

// WriteOutput for streaming capture doesn't make sense as content is written in real-time
func (soc *StreamingOutputCapture) WriteOutput(writer io.Writer) error {
	return fmt.Errorf("WriteOutput not supported for streaming capture - content is written in real-time")
}

// Close finalizes the streaming capture
func (soc *StreamingOutputCapture) Close() error {
	soc.mu.Lock()
	defer soc.mu.Unlock()
	
	if !soc.started {
		return fmt.Errorf("streaming capture not started")
	}
	
	if soc.closed {
		return nil // Already closed
	}
	
	// Finalize format-specific streaming
	switch soc.format {
	case FormatJSON, FormatStructured:
		if err := soc.finalizeJSONStreaming(); err != nil {
			return fmt.Errorf("failed to finalize JSON streaming: %w", err)
		}
	case FormatText:
		// Text format doesn't need special finalization
	}
	
	soc.closed = true
	duration := time.Since(soc.startTime)
	
	slog.Debug("Streaming output capture closed", "session_id", soc.sessionID, "duration_ms", duration.Milliseconds())
	return nil
}

// GetContent returns empty string for streaming capture
func (soc *StreamingOutputCapture) GetContent() string {
	return "" // Streaming capture doesn't buffer content
}

// streamTextMessage streams a text message immediately
func (soc *StreamingOutputCapture) streamTextMessage(msg *message.Message) error {
	if msg.Role != message.Assistant {
		return nil // Only stream assistant messages for text format
	}
	
	content := msg.Content().String()
	
	// For streaming, we want to output only new content since last update
	if len(content) > soc.contentOffset {
		newContent := content[soc.contentOffset:]
		if _, err := soc.writer.Write([]byte(newContent)); err != nil {
			return fmt.Errorf("failed to stream text content: %w", err)
		}
		soc.contentOffset = len(content)
		
		// Flush if the writer supports it
		if flusher, ok := soc.writer.(interface{ Flush() error }); ok {
			_ = flusher.Flush()
		}
	}
	
	return nil
}

// initJSONStreaming initializes JSON streaming format
func (soc *StreamingOutputCapture) initJSONStreaming() error {
	// Start JSON object
	if _, err := soc.writer.Write([]byte("{\n")); err != nil {
		return fmt.Errorf("failed to write JSON header: %w", err)
	}
	
	// Write session ID
	if _, err := soc.writer.Write([]byte(fmt.Sprintf(`  "session_id": "%s",`, soc.sessionID))); err != nil {
		return fmt.Errorf("failed to write session ID: %w", err)
	}
	
	// Start messages array
	if _, err := soc.writer.Write([]byte("\n  \"messages\": [\n")); err != nil {
		return fmt.Errorf("failed to write messages array header: %w", err)
	}
	
	return nil
}

// streamJSONMessage streams a message in JSON format
func (soc *StreamingOutputCapture) streamJSONMessage(msg *message.Message) error {
	// Add comma before message if not the first one
	if !soc.firstMessage {
		if _, err := soc.writer.Write([]byte(",\n")); err != nil {
			return fmt.Errorf("failed to write JSON separator: %w", err)
		}
	} else {
		soc.firstMessage = false
	}
	
	// Serialize message to JSON
	msgBytes, err := json.MarshalIndent(msg, "    ", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal message to JSON: %w", err)
	}
	
	// Write indented message
	if _, err := soc.writer.Write([]byte("    ")); err != nil {
		return fmt.Errorf("failed to write JSON indentation: %w", err)
	}
	
	if _, err := soc.writer.Write(msgBytes); err != nil {
		return fmt.Errorf("failed to write JSON message: %w", err)
	}
	
	// Flush if the writer supports it
	if flusher, ok := soc.writer.(interface{ Flush() error }); ok {
		_ = flusher.Flush()
	}
	
	return nil
}

// finalizeJSONStreaming closes the JSON streaming format
func (soc *StreamingOutputCapture) finalizeJSONStreaming() error {
	// Close messages array
	if _, err := soc.writer.Write([]byte("\n  ],\n")); err != nil {
		return fmt.Errorf("failed to close messages array: %w", err)
	}
	
	// Add metadata
	metadata := ConversationMetadata{
		SessionID:  soc.sessionID,
		StartTime:  soc.startTime,
		EndTime:    time.Now(),
		DurationMs: time.Since(soc.startTime).Milliseconds(),
	}
	
	metadataBytes, err := json.MarshalIndent(metadata, "  ", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata to JSON: %w", err)
	}
	
	if _, err := soc.writer.Write([]byte("  \"metadata\": ")); err != nil {
		return fmt.Errorf("failed to write metadata header: %w", err)
	}
	
	if _, err := soc.writer.Write(metadataBytes); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}
	
	// Close JSON object
	if _, err := soc.writer.Write([]byte("\n}\n")); err != nil {
		return fmt.Errorf("failed to close JSON object: %w", err)
	}
	
	// Final flush
	if flusher, ok := soc.writer.(interface{ Flush() error }); ok {
		_ = flusher.Flush()
	}
	
	return nil
}

// StreamingWriter wraps an io.Writer to provide buffering and flushing capabilities
type StreamingWriter struct {
	writer *bufio.Writer
	closer io.Closer
}

// NewStreamingWriter creates a new streaming writer with buffering
func NewStreamingWriter(w io.Writer, bufferSize int) *StreamingWriter {
	if bufferSize <= 0 {
		bufferSize = 4096
	}
	
	return &StreamingWriter{
		writer: bufio.NewWriterSize(w, bufferSize),
		closer: nil,
	}
}

// NewStreamingFileWriter creates a streaming writer for file output
func NewStreamingFileWriter(w io.WriteCloser, bufferSize int) *StreamingWriter {
	if bufferSize <= 0 {
		bufferSize = 4096
	}
	
	return &StreamingWriter{
		writer: bufio.NewWriterSize(w, bufferSize),
		closer: w,
	}
}

// Write implements io.Writer
func (sw *StreamingWriter) Write(p []byte) (n int, err error) {
	return sw.writer.Write(p)
}

// Flush flushes the buffered data
func (sw *StreamingWriter) Flush() error {
	return sw.writer.Flush()
}

// Close flushes and closes the underlying writer if it's a closer
func (sw *StreamingWriter) Close() error {
	if err := sw.writer.Flush(); err != nil {
		return err
	}
	
	if sw.closer != nil {
		return sw.closer.Close()
	}
	
	return nil
}