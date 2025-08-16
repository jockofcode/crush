package security

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrInputTooLarge              = errors.New("input exceeds maximum size limit")
	ErrInputTooManyLines          = errors.New("input exceeds maximum line limit")
	ErrInvalidCharacters          = errors.New("input contains invalid characters")
	ErrBlockedPattern             = errors.New("input matches blocked pattern")
	ErrMalformedInput             = errors.New("input is malformed")
	ErrNullBytesDetected          = errors.New("null bytes detected in input")
	ErrSuspiciousContent          = errors.New("suspicious content detected")
)

// InputValidator provides comprehensive input validation and sanitization
type InputValidator struct {
	config            *InputSecurityConfig
	allowedCharsRegex *regexp.Regexp
	blockedPatterns   []*regexp.Regexp
	sanitizer         *InputSanitizer
	auditor           *SecurityAuditor
}

// NewInputValidator creates a new input validator with the given configuration
func NewInputValidator(config *InputSecurityConfig, auditor *SecurityAuditor) (*InputValidator, error) {
	validator := &InputValidator{
		config:  config,
		auditor: auditor,
	}

	// Compile allowed characters pattern
	if config.AllowedCharPattern != "" {
		regex, err := regexp.Compile(config.AllowedCharPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed char pattern: %w", err)
		}
		validator.allowedCharsRegex = regex
	}

	// Compile blocked patterns
	validator.blockedPatterns = make([]*regexp.Regexp, len(config.BlockedPatterns))
	for i, pattern := range config.BlockedPatterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid blocked pattern %d: %w", i, err)
		}
		validator.blockedPatterns[i] = regex
	}

	// Initialize sanitizer
	validator.sanitizer = NewInputSanitizer(config)

	return validator, nil
}

