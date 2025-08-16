package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/message"
)

func TestDefaultOutputCapture_StartCapture(t *testing.T) {
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
		{
			name:      "valid structured format",
			sessionID: "test-session-3",
			format:    FormatStructured,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capture := NewOutputCapture() // Create a new capture for each test
			err := capture.StartCapture(tt.sessionID, tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("StartCapture() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultOutputCapture_CaptureMessage(t *testing.T) {
	capture := NewOutputCapture()
	err := capture.StartCapture("test-session", FormatText)
	if err != nil {
		t.Fatalf("Failed to start capture: %v", err)
	}

	testMessage := &message.Message{
		ID:        "msg-1",
		SessionID: "test-session",
		Role:      message.Assistant,
		Parts: []message.ContentPart{
			message.TextContent{Text: "Hello, world!"},
		},
		Model:     "test-model",
		Provider:  "test-provider",
		CreatedAt: time.Now().Unix(),
	}

	err = capture.CaptureMessage(testMessage)
	if err != nil {
		t.Errorf("CaptureMessage() error = %v", err)
	}

	// Test capturing without starting
	capture2 := NewOutputCapture()
	err = capture2.CaptureMessage(testMessage)
	if err == nil {
		t.Error("CaptureMessage() should fail when capture not started")
	}
}

func TestDefaultOutputCapture_WriteTextOutput(t *testing.T) {
	capture := NewOutputCapture()
	err := capture.StartCapture("test-session", FormatText)
	if err != nil {
		t.Fatalf("Failed to start capture: %v", err)
	}

	// Add test message
	testMessage := &message.Message{
		ID:        "msg-1",
		SessionID: "test-session",
		Role:      message.Assistant,
		Parts: []message.ContentPart{
			message.TextContent{Text: "Hello, world!"},
		},
	}

	err = capture.CaptureMessage(testMessage)
	if err != nil {
		t.Fatalf("Failed to capture message: %v", err)
	}

	var buffer bytes.Buffer
	err = capture.WriteOutput(&buffer)
	if err != nil {
		t.Errorf("WriteOutput() error = %v", err)
	}

	expected := "Hello, world!"
	if buffer.String() != expected {
		t.Errorf("WriteOutput() = %v, want %v", buffer.String(), expected)
	}
}

func TestDefaultOutputCapture_WriteJSONOutput(t *testing.T) {
	capture := NewOutputCapture()
	err := capture.StartCapture("test-session", FormatJSON)
	if err != nil {
		t.Fatalf("Failed to start capture: %v", err)
	}

	// Add test message
	testMessage := &message.Message{
		ID:        "msg-1",
		SessionID: "test-session",
		Role:      message.Assistant,
		Parts: []message.ContentPart{
			message.TextContent{Text: "Hello, world!"},
		},
		Model:    "test-model",
		Provider: "test-provider",
	}

	err = capture.CaptureMessage(testMessage)
	if err != nil {
		t.Fatalf("Failed to capture message: %v", err)
	}

	// Add metadata
	metadata := ConversationMetadata{
		SessionID:  "test-session",
		Model:      "test-model",
		Provider:   "test-provider",
		StartTime:  time.Now(),
		DurationMs: 1000,
	}
	err = capture.CaptureMetadata(metadata)
	if err != nil {
		t.Fatalf("Failed to capture metadata: %v", err)
	}

	var buffer bytes.Buffer
	err = capture.WriteOutput(&buffer)
	if err != nil {
		t.Errorf("WriteOutput() error = %v", err)
	}

	// Verify JSON structure by checking for expected keys
	output := buffer.String()
	if !strings.Contains(output, `"session_id": "test-session"`) {
		t.Errorf("Expected JSON to contain session_id, got: %s", output)
	}

	if !strings.Contains(output, `"messages"`) {
		t.Errorf("Expected JSON to contain messages array, got: %s", output)
	}

	if !strings.Contains(output, `"metadata"`) {
		t.Errorf("Expected JSON to contain metadata, got: %s", output)
	}

	if !strings.Contains(output, `"msg-1"`) {
		t.Errorf("Expected JSON to contain message ID, got: %s", output)
	}
}

func TestDefaultOutputCapture_Close(t *testing.T) {
	capture := NewOutputCapture()
	err := capture.StartCapture("test-session", FormatText)
	if err != nil {
		t.Fatalf("Failed to start capture: %v", err)
	}

	err = capture.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Test closing twice
	err = capture.Close()
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

	err = capture.CaptureMessage(testMessage)
	if err == nil {
		t.Error("CaptureMessage() should fail after close")
	}
}

func TestDefaultOutputCapture_GetContent(t *testing.T) {
	capture := NewOutputCapture()
	err := capture.StartCapture("test-session", FormatText)
	if err != nil {
		t.Fatalf("Failed to start capture: %v", err)
	}

	// Add test message
	testMessage := &message.Message{
		ID:        "msg-1",
		SessionID: "test-session",
		Role:      message.Assistant,
		Parts: []message.ContentPart{
			message.TextContent{Text: "Hello, world!"},
		},
	}

	err = capture.CaptureMessage(testMessage)
	if err != nil {
		t.Fatalf("Failed to capture message: %v", err)
	}

	content := capture.GetContent()
	expected := "Hello, world!"
	if content != expected {
		t.Errorf("GetContent() = %v, want %v", content, expected)
	}

	// Test with non-text format
	captureJSON := NewOutputCapture()
	err = captureJSON.StartCapture("test-session", FormatJSON)
	if err != nil {
		t.Fatalf("Failed to start JSON capture: %v", err)
	}

	content = captureJSON.GetContent()
	if content != "" {
		t.Errorf("GetContent() for JSON format should return empty string, got %v", content)
	}
}

func TestOutputFormats(t *testing.T) {
	formats := []OutputFormat{FormatText, FormatJSON, FormatStructured}
	
	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			capture := NewOutputCapture()
			err := capture.StartCapture("test-session", format)
			if err != nil {
				t.Errorf("StartCapture() with format %s error = %v", format, err)
			}

			// Add test message
			testMessage := &message.Message{
				ID:        "msg-1",
				SessionID: "test-session",
				Role:      message.Assistant,
				Parts: []message.ContentPart{
					message.TextContent{Text: "Test content"},
				},
			}

			err = capture.CaptureMessage(testMessage)
			if err != nil {
				t.Errorf("CaptureMessage() with format %s error = %v", format, err)
			}

			var buffer bytes.Buffer
			err = capture.WriteOutput(&buffer)
			if err != nil {
				t.Errorf("WriteOutput() with format %s error = %v", format, err)
			}

			if buffer.Len() == 0 {
				t.Errorf("WriteOutput() with format %s produced empty output", format)
			}
		})
	}
}