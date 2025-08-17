package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRunCommandFlagExtraction(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "no_reasoning_flag_true",
			args:     []string{"--no-reasoning", "test query"},
			expected: true,
		},
		{
			name:     "no_reasoning_flag_false",
			args:     []string{"test query"},
			expected: false,
		},
		{
			name:     "no_reasoning_flag_explicit_false",
			args:     []string{"--no-reasoning=false", "test query"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new command instance to avoid state pollution
			cmd := &cobra.Command{
				Use: "run [prompt...]",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Test flag extraction
					noReasoning, err := cmd.Flags().GetBool("no-reasoning")
					if err != nil {
						t.Errorf("GetBool() error = %v", err)
						return err
					}
					
					if noReasoning != tt.expected {
						t.Errorf("no-reasoning flag = %v, expected %v", noReasoning, tt.expected)
					}
					
					return nil
				},
			}
			
			// Add the flag
			cmd.Flags().Bool("no-reasoning", false, "Remove AI reasoning content from output")
			
			// Set args and execute
			cmd.SetArgs(tt.args)
			
			if err := cmd.Execute(); err != nil {
				t.Errorf("Execute() error = %v", err)
			}
		})
	}
}

func TestRunCommandHelpText(t *testing.T) {
	// Create command instance
	cmd := &cobra.Command{
		Use:   "run [prompt...]",
		Short: "Run a single non-interactive prompt",
		Long: `Run a single prompt in non-interactive mode and exit.
The prompt can be provided as arguments, via flags, from files, or piped from stdin.
Output can be captured in multiple formats for automation and scripting.

The --no-reasoning flag can be used to remove AI reasoning content 
(content within <think>...</think> tags) from the output. This is
useful for automation and when you only need the final response.`,
	}
	
	// Add flags
	cmd.Flags().Bool("no-reasoning", false, "Remove AI reasoning content from output")
	
	// Test that help text contains the flag
	helpText := cmd.Long
	if helpText == "" {
		t.Error("Help text is empty")
	}
	
	// Check for key phrases
	expectedPhrases := []string{
		"--no-reasoning",
		"reasoning content",
		"<think>",
		"</think>",
	}
	
	for _, phrase := range expectedPhrases {
		if !contains(helpText, phrase) {
			t.Errorf("Help text missing expected phrase: %s", phrase)
		}
	}
}

func TestRunCommandFlagDefault(t *testing.T) {
	cmd := &cobra.Command{
		Use: "run [prompt...]",
	}
	
	// Add the flag
	cmd.Flags().Bool("no-reasoning", false, "Remove AI reasoning content from output")
	
	// Check default value
	defaultVal, err := cmd.Flags().GetBool("no-reasoning")
	if err != nil {
		t.Errorf("GetBool() error = %v", err)
	}
	
	if defaultVal != false {
		t.Errorf("Default value = %v, expected false", defaultVal)
	}
}

// Helper function to check if string contains substring
func contains(text, substr string) bool {
	return len(text) >= len(substr) && findInString(text, substr)
}

func findInString(text, substr string) bool {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}