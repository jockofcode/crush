package security

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	ErrPathNotAllowed           = errors.New("file path not in allowed directory")
	ErrPathTraversalAttempt     = errors.New("path traversal attempt detected")
	ErrFileExtensionBlocked     = errors.New("file extension is blocked")
	ErrFileSizeExceeded         = errors.New("file size exceeds maximum limit")
	ErrInsufficientPermissions  = errors.New("insufficient permissions for operation")
	ErrFileDoesNotExist         = errors.New("file does not exist")
	ErrDirectoryNotAllowed      = errors.New("directory access not allowed")
	ErrMaliciousPath            = errors.New("malicious path detected")
)

// SecureFileReader provides secure file reading with validation and restrictions
type SecureFileReader struct {
	config            *FileSecurityConfig
	allowedDirs       []string
	blockedDirs       []string
	auditor           *SecurityAuditor
	allowedExtensions map[string]bool
	blockedExtensions map[string]bool
}

// NewSecureFileReader creates a new secure file reader with the given configuration
func NewSecureFileReader(config *FileSecurityConfig, auditor *SecurityAuditor) *SecureFileReader {
	reader := &SecureFileReader{
		config:  config,
		auditor: auditor,
	}

	// Canonicalize and store allowed directories
	reader.allowedDirs = make([]string, len(config.AllowedDirectories))
	for i, dir := range config.AllowedDirectories {
		canonicalDir, err := filepath.Abs(dir)
		if err != nil {
			canonicalDir = dir // Fall back to original path
		}
		reader.allowedDirs[i] = filepath.Clean(canonicalDir)
	}

	// Canonicalize and store blocked directories
	reader.blockedDirs = make([]string, len(config.BlockedDirectories))
	for i, dir := range config.BlockedDirectories {
		canonicalDir, err := filepath.Abs(dir)
		if err != nil {
			canonicalDir = dir // Fall back to original path
		}
		reader.blockedDirs[i] = filepath.Clean(canonicalDir)
	}

	// Build extension maps for fast lookup
	reader.allowedExtensions = make(map[string]bool)
	for _, ext := range config.AllowedFileExtensions {
		reader.allowedExtensions[strings.ToLower(ext)] = true
	}

	reader.blockedExtensions = make(map[string]bool)
	for _, ext := range config.BlockedFileExtensions {
		reader.blockedExtensions[strings.ToLower(ext)] = true
	}

	return reader
}

