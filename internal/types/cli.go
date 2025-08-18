package types

import (
	"fmt"
	"time"
	
	"github.com/charmbracelet/crush/internal/output"
	"github.com/charmbracelet/crush/internal/security"
)

// NonInteractiveParams contains parameters for enhanced non-interactive execution
type NonInteractiveParams struct {
	Prompt          string
	SessionTitle    string
	Timeout         time.Duration
	MaxTokens       int
	Model           string
	DisableTools    bool
	OutputFormat    output.OutputFormat
	OutputFile      string
	FilterReasoning bool    // true = filter reasoning (DEFAULT), false = show reasoning
}

// ValidateParams validates non-interactive parameters for security
func (nip *NonInteractiveParams) ValidateParams() error {
	config := security.DefaultSecurityConfig()
	validator, err := security.NewInputValidator(&config.Input, nil)
	if err != nil {
		return fmt.Errorf("failed to create validator: %w", err)
	}
	
	// Validate prompt input
	if nip.Prompt != "" {
		if err := validator.ValidateInput(nip.Prompt); err != nil {
			return fmt.Errorf("invalid prompt: %w", err)
		}
	}
	
	// Validate numeric parameters
	if nip.MaxTokens < 0 || nip.MaxTokens > 32768 {
		return fmt.Errorf("max_tokens must be between 0 and 32768")
	}
	
	// Validate timeout
	if nip.Timeout < 0 {
		return fmt.Errorf("timeout must be non-negative")
	}
	
	// Validate output file if specified
	if nip.OutputFile != "" {
		// Basic path validation - more comprehensive validation should be done elsewhere
		if len(nip.OutputFile) > 1024 {
			return fmt.Errorf("output file path too long")
		}
	}
	
	return nil
}