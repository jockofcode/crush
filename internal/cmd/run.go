package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/charmbracelet/crush/internal/output"
	"github.com/charmbracelet/crush/internal/types"
)

var runCmd = &cobra.Command{
	Use:   "run [prompt...]",
	Short: "Run a single non-interactive prompt",
	Long: `Run a single prompt in non-interactive mode and exit.
The prompt can be provided as arguments, via flags, from files, or piped from stdin.
Output can be captured in multiple formats for automation and scripting.

The --no-reasoning flag can be used to remove AI reasoning content 
(content within <think>...</think> tags) from the output. This is
useful for automation and when you only need the final response.`,
	Example: `
# Run a simple prompt
crush run Explain the use of context in Go

# Direct prompt parameter
crush -p "Find all files that test for incomplete blit renders"

# Prompt from file
crush -f /path/to/prompt.txt

# Parameterized prompts with environment variables
crush -p "Analyze ${TARGET_FILE} for ${ISSUE_TYPE} issues"

# Output to file in JSON format
crush -p "Analyze this codebase" -o analysis.json --format json

# Pipe to processing script
crush -p "Find bugs" --format json | jq '.messages[].content'

# Complex automation command
crush -p "Review ${FILE}" --format json --output review.json --timeout 300 --max-tokens 2000 --model claude-3-opus --session-title "Automated Review"

# Pipe input from stdin
echo "What is this code doing?" | crush run

# Run with quiet mode (no spinner)
crush run -q "Generate a README for this project"

# Remove reasoning content from output
crush run "explain this code" --no-reasoning
crush run "write a function" --output result.txt --no-reasoning
crush run "analyze logs" --format json --no-reasoning
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flag values
		quiet, _ := cmd.Flags().GetBool("quiet")
		directPrompt, _ := cmd.Flags().GetString("prompt")
		promptFile, _ := cmd.Flags().GetString("prompt-file")
		outputFile, _ := cmd.Flags().GetString("output")
		formatStr, _ := cmd.Flags().GetString("format")
		timeoutSecs, _ := cmd.Flags().GetInt("timeout")
		maxTokens, _ := cmd.Flags().GetInt("max-tokens")
		modelOverride, _ := cmd.Flags().GetString("model")
		noTools, _ := cmd.Flags().GetBool("no-tools")
		sessionTitle, _ := cmd.Flags().GetString("session-title")
		
		noReasoning, err := cmd.Flags().GetBool("no-reasoning")
		if err != nil {
			return fmt.Errorf("failed to get no-reasoning flag: %w", err)
		}
		slog.Info("CLI flag parsing", "no-reasoning", noReasoning)

		// Fast validation of parameters BEFORE expensive app setup
		var format output.OutputFormat
		switch formatStr {
		case "text":
			format = output.FormatText
		case "json":
			format = output.FormatJSON
		case "structured":
			format = output.FormatStructured
		default:
			return fmt.Errorf("invalid output format: %s (supported: text, json, structured)", formatStr)
		}

		// Validate output file path early to fail fast
		if outputFile != "" {
			if err := validateOutputPath(outputFile); err != nil {
				return err
			}
		}

		// Validate mutual exclusion of prompt flags
		promptSources := 0
		if directPrompt != "" {
			promptSources++
		}
		if promptFile != "" {
			promptSources++
		}
		if len(args) > 0 {
			promptSources++
		}
		
		if promptSources > 1 {
			return fmt.Errorf("cannot specify multiple prompt sources: use only one of --prompt, --prompt-file, or arguments")
		}

		app, err := setupApp(cmd)
		if err != nil {
			return err
		}
		defer app.Shutdown()

		if !app.Config().IsConfigured() {
			return fmt.Errorf("no providers configured - please run 'crush' to set up a provider interactively")
		}

		// Check for stdin input first if no explicit input is provided
		var prompt string
		if directPrompt == "" && promptFile == "" && len(args) == 0 {
			// Try to read from stdin when no other input is provided
			stdinPrompt, err := MaybePrependStdin("")
			if err != nil {
				slog.Error("Failed to read from stdin", "error", err)
				return err
			}
			prompt = stdinPrompt
		} else {
			// Process prompt input using InputProcessor
			inputProcessor := NewInputProcessor()
			// Configure security controls - only allow safe environment variables
			inputProcessor.SetAllowedEnvVars([]string{
				"HOME", "USER", "PATH", "PWD", "TMPDIR", "TEMP",
				"SHELL", "LANG", "LC_ALL", "TZ", "TERM",
			})
			promptParams := PromptParams{
				DirectPrompt: directPrompt,
				PromptFile:   promptFile,
				Args:         args,
			}

			prompt, err = inputProcessor.ProcessPrompt(promptParams)
			if err != nil {
				return fmt.Errorf("failed to process prompt: %w", err)
			}

			// Handle stdin input for backward compatibility (when args are provided but prompt is empty)
			if prompt == "" {
				stdinPrompt := strings.Join(args, " ")
				stdinPrompt, err = MaybePrependStdin(stdinPrompt)
				if err != nil {
					slog.Error("Failed to read from stdin", "error", err)
					return err
				}
				prompt = stdinPrompt
			}
		}

		if prompt == "" {
			return fmt.Errorf("no prompt provided - use -p, -f, arguments, or pipe from stdin")
		}

		// Prepare timeout
		var timeout time.Duration
		if timeoutSecs > 0 {
			timeout = time.Duration(timeoutSecs) * time.Second
		}

		// Use enhanced non-interactive method with new parameters
		params := types.NonInteractiveParams{
			Prompt:          prompt,
			SessionTitle:    sessionTitle,
			Timeout:         timeout,
			MaxTokens:       maxTokens,
			Model:           modelOverride,
			DisableTools:    noTools,
			OutputFormat:    format,
			OutputFile:      outputFile,
			FilterReasoning: noReasoning,    // NEW
		}

		// Try to use the enhanced method (implemented in this enhancement)
		return app.RunNonInteractiveWithCapture(cmd.Context(), params, quiet)
	},
}

func init() {
	// Existing flags
	runCmd.Flags().BoolP("quiet", "q", false, "Hide spinner")
	
	// Prompt input flags
	runCmd.Flags().StringP("prompt", "p", "", "Direct prompt parameter")
	runCmd.Flags().StringP("prompt-file", "f", "", "Read prompt from file")
	
	// Output control flags
	runCmd.Flags().StringP("output", "o", "", "Output file path (default: stdout)")
	runCmd.Flags().String("format", "text", "Output format: text, json, structured")
	runCmd.Flags().Bool("no-reasoning", false, "Remove AI reasoning content from output")
	
	// Execution control flags
	runCmd.Flags().Int("timeout", 0, "Response timeout in seconds (0 = no timeout)")
	runCmd.Flags().Int("max-tokens", 0, "Maximum response tokens (0 = no limit)")
	runCmd.Flags().String("model", "", "Override default model")
	runCmd.Flags().Bool("no-tools", false, "Disable tool usage")
	runCmd.Flags().String("session-title", "", "Custom session title")
}

// validateOutputPath performs fast validation of output file path
func validateOutputPath(outputPath string) error {
	// Clean the path
	cleanPath := filepath.Clean(outputPath)
	
	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed in output path: %s", outputPath)
	}
	
	// Check if parent directory exists (without creating the file)
	dir := filepath.Dir(cleanPath)
	if dir != "." && dir != "/" {
		// Convert to absolute path for checking
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("invalid output path: %s", outputPath)
		}
		
		// Check if directory exists
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			return fmt.Errorf("output directory does not exist: %s", dir)
		}
	}
	
	// Check file extension (allow common text formats)
	ext := strings.ToLower(filepath.Ext(cleanPath))
	allowedExts := []string{".txt", ".json", ".md", ".log", ".out", ".yaml", ".yml"}
	
	if ext != "" {
		for _, allowed := range allowedExts {
			if ext == allowed {
				return nil // Valid extension found
			}
		}
		return fmt.Errorf("file extension %s not allowed", ext)
	}
	
	return nil
}
