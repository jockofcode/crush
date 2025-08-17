package output

import (
	"bytes"
	"encoding/json"
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
			capture := NewOutputCapture(false) // Create a new capture for each test
			err := capture.StartCapture(tt.sessionID, tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("StartCapture() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultOutputCapture_CaptureMessage(t *testing.T) {
	capture := NewOutputCapture(false)
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
	capture2 := NewOutputCapture(false)
	err = capture2.CaptureMessage(testMessage)
	if err == nil {
		t.Error("CaptureMessage() should fail when capture not started")
	}
}

func TestDefaultOutputCapture_WriteTextOutput(t *testing.T) {
	capture := NewOutputCapture(false)
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
	capture := NewOutputCapture(false)
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
	capture := NewOutputCapture(false)
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
	capture := NewOutputCapture(false)
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
	captureJSON := NewOutputCapture(false)
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
			capture := NewOutputCapture(false)
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

func TestFilterThinkingContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple_thinking_removal",
			input:    "Hello <think>reasoning content</think> World",
			expected: "Hello  World",
		},
		{
			name:     "multiple_thinking_tags",
			input:    "Start <think>first thought</think> middle <think>second thought</think> end",
			expected: "Start  middle  end",
		},
		{
			name:     "case_insensitive_tags",
			input:    "Test <THINK>uppercase</THINK> and <Think>mixed case</Think> content",
			expected: "Test  and  content",
		},
		{
			name:     "nested_content_handling",
			input:    "Text <think>outer <inner>nested</inner> content</think> more text",
			expected: "Text  more text",
		},
		{
			name:     "no_thinking_tags",
			input:    "Just regular content without any tags",
			expected: "Just regular content without any tags",
		},
		{
			name:     "malformed_tags_preserved",
			input:    "Text <think unclosed tag and </think> unmatched closing",
			expected: "Text <think unclosed tag and </think> unmatched closing",
		},
		{
			name:     "empty_thinking_tags",
			input:    "Before <think></think> after",
			expected: "Before  after",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterThinkingContent(tt.input)
			if result != tt.expected {
				t.Errorf("filterThinkingContent() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestFilterThinkingContentSecurity(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "redos_protection_nested_tags",
			input:    strings.Repeat("<think><think>", 1000) + "content",
			expected: strings.Repeat("<think><think>", 1000) + "content", // Should return original due to protection
		},
		{
			name:     "large_content_protection",
			input:    strings.Repeat("x", 1024*1024+1), // >1MB
			expected: strings.Repeat("x", 1024*1024+1), // Should return original due to size limit
		},
		{
			name:     "excessive_tag_count",
			input:    strings.Repeat("<think>x</think>", 101), // >100 tags
			expected: strings.Repeat("<think>x</think>", 101), // Should return original due to tag limit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			result := filterThinkingContent(tt.input)
			duration := time.Since(start)

			if duration > 200*time.Millisecond {
				t.Errorf("filterThinkingContent() took too long: %v", duration)
			}

			if result != tt.expected {
				t.Errorf("filterThinkingContent() security protection failed")
			}
		})
	}
}

func TestIsValidThinkingContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "balanced_tags",
			input:    "Text <think>content</think> more text",
			expected: true,
		},
		{
			name:     "multiple_balanced_tags",
			input:    "<think>first</think> and <think>second</think>",
			expected: true,
		},
		{
			name:     "unbalanced_open",
			input:    "Text <think>content without closing",
			expected: false,
		},
		{
			name:     "unbalanced_close",
			input:    "Text content</think> without opening",
			expected: false,
		},
		{
			name:     "no_tags",
			input:    "Just regular content",
			expected: true,
		},
		{
			name:     "excessive_tags",
			input:    strings.Repeat("<think>x</think>", 101),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidThinkingContent(tt.input)
			if result != tt.expected {
				t.Errorf("isValidThinkingContent() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestReDoSProtection(t *testing.T) {
	// Test nested tag content that could cause ReDoS
	maliciousContent := strings.Repeat("<think><think>", 1000) + "content"
	
	start := time.Now()
	result := filterThinkingContent(maliciousContent)
	duration := time.Since(start)
	
	if duration > 200*time.Millisecond {
		t.Errorf("ReDoS protection failed: took %v", duration)
	}
	
	// Should return original content due to protection
	if result != maliciousContent {
		t.Error("ReDoS protection should preserve original content")
	}
}

func TestSizeLimitProtection(t *testing.T) {
	// Create content larger than 1MB
	largeContent := strings.Repeat("x", 1024*1024+1)
	
	start := time.Now()
	result := filterThinkingContent(largeContent)
	duration := time.Since(start)
	
	if duration > 10*time.Millisecond {
		t.Errorf("Size limit protection too slow: took %v", duration)
	}
	
	// Should return original content due to size limit
	if result != largeContent {
		t.Error("Size limit protection should preserve original content")
	}
}

func TestTimeoutProtection(t *testing.T) {
	// Create content that might trigger timeout
	complexContent := strings.Repeat("<think>", 200) + "content" + strings.Repeat("</think>", 200)
	
	start := time.Now()
	result := filterThinkingContent(complexContent)
	duration := time.Since(start)
	
	// Should complete within reasonable time
	if duration > 200*time.Millisecond {
		t.Errorf("Timeout protection failed: took %v", duration)
	}
	
	// With balanced tags, it should process successfully
	if !strings.Contains(result, "content") {
		t.Error("Should have processed balanced tags successfully")
	}
}

func TestTagCountLimitProtection(t *testing.T) {
	// Create content with excessive tag count (>100)
	excessiveContent := strings.Repeat("<think>x</think>", 101)
	
	start := time.Now()
	result := filterThinkingContent(excessiveContent)
	duration := time.Since(start)
	
	if duration > 10*time.Millisecond {
		t.Errorf("Tag count protection too slow: took %v", duration)
	}
	
	// Should return original content due to tag limit
	if result != excessiveContent {
		t.Error("Tag count protection should preserve original content")
	}
}

func BenchmarkFilteringPerformance(b *testing.B) {
	content := strings.Repeat("Text <think>reasoning content</think> more text.\n", 500) // ~10KB
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filtered := filterThinkingContent(content)
		_ = filtered
	}
}

func BenchmarkSecurityControlOverhead(b *testing.B) {
	content := "Simple content without thinking tags"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filtered := filterThinkingContent(content)
		_ = filtered
	}
}

