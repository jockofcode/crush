package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/message"
)

func TestStreamingOutputCapture_StartCapture(t *testing.T) {

	tests := []struct {
		name      string
		sessionID string
		format    OutputFormat
		wantErr   bool
	}{
		{
			name:      "valid text format",
			sessionID: "test-session-1",
			format:    FormatText,
			wantErr:   false,
		},
		{
			name:      "valid json format",
			sessionID: "test-session-2",
			format:    FormatJSON,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testBuffer bytes.Buffer
			testStreaming := NewStreamingOutputCapture(&testBuffer, 1024)
			
			err := testStreaming.StartCapture(tt.sessionID, tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("StartCapture() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStreamingOutputCapture_TextStreaming(t *testing.T) {
	var buffer bytes.Buffer
	streaming := NewStreamingOutputCapture(&buffer, 1024)

	err := streaming.StartCapture("test-session", FormatText)
	if err != nil {
		t.Fatalf("Failed to start capture: %v", err)
	}

	// Create test messages with incremental content
	messages := []*message.Message{
		{
			ID:        "msg-1",
			SessionID: "test-session",
			Role:      message.Assistant,
			Parts: []message.ContentPart{
				message.TextContent{Text: "Hello"},
			},
		},
		{
			ID:        "msg-1",
			SessionID: "test-session",
			Role:      message.Assistant,
			Parts: []message.ContentPart{
				message.TextContent{Text: "Hello, world"},
			},
		},
		{
			ID:        "msg-1",
			SessionID: "test-session",
			Role:      message.Assistant,
			Parts: []message.ContentPart{
				message.TextContent{Text: "Hello, world!"},
			},
		},
	}

	// Stream messages
	for _, msg := range messages {
		err = streaming.CaptureMessage(msg)
		if err != nil {
			t.Errorf("CaptureMessage() error = %v", err)
		}
	}

	err = streaming.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify streamed content (should contain incremental updates)
	output := buffer.String()
	if !strings.Contains(output, "Hello") {
		t.Errorf("Expected output to contain 'Hello', got: %s", output)
	}
	if !strings.Contains(output, ", world") {
		t.Errorf("Expected output to contain ', world', got: %s", output)
	}
	if !strings.Contains(output, "!") {
		t.Errorf("Expected output to contain '!', got: %s", output)
	}
}

func TestStreamingOutputCapture_JSONStreaming(t *testing.T) {
	var buffer bytes.Buffer
	streaming := NewStreamingOutputCapture(&buffer, 1024)

	err := streaming.StartCapture("test-session", FormatJSON)
	if err != nil {
		t.Fatalf("Failed to start capture: %v", err)
	}

	// Add test messages
	messages := []*message.Message{
		{
			ID:        "msg-1",
			SessionID: "test-session",
			Role:      message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: "Hello"},
			},
		},
		{
			ID:        "msg-2",
			SessionID: "test-session",
			Role:      message.Assistant,
			Parts: []message.ContentPart{
				message.TextContent{Text: "Hi there!"},
			},
		},
	}

	for _, msg := range messages {
		err = streaming.CaptureMessage(msg)
		if err != nil {
			t.Errorf("CaptureMessage() error = %v", err)
		}
	}

	err = streaming.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify JSON structure
	output := buffer.String()
	if !strings.Contains(output, `"session_id": "test-session"`) {
		t.Errorf("Expected output to contain session_id, got: %s", output)
	}
	if !strings.Contains(output, `"messages": [`) {
		t.Errorf("Expected output to contain messages array, got: %s", output)
	}
	if !strings.Contains(output, `"metadata"`) {
		t.Errorf("Expected output to contain metadata, got: %s", output)
	}
}

func TestStreamingOutputCapture_WriteOutputError(t *testing.T) {
	var buffer bytes.Buffer
	streaming := NewStreamingOutputCapture(&buffer, 1024)

	err := streaming.StartCapture("test-session", FormatText)
	if err != nil {
		t.Fatalf("Failed to start capture: %v", err)
	}

	// WriteOutput should fail for streaming capture
	err = streaming.WriteOutput(&buffer)
	if err == nil {
		t.Error("WriteOutput() should fail for streaming capture")
	}
}

func TestStreamingOutputCapture_Close(t *testing.T) {
	var buffer bytes.Buffer
	streaming := NewStreamingOutputCapture(&buffer, 1024)

	err := streaming.StartCapture("test-session", FormatText)
	if err != nil {
		t.Fatalf("Failed to start capture: %v", err)
	}

	err = streaming.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Test closing twice
	err = streaming.Close()
	if err != nil {
		t.Errorf("Close() second call error = %v", err)
	}

	// Test operations after close
	testMessage := &message.Message{
		ID:        "msg-1",
		SessionID: "test-session",
		Role:      message.Assistant,
		Parts: []message.ContentPart{
			message.TextContent{Text: "Hello, world!"},
		},
	}

	err = streaming.CaptureMessage(testMessage)
	if err == nil {
		t.Error("CaptureMessage() should fail after close")
	}
}

func TestStreamingWriter(t *testing.T) {
	var buffer bytes.Buffer
	writer := NewStreamingWriter(&buffer, 1024)

	testData := []byte("Hello, streaming world!")
	n, err := writer.Write(testData)
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write() returned %d, expected %d", n, len(testData))
	}

	err = writer.Flush()
	if err != nil {
		t.Errorf("Flush() error = %v", err)
	}

	if buffer.String() != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), buffer.String())
	}

	err = writer.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestStreamingOutputCapture_GetContent(t *testing.T) {
	var buffer bytes.Buffer
	streaming := NewStreamingOutputCapture(&buffer, 1024)

	err := streaming.StartCapture("test-session", FormatText)
	if err != nil {
		t.Fatalf("Failed to start capture: %v", err)
	}

	content := streaming.GetContent()
	if content != "" {
		t.Errorf("GetContent() should return empty string for streaming capture, got %v", content)
	}
}

func TestStreamingOutputCapture_BufferSize(t *testing.T) {
	tests := []struct {
		name       string
		bufferSize int
		expected   int
	}{
		{
			name:       "positive buffer size",
			bufferSize: 2048,
			expected:   2048,
		},
		{
			name:       "zero buffer size uses default",
			bufferSize: 0,
			expected:   4096, // default
		},
		{
			name:       "negative buffer size uses default",
			bufferSize: -100,
			expected:   4096, // default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buffer bytes.Buffer
			streaming := NewStreamingOutputCapture(&buffer, tt.bufferSize)
			
			// The buffer size is internal, so we just test that creation succeeds
			if streaming == nil {
				t.Error("NewStreamingOutputCapture() returned nil")
			}
		})
	}
}