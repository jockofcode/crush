package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// InputProcessor handles prompt input processing with environment variable expansion
type InputProcessor struct {
	maxPromptSize  int64
	allowedEnvVars []string
}

// PromptParams contains parameters for prompt processing
type PromptParams struct {
	DirectPrompt string
	PromptFile   string
	Args         []string
}

// NewInputProcessor creates a new input processor with default settings
func NewInputProcessor() *InputProcessor {
	return &InputProcessor{
		maxPromptSize:  1024 * 1024, // 1MB max prompt size
		allowedEnvVars: []string{},   // Empty means allow all
	}
}

// ProcessPrompt processes prompt input from various sources and combines them
func (ip *InputProcessor) ProcessPrompt(params PromptParams) (string, error) {
	var prompts []string

	// Process direct prompt parameter
	if params.DirectPrompt != "" {
		expandedPrompt, err := ip.ExpandEnvironmentVars(params.DirectPrompt)
		if err != nil {
			return "", fmt.Errorf("failed to expand environment variables in prompt: %w", err)
		}
		prompts = append(prompts, expandedPrompt)
	}

	// Process prompt file
	if params.PromptFile != "" {
		fileContent, err := ip.ReadPromptFile(params.PromptFile)
		if err != nil {
			return "", fmt.Errorf("failed to read prompt file: %w", err)
		}
		expandedContent, err := ip.ExpandEnvironmentVars(fileContent)
		if err != nil {
			return "", fmt.Errorf("failed to expand environment variables in prompt file: %w", err)
		}
		prompts = append(prompts, expandedContent)
	}

	// Process command line arguments (for backward compatibility)
	if len(params.Args) > 0 {
		argsPrompt := strings.Join(params.Args, " ")
		expandedArgs, err := ip.ExpandEnvironmentVars(argsPrompt)
		if err != nil {
			return "", fmt.Errorf("failed to expand environment variables in arguments: %w", err)
		}
		prompts = append(prompts, expandedArgs)
	}

	// Combine all prompts
	finalPrompt := strings.Join(prompts, "\n\n")

	// Validate the final prompt
	if err := ip.ValidateInput(finalPrompt); err != nil {
		return "", err
	}

	return finalPrompt, nil
}

// ExpandEnvironmentVars expands environment variables in the format ${VAR} or $VAR
func (ip *InputProcessor) ExpandEnvironmentVars(text string) (string, error) {
	// Regex to match ${VAR} and $VAR patterns
	envVarRegex := regexp.MustCompile(`\$\{([^}]+)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)
	
	result := envVarRegex.ReplaceAllStringFunc(text, func(match string) string {
		var varName string
		
		// Extract variable name from either ${VAR} or $VAR format
		if strings.HasPrefix(match, "${") {
			varName = match[2 : len(match)-1] // Remove ${ and }
		} else {
			varName = match[1:] // Remove $
		}
		
		// Check if variable is allowed (if allowlist is configured)
		if len(ip.allowedEnvVars) > 0 && !ip.isVarAllowed(varName) {
			return match // Return original if not allowed
		}
		
		// Get environment variable value
		value := os.Getenv(varName)
		if value == "" {
			return match // Return original if variable not set
		}
		
		return value
	})
	
	return result, nil
}

// ReadPromptFile reads a prompt from a file with proper validation
func (ip *InputProcessor) ReadPromptFile(path string) (string, error) {
	// Clean and validate the file path
	cleanPath := filepath.Clean(path)
	
	// Security check: prevent path traversal
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("invalid file path: %w", err)
	}
	
	// Check if file exists and is readable
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("cannot access prompt file %s: %w", path, err)
	}
	
	// Check file size
	if info.Size() > ip.maxPromptSize {
		return "", fmt.Errorf("prompt file %s is too large (%d bytes, max %d bytes)", 
			path, info.Size(), ip.maxPromptSize)
	}
	
	// Read file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file %s: %w", path, err)
	}
	
	return string(content), nil
}

// ValidateInput validates the processed prompt input
func (ip *InputProcessor) ValidateInput(prompt string) error {
	if len(prompt) == 0 {
		return fmt.Errorf("prompt cannot be empty")
	}
	
	if int64(len(prompt)) > ip.maxPromptSize {
		return fmt.Errorf("prompt is too large (%d characters, max %d)", 
			len(prompt), ip.maxPromptSize)
	}
	
	// Check for potentially dangerous content
	if strings.Contains(prompt, "\x00") {
		return fmt.Errorf("prompt contains null bytes")
	}
	
	return nil
}

// isVarAllowed checks if an environment variable is in the allowed list
func (ip *InputProcessor) isVarAllowed(varName string) bool {
	for _, allowed := range ip.allowedEnvVars {
		if allowed == varName {
			return true
		}
	}
	return false
}

// SetAllowedEnvVars sets the list of allowed environment variables
func (ip *InputProcessor) SetAllowedEnvVars(vars []string) {
	ip.allowedEnvVars = vars
}

// SetMaxPromptSize sets the maximum allowed prompt size
func (ip *InputProcessor) SetMaxPromptSize(size int64) {
	ip.maxPromptSize = size
}