func BenchmarkSizeValidation(b *testing.B) {
	content := strings.Repeat("x", 10240) // 10KB content
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valid := isValidThinkingContent(content)
		_ = valid
	}
}

func BenchmarkRegexCompilation(b *testing.B) {
	// This tests that our global regex variables don't get recompiled
	content := "Hello <think>world</think> test"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use the global regex directly
		result := thinkingTagRegex.ReplaceAllString(content, "")
		_ = result
	}
}

func BenchmarkCleanWhitespace(b *testing.B) {
	content := "Text with\n\n\n\nexcessive\n\n\nwhitespace"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cleaned := cleanWhitespace(content)
		_ = cleaned
	}
}

func BenchmarkFilteringWithMultipleTags(b *testing.B) {
	// Create content with multiple thinking tags
	content := strings.Repeat("Before <think>reasoning</think> middle <think>more reasoning</think> after.\n", 100)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filtered := filterThinkingContent(content)
		_ = filtered
	}
}

func BenchmarkValidationWithBalancedTags(b *testing.B) {
	content := strings.Repeat("<think>content</think>", 50) // 50 balanced tags
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valid := isValidThinkingContent(content)
		_ = valid
	}
}

func TestPerformanceTargets(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		maxDuration time.Duration
		description string
	}{
		{
			name:        "filtering_10kb_content",
			content:     strings.Repeat("Text <think>reasoning content</think> more text.\n", 500),
			maxDuration: 5 * time.Millisecond,
			description: "10KB content should filter within 5ms",
		},
		{
			name:        "security_control_simple",
			content:     "Simple content without thinking tags",
			maxDuration: 100 * time.Microsecond,
			description: "Security controls should add <0.1ms overhead",
		},
		{
			name:        "size_validation_10kb",
			content:     strings.Repeat("x", 10240),
			maxDuration: time.Microsecond,
			description: "Size validation should complete in <0.001ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			
			switch tt.name {
			case "filtering_10kb_content", "security_control_simple":
				_ = filterThinkingContent(tt.content)
			case "size_validation_10kb":
				_ = isValidThinkingContent(tt.content)
			}
			
			duration := time.Since(start)
			
			if duration > tt.maxDuration {
				t.Errorf("%s: took %v, expected <%v", tt.description, duration, tt.maxDuration)
			} else {
				t.Logf("%s: ✅ %v (target: <%v)", tt.description, duration, tt.maxDuration)
			}
		})
	}
}

