package security

import (
	"regexp"
	"time"
)

// SecurityMode defines the security level for operations
type SecurityMode string

const (
	SecurityModeStrict   SecurityMode = "strict"
	SecurityModeBalanced SecurityMode = "balanced"
	SecurityModeRelaxed  SecurityMode = "relaxed"
)

// RedactionLevel controls how sensitive data is redacted
type RedactionLevel string

const (
	RedactionNone    RedactionLevel = "none"
	RedactionPartial RedactionLevel = "partial"
	RedactionFull    RedactionLevel = "full"
)

// SecurityConfig holds comprehensive security settings for the application
type SecurityConfig struct {
	DefaultMode     SecurityMode        `yaml:"default_mode" json:"default_mode"`
	Input           InputSecurityConfig `yaml:"input" json:"input"`
	Environment     EnvSecurityConfig   `yaml:"environment" json:"environment"`
	Output          OutputSecurityConfig `yaml:"output" json:"output"`
	Permissions     PermissionConfig    `yaml:"permissions" json:"permissions"`
	FileOperations  FileSecurityConfig  `yaml:"file_operations" json:"file_operations"`
	AuditLog        AuditConfig         `yaml:"audit_log" json:"audit_log"`
}

// InputSecurityConfig controls input validation and sanitization
type InputSecurityConfig struct {
	MaxSize            int64    `yaml:"max_size" json:"max_size"`
	MaxLines           int      `yaml:"max_lines" json:"max_lines"`
	AllowedCharPattern string   `yaml:"allowed_char_pattern" json:"allowed_char_pattern"`
	BlockedPatterns    []string `yaml:"blocked_patterns" json:"blocked_patterns"`
	EnableSanitization bool     `yaml:"enable_sanitization" json:"enable_sanitization"`
	StripNullBytes     bool     `yaml:"strip_null_bytes" json:"strip_null_bytes"`
	NormalizeLineEnds  bool     `yaml:"normalize_line_ends" json:"normalize_line_ends"`
}

// EnvSecurityConfig controls environment variable access and expansion
type EnvSecurityConfig struct {
	EnableExpansion   bool     `yaml:"enable_expansion" json:"enable_expansion"`
	WhitelistPatterns []string `yaml:"whitelist_patterns" json:"whitelist_patterns"`
	BlacklistPatterns []string `yaml:"blacklist_patterns" json:"blacklist_patterns"`
	MaxVarSize        int      `yaml:"max_var_size" json:"max_var_size"`
	MaxVarCount       int      `yaml:"max_var_count" json:"max_var_count"`
	SensitivePatterns []string `yaml:"sensitive_patterns" json:"sensitive_patterns"`
}

// OutputSecurityConfig controls output capture and sensitive data handling
type OutputSecurityConfig struct {
	EnableSensitiveDetection bool           `yaml:"enable_sensitive_detection" json:"enable_sensitive_detection"`
	RedactionLevel           RedactionLevel `yaml:"redaction_level" json:"redaction_level"`
	MaxCaptureSize           int64          `yaml:"max_capture_size" json:"max_capture_size"`
	BufferFlushInterval      time.Duration  `yaml:"buffer_flush_interval" json:"buffer_flush_interval"`
	EnableEncryption         bool           `yaml:"enable_encryption" json:"enable_encryption"`
}

// PermissionConfig extends the existing permission system
type PermissionConfig struct {
	RequireExplicitApproval bool     `yaml:"require_explicit_approval" json:"require_explicit_approval"`
	SessionIsolation        bool     `yaml:"session_isolation" json:"session_isolation"`
	ToolRestrictions        []string `yaml:"tool_restrictions" json:"tool_restrictions"`
	TimeoutDuration         int      `yaml:"timeout_duration" json:"timeout_duration"`
}

// FileSecurityConfig controls file operation security
type FileSecurityConfig struct {
	AllowedDirectories    []string `yaml:"allowed_directories" json:"allowed_directories"`
	BlockedDirectories    []string `yaml:"blocked_directories" json:"blocked_directories"`
	MaxFileSize           int64    `yaml:"max_file_size" json:"max_file_size"`
	AllowedFileExtensions []string `yaml:"allowed_file_extensions" json:"allowed_file_extensions"`
	BlockedFileExtensions []string `yaml:"blocked_file_extensions" json:"blocked_file_extensions"`
	EnablePathValidation  bool     `yaml:"enable_path_validation" json:"enable_path_validation"`
	PreventTraversal      bool     `yaml:"prevent_traversal" json:"prevent_traversal"`
}

// AuditConfig controls security audit logging
type AuditConfig struct {
	Enabled         bool   `yaml:"enabled" json:"enabled"`
	LogLevel        string `yaml:"log_level" json:"log_level"`
	IncludePayloads bool   `yaml:"include_payloads" json:"include_payloads"`
	MaxLogSize      int64  `yaml:"max_log_size" json:"max_log_size"`
	RetentionDays   int    `yaml:"retention_days" json:"retention_days"`
}

