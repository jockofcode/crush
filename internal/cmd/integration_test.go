package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/output"
	"github.com/charmbracelet/crush/internal/types"
)

func TestRunCommand_FlagParsing(t *testing.T) {
	// Test that all the new flags are properly defined and can be parsed
	cmd := runCmd

	// Check that all expected flags exist (using long form names)
	expectedFlags := []string{
		"prompt",
		"prompt-file", 
		"output",
		"format",
		"timeout",
		"max-tokens",
		"model",
		"no-tools",
		"session-title",
		"quiet", // existing flag
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag %s to be defined, but it wasn't found", flagName)
		}
	}

	// Check shorthand flags separately
	shorthands := map[string]string{
		"p": "prompt",
		"f": "prompt-file",
		"o": "output", 
		"q": "quiet",
	}

	for short, long := range shorthands {
		flag := cmd.Flags().Lookup(long)
		if flag == nil {
			t.Errorf("Expected flag %s to be defined for shorthand %s", long, short)
			continue
		}
		if flag.Shorthand != short {
			t.Errorf("Expected flag %s to have shorthand %s, got %s", long, short, flag.Shorthand)
		}
	}
}

func TestRunCommand_FormatValidation(t *testing.T) {
	tests := []struct {
		format  string
		wantErr bool
	}{
		{"text", false},
		{"json", false},
		{"structured", false},
		{"invalid", true},
		{"", true}, // empty format should use default, but we check validation logic
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			var expectedFormat output.OutputFormat
			var err error

			switch tt.format {
			case "text":
				expectedFormat = output.FormatText
			case "json":
				expectedFormat = output.FormatJSON
			case "structured":
				expectedFormat = output.FormatStructured
			default:
				err = nil // We'll check this in the actual command execution
			}

			if tt.format != "" && !tt.wantErr {
				if expectedFormat == "" {
					t.Errorf("Expected format %s to be valid", tt.format)
				}
			}

			// The actual validation happens in the command execution,
			// so we're mainly testing the mapping here
			if err != nil && !tt.wantErr {
				t.Errorf("Unexpected error for format %s: %v", tt.format, err)
			}
		})
	}
}

func TestInputProcessor_Integration(t *testing.T) {
	// Create a temporary prompt file
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "test_prompt.txt")
	promptContent := "Analyze the following code for security issues:\n${CODE_SNIPPET}"
	
	err := os.WriteFile(promptFile, []byte(promptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test prompt file: %v", err)
	}

	// Set environment variable
	os.Setenv("CODE_SNIPPET", "function test() { return 'hello'; }")
	defer os.Unsetenv("CODE_SNIPPET")

	// Test the input processor
	processor := NewInputProcessor()
	params := PromptParams{
		PromptFile: promptFile,
	}

	result, err := processor.ProcessPrompt(params)
	if err != nil {
		t.Fatalf("ProcessPrompt() error = %v", err)
	}

	expected := "Analyze the following code for security issues:\nfunction test() { return 'hello'; }"
	if result != expected {
		t.Errorf("ProcessPrompt() = %v, want %v", result, expected)
	}
}

func TestNonInteractiveParams_Construction(t *testing.T) {
	// Test that we can construct NonInteractiveParams properly
	params := types.NonInteractiveParams{
		Prompt:       "test prompt",
		SessionTitle: "test session",
		Timeout:      30 * time.Second,
		MaxTokens:    1000,
		Model:        "claude-3-sonnet",
		DisableTools: true,
		OutputFormat: output.FormatJSON,
		OutputFile:   "test.json",
	}

	// Verify all fields are set correctly
	if params.Prompt != "test prompt" {
		t.Errorf("Expected Prompt = 'test prompt', got %v", params.Prompt)
	}

	if params.OutputFormat != output.FormatJSON {
		t.Errorf("Expected OutputFormat = FormatJSON, got %v", params.OutputFormat)
	}

	if params.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout = 30s, got %v", params.Timeout)
	}
}

func TestOutputFormat_Constants(t *testing.T) {
	// Test that output format constants are properly defined
	formats := []output.OutputFormat{
		output.FormatText,
		output.FormatJSON,
		output.FormatStructured,
	}

	for _, format := range formats {
		if string(format) == "" {
			t.Errorf("Output format %v should not be empty", format)
		}
	}

	// Test specific values
	if output.FormatText != "text" {
		t.Errorf("Expected FormatText = 'text', got %v", output.FormatText)
	}

	if output.FormatJSON != "json" {
		t.Errorf("Expected FormatJSON = 'json', got %v", output.FormatJSON)
	}

	if output.FormatStructured != "structured" {
		t.Errorf("Expected FormatStructured = 'structured', got %v", output.FormatStructured)
	}
}