package types

import (
	"time"
	
	"github.com/charmbracelet/crush/internal/output"
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
	FilterReasoning bool    // NEW: Add this field exactly here
}