// DefaultSecurityConfig returns a secure default configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		DefaultMode: SecurityModeBalanced,
		Input: InputSecurityConfig{
			MaxSize:            10 * 1024 * 1024, // 10MB
			MaxLines:           10000,
			AllowedCharPattern: `[\p{L}\p{N}\p{P}\p{S}\p{Z}\s\n\r\t]`,
			BlockedPatterns: []string{
				`\$\(.*\)`,           // Command substitution
				`\x00`,               // Null bytes
				`\x0[1-8]`,           // Control characters
				`\x0[e-f]`,           // More control characters
				`\x1[0-9a-f]`,        // More control characters
				`<script.*?>.*?</script>`, // Script tags
				`javascript:`,        // JavaScript URLs
				`vbscript:`,         // VBScript URLs
			},
			EnableSanitization: true,
			StripNullBytes:     true,
			NormalizeLineEnds:  true,
		},
		Environment: EnvSecurityConfig{
			EnableExpansion: false, // Disabled by default for security
			WhitelistPatterns: []string{
				`^[A-Z][A-Z0-9_]*$`, // Standard environment variable pattern
			},
			BlacklistPatterns: []string{
				`(?i).*password.*`,
				`(?i).*secret.*`,
				`(?i).*key.*`,
				`(?i).*token.*`,
				`(?i).*auth.*`,
				`(?i).*credential.*`,
				`(?i).*api_key.*`,
				`(?i).*private.*`,
			},
			MaxVarSize:  1024, // 1KB per variable
			MaxVarCount: 100,  // Max 100 variables expanded
			SensitivePatterns: []string{
				`(?i).*password.*`,
				`(?i).*secret.*`,
				`(?i).*key.*`,
				`(?i).*token.*`,
			},
		},
		Output: OutputSecurityConfig{
			EnableSensitiveDetection: true,
			RedactionLevel:           RedactionPartial,
			MaxCaptureSize:           100 * 1024 * 1024, // 100MB
			BufferFlushInterval:      5 * time.Second,
			EnableEncryption:         false, // Would require key management
		},
		Permissions: PermissionConfig{
			RequireExplicitApproval: true,
			SessionIsolation:        true,
			ToolRestrictions:        []string{},
			TimeoutDuration:         300, // 5 minutes
		},
		FileOperations: FileSecurityConfig{
			AllowedDirectories: []string{
				".", // Current directory
			},
			BlockedDirectories: []string{
				"/etc",
				"/proc",
				"/sys",
				"/dev",
				"/root",
				"/boot",
				"/var/log",
				"/usr/bin",
				"/usr/sbin",
				"/sbin",
				"/bin",
			},
			MaxFileSize:           50 * 1024 * 1024, // 50MB
			AllowedFileExtensions: []string{}, // Empty means all allowed except blocked
			BlockedFileExtensions: []string{
				".exe", ".bat", ".cmd", ".com", ".scr", ".pif",
				".dll", ".sys", ".drv", ".vbs", ".js", ".jar",
			},
			EnablePathValidation: true,
			PreventTraversal:     true,
		},
		AuditLog: AuditConfig{
			Enabled:         true,
			LogLevel:        "INFO",
			IncludePayloads: false, // Don't log sensitive data by default
			MaxLogSize:      10 * 1024 * 1024, // 10MB
			RetentionDays:   30,
		},
	}
}

// Validate ensures the security configuration is valid and consistent
func (c *SecurityConfig) Validate() error {
	// Validate input configuration
	if c.Input.MaxSize <= 0 {
		c.Input.MaxSize = 10 * 1024 * 1024 // Default to 10MB
	}
	if c.Input.MaxLines <= 0 {
		c.Input.MaxLines = 10000
	}

	// Validate regex patterns
	if c.Input.AllowedCharPattern != "" {
		if _, err := regexp.Compile(c.Input.AllowedCharPattern); err != nil {
			return err
		}
	}

	for _, pattern := range c.Input.BlockedPatterns {
		if _, err := regexp.Compile(pattern); err != nil {
			return err
		}
	}

	// Validate environment configuration
	if c.Environment.MaxVarSize <= 0 {
		c.Environment.MaxVarSize = 1024
	}
	if c.Environment.MaxVarCount <= 0 {
		c.Environment.MaxVarCount = 100
	}

	// Validate output configuration
	if c.Output.MaxCaptureSize <= 0 {
		c.Output.MaxCaptureSize = 100 * 1024 * 1024
	}
	if c.Output.BufferFlushInterval <= 0 {
		c.Output.BufferFlushInterval = 5 * time.Second
	}

	// Validate file operations configuration
	if c.FileOperations.MaxFileSize <= 0 {
		c.FileOperations.MaxFileSize = 50 * 1024 * 1024
	}

	return nil
}

// IsStrictMode returns true if security is in strict mode
func (c *SecurityConfig) IsStrictMode() bool {
	return c.DefaultMode == SecurityModeStrict
}

// IsRelaxedMode returns true if security is in relaxed mode
func (c *SecurityConfig) IsRelaxedMode() bool {
	return c.DefaultMode == SecurityModeRelaxed
}