package types

import (
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/output"
)

func TestNonInteractiveParams_Fields(t *testing.T) {
	params := NonInteractiveParams{
		Prompt:       "test prompt",
		SessionTitle: "test session",
		Timeout:      30 * time.Second,
		MaxTokens:    1000,
		Model:        "claude-3-sonnet",
		DisableTools: true,
		OutputFormat: output.FormatJSON,
		OutputFile:   "output.json",
	}

	// Test that all fields are properly set
	if params.Prompt != "test prompt" {
		t.Errorf("Expected Prompt = 'test prompt', got %v", params.Prompt)
	}

	if params.SessionTitle != "test session" {
		t.Errorf("Expected SessionTitle = 'test session', got %v", params.SessionTitle)
	}

	if params.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout = 30s, got %v", params.Timeout)
	}

	if params.MaxTokens != 1000 {
		t.Errorf("Expected MaxTokens = 1000, got %v", params.MaxTokens)
	}

	if params.Model != "claude-3-sonnet" {
		t.Errorf("Expected Model = 'claude-3-sonnet', got %v", params.Model)
	}

	if !params.DisableTools {
		t.Errorf("Expected DisableTools = true, got %v", params.DisableTools)
	}

	if params.OutputFormat != output.FormatJSON {
		t.Errorf("Expected OutputFormat = FormatJSON, got %v", params.OutputFormat)
	}

	if params.OutputFile != "output.json" {
		t.Errorf("Expected OutputFile = 'output.json', got %v", params.OutputFile)
	}
}

func TestNonInteractiveParams_DefaultValues(t *testing.T) {
	var params NonInteractiveParams

	// Test zero values
	if params.Prompt != "" {
		t.Errorf("Expected default Prompt to be empty, got %v", params.Prompt)
	}

	if params.SessionTitle != "" {
		t.Errorf("Expected default SessionTitle to be empty, got %v", params.SessionTitle)
	}

	if params.Timeout != 0 {
		t.Errorf("Expected default Timeout to be 0, got %v", params.Timeout)
	}

	if params.MaxTokens != 0 {
		t.Errorf("Expected default MaxTokens to be 0, got %v", params.MaxTokens)
	}

	if params.Model != "" {
		t.Errorf("Expected default Model to be empty, got %v", params.Model)
	}

	if params.DisableTools {
		t.Errorf("Expected default DisableTools to be false, got %v", params.DisableTools)
	}

	if params.OutputFormat != "" {
		t.Errorf("Expected default OutputFormat to be empty, got %v", params.OutputFormat)
	}

	if params.OutputFile != "" {
		t.Errorf("Expected default OutputFile to be empty, got %v", params.OutputFile)
	}
}