// ReadSecurely reads a file with security validation and size limits
func (r *SecureFileReader) ReadSecurely(path string) (string, error) {
	// Validate the path first
	validatedPath, err := r.ValidatePath(path)
	if err != nil {
		r.auditor.LogSecurityEvent("file_access_denied", map[string]interface{}{
			"path":   path,
			"error":  err.Error(),
			"action": "read",
		})
		return "", err
	}

	// Check file permissions and existence
	fileInfo, err := os.Stat(validatedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrFileDoesNotExist
		}
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	// Ensure it's a regular file
	if !fileInfo.Mode().IsRegular() {
		r.auditor.LogSecurityEvent("non_regular_file_access_attempted", map[string]interface{}{
			"path": validatedPath,
			"mode": fileInfo.Mode().String(),
		})
		return "", errors.New("not a regular file")
	}

	// Check file size
	if fileInfo.Size() > r.config.MaxFileSize {
		r.auditor.LogSecurityEvent("file_size_exceeded", map[string]interface{}{
			"path":      validatedPath,
			"size":      fileInfo.Size(),
			"max_size":  r.config.MaxFileSize,
		})
		return "", fmt.Errorf("%w: file size %d exceeds limit %d", 
			ErrFileSizeExceeded, fileInfo.Size(), r.config.MaxFileSize)
	}

	// Open and read the file securely
	file, err := os.Open(validatedPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read with a limited reader to enforce size constraints
	limitedReader := io.LimitReader(file, r.config.MaxFileSize)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Log successful read
	r.auditor.LogSecurityEvent("file_read_success", map[string]interface{}{
		"path":         validatedPath,
		"size":         len(content),
		"content_hash": r.hashContent(content),
	})

	return string(content), nil
}

// ReadLines reads a file line by line with security validation
func (r *SecureFileReader) ReadLines(path string, maxLines int) ([]string, error) {
	validatedPath, err := r.ValidatePath(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(validatedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	lineCount := 0

	for scanner.Scan() && lineCount < maxLines {
		lines = append(lines, scanner.Text())
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	r.auditor.LogSecurityEvent("file_lines_read", map[string]interface{}{
		"path":       validatedPath,
		"lines_read": lineCount,
		"max_lines":  maxLines,
	})

	return lines, nil
}

// ValidatePath performs comprehensive path validation and canonicalization
func (r *SecureFileReader) ValidatePath(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty path")
	}

	// Check for malicious patterns in the path
	if err := r.checkMaliciousPatterns(path); err != nil {
		return "", err
	}

	// Canonicalize the path
	canonicalPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to canonicalize path: %w", err)
	}
	canonicalPath = filepath.Clean(canonicalPath)

	// Check for path traversal
	if r.config.PreventTraversal && r.hasPathTraversal(path, canonicalPath) {
		return "", ErrPathTraversalAttempt
	}

	// Check if path is in blocked directories
	if r.isPathBlocked(canonicalPath) {
		return "", ErrDirectoryNotAllowed
	}

	// Check if path is in allowed directories (if specified)
	if len(r.allowedDirs) > 0 && !r.isPathAllowed(canonicalPath) {
		return "", ErrPathNotAllowed
	}

	// Check file extension
	if err := r.validateFileExtension(canonicalPath); err != nil {
		return "", err
	}

	return canonicalPath, nil
}

func (r *SecureFileReader) checkMaliciousPatterns(path string) error {
	maliciousPatterns := []string{
		`\x00`,           // Null bytes
		`\.\./`,          // Path traversal
		`\\\.\.\\`,       // Windows path traversal
		`/proc/`,         // Linux proc filesystem
		`/dev/`,          // Device files
		`/sys/`,          // System files
		`\\\\\.\\`,       // Windows device namespace
		`CON`, `PRN`, `AUX`, `NUL`, // Windows reserved names
		`COM[1-9]`, `LPT[1-9]`,     // Windows reserved device names
	}

	pathLower := strings.ToLower(path)
	
	for _, pattern := range maliciousPatterns {
		matched, err := regexp.MatchString(pattern, pathLower)
		if err != nil {
			continue // Skip invalid patterns
		}
		if matched {
			return fmt.Errorf("%w: pattern '%s' detected", ErrMaliciousPath, pattern)
		}
	}

	return nil
}

func (r *SecureFileReader) hasPathTraversal(originalPath, canonicalPath string) bool {
	// Check for directory traversal patterns
	traversalPatterns := []string{
		"../", "..\\", 
		"%2e%2e%2f", "%2e%2e%5c", // URL encoded
		"%252e%252e%252f",        // Double URL encoded
	}

	lowerOriginal := strings.ToLower(originalPath)
	for _, pattern := range traversalPatterns {
		if strings.Contains(lowerOriginal, pattern) {
			return true
		}
	}

	// Additional check: ensure canonical path doesn't escape intended boundaries
	workingDir, _ := os.Getwd()
	if workingDir != "" {
		workingDirAbs, _ := filepath.Abs(workingDir)
		if !strings.HasPrefix(canonicalPath, workingDirAbs) {
			// Path escapes working directory
			return true
		}
	}

	return false
}

func (r *SecureFileReader) isPathBlocked(path string) bool {
	for _, blockedDir := range r.blockedDirs {
		if strings.HasPrefix(path, blockedDir) {
			return true
		}
	}
	return false
}

func (r *SecureFileReader) isPathAllowed(path string) bool {
	for _, allowedDir := range r.allowedDirs {
		if strings.HasPrefix(path, allowedDir) {
			return true
		}
	}
	return false
}

func (r *SecureFileReader) validateFileExtension(path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	
	// Check blocked extensions first
	if len(r.blockedExtensions) > 0 && r.blockedExtensions[ext] {
		return fmt.Errorf("%w: extension '%s'", ErrFileExtensionBlocked, ext)
	}

	// If allowed extensions are specified, check if this extension is allowed
	if len(r.allowedExtensions) > 0 && !r.allowedExtensions[ext] {
		return fmt.Errorf("%w: extension '%s' not in allowed list", ErrFileExtensionBlocked, ext)
	}

	return nil
}

func (r *SecureFileReader) hashContent(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash)[:16] // First 16 characters of hash
}

// SecureFileWriter provides secure file writing with validation and restrictions
type SecureFileWriter struct {
	config  *FileSecurityConfig
	reader  *SecureFileReader // Reuse path validation
	auditor *SecurityAuditor
}

// NewSecureFileWriter creates a new secure file writer
func NewSecureFileWriter(config *FileSecurityConfig, auditor *SecurityAuditor) *SecureFileWriter {
	return &SecureFileWriter{
		config:  config,
		reader:  NewSecureFileReader(config, auditor),
		auditor: auditor,
	}
}

// WriteSecurely writes content to a file with security validation
func (w *SecureFileWriter) WriteSecurely(path string, content []byte) error {
	// Validate the path
	validatedPath, err := w.reader.ValidatePath(path)
	if err != nil {
		w.auditor.LogSecurityEvent("file_write_denied", map[string]interface{}{
			"path":  path,
			"error": err.Error(),
		})
		return err
	}

	// Check content size
	if int64(len(content)) > w.config.MaxFileSize {
		w.auditor.LogSecurityEvent("file_write_size_exceeded", map[string]interface{}{
			"path":     validatedPath,
			"size":     len(content),
			"max_size": w.config.MaxFileSize,
		})
		return fmt.Errorf("%w: content size %d exceeds limit %d", 
			ErrFileSizeExceeded, len(content), w.config.MaxFileSize)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(validatedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temporary file first for atomic operation
	tempPath := validatedPath + ".tmp." + fmt.Sprintf("%d", time.Now().UnixNano())
	
	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	// Write content
	_, writeErr := file.Write(content)
	closeErr := file.Close()

	if writeErr != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to write content: %w", writeErr)
	}

	if closeErr != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to close file: %w", closeErr)
	}

	// Atomically move temp file to final location
	if err := os.Rename(tempPath, validatedPath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to move file to final location: %w", err)
	}

	// Log successful write
	w.auditor.LogSecurityEvent("file_write_success", map[string]interface{}{
		"path":         validatedPath,
		"size":         len(content),
		"content_hash": w.reader.hashContent(content),
	})

	return nil
}

// AppendSecurely appends content to a file with security validation
func (w *SecureFileWriter) AppendSecurely(path string, content []byte) error {
	validatedPath, err := w.reader.ValidatePath(path)
	if err != nil {
		return err
	}

	// Check if file exists and get current size
	var currentSize int64
	if info, err := os.Stat(validatedPath); err == nil {
		currentSize = info.Size()
	}

	// Check total size after append
	totalSize := currentSize + int64(len(content))
	if totalSize > w.config.MaxFileSize {
		w.auditor.LogSecurityEvent("file_append_size_exceeded", map[string]interface{}{
			"path":         validatedPath,
			"current_size": currentSize,
			"append_size":  len(content),
			"total_size":   totalSize,
			"max_size":     w.config.MaxFileSize,
		})
		return fmt.Errorf("%w: total size %d would exceed limit %d", 
			ErrFileSizeExceeded, totalSize, w.config.MaxFileSize)
	}

	// Open file for append
	file, err := os.OpenFile(validatedPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for append: %w", err)
	}
	defer file.Close()

	// Write content
	if _, err := file.Write(content); err != nil {
		return fmt.Errorf("failed to append content: %w", err)
	}

	w.auditor.LogSecurityEvent("file_append_success", map[string]interface{}{
		"path":        validatedPath,
		"append_size": len(content),
		"total_size":  totalSize,
	})

	return nil
}