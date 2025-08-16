package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInputProcessor_ProcessPrompt(t *testing.T) {
	processor := NewInputProcessor()

	tests := []struct {
		name        string
		params      PromptParams
		envVars     map[string]string
		want        string
		wantErr     bool
		description string
	}{
		{
			name: "direct prompt only",
			params: PromptParams{
				DirectPrompt: "Hello world",
			},
			want:        "Hello world",
			wantErr:     false,
			description: "Should process direct prompt parameter",
		},
		{
			name: "environment variable expansion",
			params: PromptParams{
				DirectPrompt: "Analyze ${TEST_FILE} for security issues",
			},
			envVars: map[string]string{
				"TEST_FILE": "main.go",
			},
			want:        "Analyze main.go for security issues",
			wantErr:     false,
			description: "Should expand environment variables",
		},
		{
			name: "multiple prompts combined",
			params: PromptParams{
				DirectPrompt: "First prompt",
				Args:         []string{"second", "prompt"},
			},
			want:        "First prompt\n\nsecond prompt",
			wantErr:     false,
			description: "Should combine multiple prompt sources",
		},
		{
			name: "empty prompt",
			params: PromptParams{
				DirectPrompt: "",
			},
			want:        "",
			wantErr:     true,
			description: "Should fail with empty prompt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for test
			for key, value := range tt.envVars {
				originalValue := os.Getenv(key)
				os.Setenv(key, value)
				defer func(k, v string) {
					if v == "" {
						os.Unsetenv(k)
					} else {
						os.Setenv(k, v)
					}
				}(key, originalValue)
			}

			got, err := processor.ProcessPrompt(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessPrompt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ProcessPrompt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInputProcessor_ExpandEnvironmentVars(t *testing.T) {
	processor := NewInputProcessor()

	tests := []struct {
		name    string
		text    string
		envVars map[string]string
		want    string
	}{
		{
			name: "basic variable expansion",
			text: "Hello ${NAME}",
			envVars: map[string]string{
				"NAME": "World",
			},
			want: "Hello World",
		},
		{
			name: "dollar sign variable expansion",
			text: "Hello $NAME",
			envVars: map[string]string{
				"NAME": "World",
			},
			want: "Hello World",
		},
		{
			name: "multiple variables",
			text: "${GREETING} ${NAME}!",
			envVars: map[string]string{
				"GREETING": "Hello",
				"NAME":     "World",
			},
			want: "Hello World!",
		},
		{
			name: "missing variable",
			text: "Hello ${MISSING}",
			envVars: map[string]string{},
			want: "Hello ${MISSING}",
		},
		{
			name: "no variables",
			text: "Hello World",
			envVars: map[string]string{},
			want: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for test
			for key, value := range tt.envVars {
				originalValue := os.Getenv(key)
				os.Setenv(key, value)
				defer func(k, v string) {
					if v == "" {
						os.Unsetenv(k)
					} else {
						os.Setenv(k, v)
					}
				}(key, originalValue)
			}

			got, err := processor.ExpandEnvironmentVars(tt.text)
			if err != nil {
				t.Errorf("ExpandEnvironmentVars() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("ExpandEnvironmentVars() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInputProcessor_ReadPromptFile(t *testing.T) {
	processor := NewInputProcessor()

	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_prompt.txt")
	testContent := "This is a test prompt\nwith multiple lines"
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name:    "valid file",
			path:    testFile,
			want:    testContent,
			wantErr: false,
		},
		{
			name:    "non-existent file",
			path:    filepath.Join(tmpDir, "nonexistent.txt"),
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processor.ReadPromptFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadPromptFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReadPromptFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInputProcessor_ValidateInput(t *testing.T) {
	processor := NewInputProcessor()

	tests := []struct {
		name    string
		prompt  string
		wantErr bool
	}{
		{
			name:    "valid prompt",
			prompt:  "This is a valid prompt",
			wantErr: false,
		},
		{
			name:    "empty prompt",
			prompt:  "",
			wantErr: true,
		},
		{
			name:    "prompt with null bytes",
			prompt:  "Invalid\x00prompt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := processor.ValidateInput(tt.prompt)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInputProcessor_SetAllowedEnvVars(t *testing.T) {
	processor := NewInputProcessor()
	processor.SetAllowedEnvVars([]string{"ALLOWED_VAR"})

	// Set environment variables
	os.Setenv("ALLOWED_VAR", "allowed_value")
	os.Setenv("FORBIDDEN_VAR", "forbidden_value")
	defer func() {
		os.Unsetenv("ALLOWED_VAR")
		os.Unsetenv("FORBIDDEN_VAR")
	}()

	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "allowed variable expanded",
			text: "Value: ${ALLOWED_VAR}",
			want: "Value: allowed_value",
		},
		{
			name: "forbidden variable not expanded",
			text: "Value: ${FORBIDDEN_VAR}",
			want: "Value: ${FORBIDDEN_VAR}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processor.ExpandEnvironmentVars(tt.text)
			if err != nil {
				t.Errorf("ExpandEnvironmentVars() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("ExpandEnvironmentVars() = %v, want %v", got, tt.want)
			}
		})
	}
}