// ValidateInput performs comprehensive validation on input text
func (v *InputValidator) ValidateInput(input string) error {
	// Check size limits
	if err := v.validateSize(input); err != nil {
		v.auditor.LogSecurityEvent("input_validation_failed", map[string]interface{}{
			"error":      err.Error(),
			"input_size": len(input),
			"max_size":   v.config.MaxSize,
		})
		return err
	}

	// Check line limits
	if err := v.validateLineCount(input); err != nil {
		v.auditor.LogSecurityEvent("input_validation_failed", map[string]interface{}{
			"error":      err.Error(),
			"line_count": strings.Count(input, "\n") + 1,
			"max_lines":  v.config.MaxLines,
		})
		return err
	}

	// Check for null bytes
	if v.config.StripNullBytes && strings.Contains(input, "\x00") {
		v.auditor.LogSecurityEvent("null_bytes_detected", map[string]interface{}{
			"input_length": len(input),
		})
		return ErrNullBytesDetected
	}

	// Validate UTF-8 encoding
	if !utf8.ValidString(input) {
		v.auditor.LogSecurityEvent("invalid_utf8_detected", map[string]interface{}{
			"input_length": len(input),
		})
		return ErrMalformedInput
	}

	// Check character restrictions
	if err := v.validateCharacters(input); err != nil {
		v.auditor.LogSecurityEvent("invalid_characters_detected", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	// Check blocked patterns
	if err := v.validatePatterns(input); err != nil {
		v.auditor.LogSecurityEvent("blocked_pattern_detected", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	// Additional security checks
	if err := v.checkSuspiciousContent(input); err != nil {
		v.auditor.LogSecurityEvent("suspicious_content_detected", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	return nil
}

// SanitizeInput cleans and normalizes input text
func (v *InputValidator) SanitizeInput(input string) (string, error) {
	if !v.config.EnableSanitization {
		return input, nil
	}

	result, err := v.sanitizer.Sanitize(input)
	if err != nil {
		v.auditor.LogSecurityEvent("sanitization_failed", map[string]interface{}{
			"error":        err.Error(),
			"input_length": len(input),
		})
		return "", err
	}

	v.auditor.LogSecurityEvent("input_sanitized", map[string]interface{}{
		"original_length": len(input),
		"sanitized_length": len(result),
	})

	return result, nil
}

// ProcessInput validates and optionally sanitizes input
func (v *InputValidator) ProcessInput(input string) (string, error) {
	// First validate the raw input
	if err := v.ValidateInput(input); err != nil {
		return "", err
	}

	// Then sanitize if enabled
	return v.SanitizeInput(input)
}

func (v *InputValidator) validateSize(input string) error {
	if int64(len(input)) > v.config.MaxSize {
		return fmt.Errorf("%w: input size %d exceeds limit %d", 
			ErrInputTooLarge, len(input), v.config.MaxSize)
	}
	return nil
}

func (v *InputValidator) validateLineCount(input string) error {
	lineCount := strings.Count(input, "\n") + 1
	if lineCount > v.config.MaxLines {
		return fmt.Errorf("%w: line count %d exceeds limit %d", 
			ErrInputTooManyLines, lineCount, v.config.MaxLines)
	}
	return nil
}

func (v *InputValidator) validateCharacters(input string) error {
	if v.allowedCharsRegex == nil {
		return nil // No character restrictions
	}

	// Check if entire input matches allowed characters
	if !v.allowedCharsRegex.MatchString(input) {
		// Find first invalid character for better error reporting
		for i, r := range input {
			char := string(r)
			if !v.allowedCharsRegex.MatchString(char) {
				return fmt.Errorf("%w: invalid character '%c' at position %d", 
					ErrInvalidCharacters, r, i)
			}
		}
	}

	return nil
}

func (v *InputValidator) validatePatterns(input string) error {
	for i, pattern := range v.blockedPatterns {
		if pattern.MatchString(input) {
			matches := pattern.FindAllString(input, -1)
			return fmt.Errorf("%w: pattern %d matched: %v", 
				ErrBlockedPattern, i, matches)
		}
	}
	return nil
}

func (v *InputValidator) checkSuspiciousContent(input string) error {
	// Check for potential injection attempts
	suspiciousPatterns := []string{
		`\$\{.*\}`,    // Variable injection
		`\$\(.*\)`,    // Command substitution
		`\x00`,        // Null injection
		`\.\./`,       // Path traversal
		`<\?php`,      // PHP tags
		`<script`,     // Script tags (case insensitive)
		`javascript:`, // JavaScript URLs
		`data:`,       // Data URLs
	}

	lowercaseInput := strings.ToLower(input)
	
	for _, pattern := range suspiciousPatterns {
		matched, err := regexp.MatchString(pattern, lowercaseInput)
		if err != nil {
			continue
		}
		if matched {
			return fmt.Errorf("%w: potentially malicious content detected", ErrSuspiciousContent)
		}
	}

	// Check for excessive special characters (potential obfuscation)
	specialCharCount := 0
	totalChars := 0
	
	for _, r := range input {
		totalChars++
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) && !unicode.IsSpace(r) {
			specialCharCount++
		}
	}

	if totalChars > 0 {
		specialCharRatio := float64(specialCharCount) / float64(totalChars)
		if specialCharRatio > 0.3 { // More than 30% special characters
			slog.Warn("High ratio of special characters detected", 
				"ratio", specialCharRatio, "total_chars", totalChars)
		}
	}

	return nil
}

// InputSanitizer handles input cleaning and normalization
type InputSanitizer struct {
	config *InputSecurityConfig
}

// NewInputSanitizer creates a new input sanitizer
func NewInputSanitizer(config *InputSecurityConfig) *InputSanitizer {
	return &InputSanitizer{config: config}
}

// Sanitize cleans and normalizes input text
func (s *InputSanitizer) Sanitize(input string) (string, error) {
	result := input

	// Strip null bytes
	if s.config.StripNullBytes {
		result = strings.ReplaceAll(result, "\x00", "")
	}

	// Normalize line endings
	if s.config.NormalizeLineEnds {
		result = strings.ReplaceAll(result, "\r\n", "\n")
		result = strings.ReplaceAll(result, "\r", "\n")
	}

	// Remove other control characters except tabs and newlines
	result = strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' {
			return r
		}
		if unicode.IsControl(r) {
			return -1 // Remove character
		}
		return r
	}, result)

	// Trim excessive whitespace
	result = strings.TrimSpace(result)

	// Collapse multiple consecutive newlines
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}

	return result, nil
}