func TestOutputCaptureWithFiltering(t *testing.T) {
	tests := []struct {
		name            string
		filterReasoning bool
		input           string
		expectedText    string
		expectedJSON    string
	}{
		{
			name:            "filtering_enabled",
			filterReasoning: true,
			input:           "Hello <think>private reasoning</think> World",
			expectedText:    "Hello  World",
			expectedJSON:    "Hello  World", // JSON contains the filtered content
		},
		{
			name:            "filtering_disabled",
			filterReasoning: false,
			input:           "Hello <think>private reasoning</think> World",
			expectedText:    "Hello <think>private reasoning</think> World",
			expectedJSON:    "Hello <think>private reasoning</think> World",
		},
		{
			name:            "no_thinking_tags_filtered",
			filterReasoning: true,
			input:           "Simple content without tags",
			expectedText:    "Simple content without tags",
			expectedJSON:    "Simple content without tags",
		},
		{
			name:            "no_thinking_tags_unfiltered",
			filterReasoning: false,
			input:           "Simple content without tags",
			expectedText:    "Simple content without tags",
			expectedJSON:    "Simple content without tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Text Format
			t.Run("text_format", func(t *testing.T) {
				capture := NewOutputCapture(tt.filterReasoning)
				err := capture.StartCapture("test-session", FormatText)
				if err != nil {
					t.Fatalf("StartCapture() error = %v", err)
				}

				// Add test message
				testMessage := &message.Message{
					ID:        "msg-1",
					SessionID: "test-session",
					Role:      message.Assistant,
					Parts: []message.ContentPart{
						message.TextContent{Text: tt.input},
					},
				}

				err = capture.CaptureMessage(testMessage)
				if err != nil {
					t.Fatalf("CaptureMessage() error = %v", err)
				}

				var buffer bytes.Buffer
				err = capture.WriteOutput(&buffer)
				if err != nil {
					t.Fatalf("WriteOutput() error = %v", err)
				}

				if buffer.String() != tt.expectedText {
					t.Errorf("Text output = %q, expected %q", buffer.String(), tt.expectedText)
				}
			})

			// Test JSON Format
			t.Run("json_format", func(t *testing.T) {
				capture := NewOutputCapture(tt.filterReasoning)
				err := capture.StartCapture("test-session", FormatJSON)
				if err != nil {
					t.Fatalf("StartCapture() error = %v", err)
				}

				// Add test message
				testMessage := &message.Message{
					ID:        "msg-1",
					SessionID: "test-session",
					Role:      message.Assistant,
					Parts: []message.ContentPart{
						message.TextContent{Text: tt.input},
					},
				}

				err = capture.CaptureMessage(testMessage)
				if err != nil {
					t.Fatalf("CaptureMessage() error = %v", err)
				}

				var buffer bytes.Buffer
				err = capture.WriteOutput(&buffer)
				if err != nil {
					t.Fatalf("WriteOutput() error = %v", err)
				}

				// Check that JSON is valid and contains expected content
				var jsonData map[string]interface{}
				if err := json.Unmarshal(buffer.Bytes(), &jsonData); err != nil {
					t.Fatalf("JSON unmarshal error = %v", err)
				}

				// Check that JSON contains the expected content (accounting for JSON escaping)
				jsonStr := buffer.String()
				// JSON escapes < and > characters
				expectedJSONEscaped := strings.ReplaceAll(tt.expectedJSON, "<", "\\u003c")
				expectedJSONEscaped = strings.ReplaceAll(expectedJSONEscaped, ">", "\\u003e")
				
				if !strings.Contains(jsonStr, expectedJSONEscaped) {
					t.Errorf("JSON output does not contain expected content %q (escaped: %q)\nActual output: %s", tt.expectedJSON, expectedJSONEscaped, jsonStr)
				}
			})

			// Test Structured Format
			t.Run("structured_format", func(t *testing.T) {
				capture := NewOutputCapture(tt.filterReasoning)
				err := capture.StartCapture("test-session", FormatStructured)
				if err != nil {
					t.Fatalf("StartCapture() error = %v", err)
				}

				// Add test message
				testMessage := &message.Message{
					ID:        "msg-1",
					SessionID: "test-session",
					Role:      message.Assistant,
					Parts: []message.ContentPart{
						message.TextContent{Text: tt.input},
					},
				}

				err = capture.CaptureMessage(testMessage)
				if err != nil {
					t.Fatalf("CaptureMessage() error = %v", err)
				}

				var buffer bytes.Buffer
				err = capture.WriteOutput(&buffer)
				if err != nil {
					t.Fatalf("WriteOutput() error = %v", err)
				}

				// Check that structured format (JSON) is valid and contains expected content
				var jsonData map[string]interface{}
				if err := json.Unmarshal(buffer.Bytes(), &jsonData); err != nil {
					t.Fatalf("JSON unmarshal error = %v", err)
				}

				// Check that structured output contains the expected content (accounting for JSON escaping)
				jsonStr := buffer.String()
				// JSON escapes < and > characters
				expectedJSONEscaped := strings.ReplaceAll(tt.expectedJSON, "<", "\\u003c")
				expectedJSONEscaped = strings.ReplaceAll(expectedJSONEscaped, ">", "\\u003e")
				
				if !strings.Contains(jsonStr, expectedJSONEscaped) {
					t.Errorf("Structured output does not contain expected content %q (escaped: %q)\nActual output: %s", tt.expectedJSON, expectedJSONEscaped, jsonStr)
				}
			})
		})
	}
}

