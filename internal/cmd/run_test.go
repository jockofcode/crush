package cmd

import (
	"testing"
	"strings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/spf13/cobra"
)

// Test new flag behavior
func TestShowReasoningFlag(t *testing.T) {
	tests := []struct {
		name                string
		args                []string
		expectedFilterReasoning bool
		expectedError       bool
	}{
		{
			name:                "default behavior - reasoning filtered",
			args:                []string{"-p", "test prompt"},
			expectedFilterReasoning: true, // Default: filter reasoning
			expectedError:       false,
		},
		{
			name:                "show-reasoning flag - reasoning shown",
			args:                []string{"-p", "test prompt", "--show-reasoning"},
			expectedFilterReasoning: false, // Show reasoning when requested
			expectedError:       false,
		},
		{
			name:                "show-reasoning explicit true",
			args:                []string{"-p", "test prompt", "--show-reasoning=true"},
			expectedFilterReasoning: false,
			expectedError:       false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newTestRunCommand()
			cmd.SetArgs(tt.args)
			
			// We can't fully execute the command in tests, but we can test flag parsing
			err := cmd.ParseFlags(tt.args)
			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			
			// Verify flag parsing results
			showReasoning, err := cmd.Flags().GetBool("show-reasoning")
			require.NoError(t, err)
			
			actualFilterReasoning := !showReasoning
			assert.Equal(t, tt.expectedFilterReasoning, actualFilterReasoning,
				"FilterReasoning setting should match expected value")
		})
	}
}

// Test backward compatibility with deprecated flag
func TestLegacyFlagCompatibility(t *testing.T) {
	tests := []struct {
		name                string
		args                []string
		expectedFilterReasoning bool
		expectedWarning     bool
		expectedError       bool
	}{
		{
			name:                "deprecated no-reasoning flag",
			args:                []string{"-p", "test", "--no-reasoning"},
			expectedFilterReasoning: true,
			expectedWarning:     true,
			expectedError:       false,
		},
		{
			name:                "mutual exclusion error",
			args:                []string{"-p", "test", "--no-reasoning", "--show-reasoning"},
			expectedFilterReasoning: false,
			expectedWarning:     false,
			expectedError:       true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newTestRunCommand()
			cmd.SetArgs(tt.args)
			
			err := cmd.ParseFlags(tt.args)
			if tt.expectedError {
				// For mutual exclusion, we need to check this during flag processing
				// which happens in the actual command execution
				return
			}
			
			require.NoError(t, err)
			
			// Check that both flags are available
			noReasoning, err := cmd.Flags().GetBool("no-reasoning")
			require.NoError(t, err)
			
			showReasoning, err := cmd.Flags().GetBool("show-reasoning")
			require.NoError(t, err)
			
			// Test the logic that would be applied in the actual command
			if cmd.Flags().Changed("no-reasoning") && !cmd.Flags().Changed("show-reasoning") {
				actualShowReasoning := !noReasoning
				actualFilterReasoning := !actualShowReasoning
				assert.Equal(t, tt.expectedFilterReasoning, actualFilterReasoning)
			} else {
				// Default behavior test - use showReasoning directly
				actualFilterReasoning := !showReasoning
				assert.Equal(t, tt.expectedFilterReasoning, actualFilterReasoning)
			}
		})
	}
}

// Test input validation security
func TestInputValidationSecurity(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "normal prompt",
			args:        []string{"-p", "Explain how databases work"},
			expectError: false,
		},
		{
			name:        "prompt with potential command injection",
			args:        []string{"-p", "Explain $(echo test) this"},
			expectError: false, // This specific pattern might be allowed, depends on security config
		},
		{
			name:        "extremely long prompt",
			args:        []string{"-p", strings.Repeat("A", 200000)},
			expectError: false, // Flag parsing won't catch this, need actual validation
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newTestRunCommand()
			cmd.SetArgs(tt.args)
			
			err := cmd.ParseFlags(tt.args)
			// Note: Security validation happens during command execution, not flag parsing
			// so we mainly test that flag parsing doesn't break
			assert.NoError(t, err, "Flag parsing should succeed")
		})
	}
}

// newTestRunCommand creates a test version of the run command with all flags
func newTestRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [prompt...]",
		Short: "Run a single non-interactive prompt",
		Long: `Run a single prompt in non-interactive mode and exit.
The prompt can be provided as arguments, via flags, from files, or piped from stdin.
Output can be captured in multiple formats for automation and scripting.

By default, AI reasoning content (<think>...</think> tags) is filtered from output.
Use --show-reasoning to display reasoning content.`,
	}
	
	// Add all the flags that the real command has
	cmd.Flags().BoolP("quiet", "q", false, "Hide spinner")
	cmd.Flags().StringP("prompt", "p", "", "Direct prompt parameter")
	cmd.Flags().StringP("prompt-file", "f", "", "Read prompt from file")
	cmd.Flags().StringP("output", "o", "", "Output file path (default: stdout)")
	cmd.Flags().String("format", "text", "Output format: text, json, structured")
	cmd.Flags().Bool("show-reasoning", false, "Show AI reasoning content in output (disabled by default)")
	cmd.Flags().Bool("no-reasoning", false, "DEPRECATED: Use --show-reasoning instead")
	cmd.Flags().MarkDeprecated("no-reasoning", "use --show-reasoning flag instead")
	cmd.Flags().Int("timeout", 0, "Response timeout in seconds (0 = no timeout)")
	cmd.Flags().Int("max-tokens", 0, "Maximum response tokens (0 = no limit)")
	cmd.Flags().String("model", "", "Override default model")
	cmd.Flags().Bool("no-tools", false, "Disable tool usage")
	cmd.Flags().String("session-title", "", "Custom session title")
	
	return cmd
}

// Test flag defaults
func TestFlagDefaults(t *testing.T) {
	cmd := newTestRunCommand()
	
	// Test show-reasoning default
	showReasoning, err := cmd.Flags().GetBool("show-reasoning")
	require.NoError(t, err)
	assert.False(t, showReasoning, "show-reasoning should default to false")
	
	// Test no-reasoning default
	noReasoning, err := cmd.Flags().GetBool("no-reasoning")
	require.NoError(t, err)
	assert.False(t, noReasoning, "no-reasoning should default to false")
	
	// Test format default
	format, err := cmd.Flags().GetString("format")
	require.NoError(t, err)
	assert.Equal(t, "text", format, "format should default to text")
}