func TestBackwardCompatibility(t *testing.T) {
	// Test that filtering is disabled by default (backward compatibility)
	capture := NewOutputCapture(false)
	err := capture.StartCapture("test-session", FormatText)
	if err != nil {
		t.Fatalf("StartCapture() error = %v", err)
	}

	// Add message with thinking content
	testMessage := &message.Message{
		ID:        "msg-1",
		SessionID: "test-session",
		Role:      message.Assistant,
		Parts: []message.ContentPart{
			message.TextContent{Text: "Hello <think>this should be preserved</think> World"},
		},
	}

	err = capture.CaptureMessage(testMessage)
	if err != nil {
		t.Fatalf("CaptureMessage() error = %v", err)
	}

	var buffer bytes.Buffer
	err = capture.WriteOutput(&buffer)
	if err != nil {
		t.Fatalf("WriteOutput() error = %v", err)
	}

	expected := "Hello <think>this should be preserved</think> World"
	if buffer.String() != expected {
		t.Errorf("Backward compatibility failed: got %q, expected %q", buffer.String(), expected)
	}
}

func TestGetContentWithFiltering(t *testing.T) {
	tests := []struct {
		name            string
		filterReasoning bool
		input           string
		expected        string
	}{
		{
			name:            "with_filtering",
			filterReasoning: true,
			input:           "Content <think>reasoning</think> more content",
			expected:        "Content  more content",
		},
		{
			name:            "without_filtering",
			filterReasoning: false,
			input:           "Content <think>reasoning</think> more content",
			expected:        "Content <think>reasoning</think> more content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capture := NewOutputCapture(tt.filterReasoning)
			err := capture.StartCapture("test-session", FormatText)
			if err != nil {
				t.Fatalf("StartCapture() error = %v", err)
			}

			// Add test message
			testMessage := &message.Message{
				ID:        "msg-1",
				SessionID: "test-session",
				Role:      message.Assistant,
				Parts: []message.ContentPart{
					message.TextContent{Text: tt.input},
				},
			}

			err = capture.CaptureMessage(testMessage)
			if err != nil {
				t.Fatalf("CaptureMessage() error = %v", err)
			}

			content := capture.GetContent()
			if content != tt.expected {
				t.Errorf("GetContent() = %q, expected %q", content, tt.expected)
			}
		})
	}
}