package security

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"
)

// SecurityAuditor handles security event logging and monitoring
type SecurityAuditor struct {
	config   *AuditConfig
	logger   *slog.Logger
	events   []SecurityEvent
	eventsMu sync.RWMutex
}

// SecurityEvent represents a security-related event in the system
type SecurityEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	Level       SecurityLevel          `json:"level"`
	Message     string                 `json:"message"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	UserContext map[string]string      `json:"user_context,omitempty"`
}

// SecurityLevel represents the severity of a security event
type SecurityLevel string

const (
	SecurityLevelInfo     SecurityLevel = "INFO"
	SecurityLevelWarning  SecurityLevel = "WARNING"
	SecurityLevelError    SecurityLevel = "ERROR"
	SecurityLevelCritical SecurityLevel = "CRITICAL"
)

// NewSecurityAuditor creates a new security auditor with the given configuration
func NewSecurityAuditor(config *AuditConfig, logger *slog.Logger) *SecurityAuditor {
	if logger == nil {
		logger = slog.Default()
	}

	return &SecurityAuditor{
		config: config,
		logger: logger.With("component", "security_auditor"),
		events: make([]SecurityEvent, 0),
	}
}

// LogSecurityEvent records a security event with the specified type and metadata
func (a *SecurityAuditor) LogSecurityEvent(eventType string, metadata map[string]interface{}) {
	if !a.config.Enabled {
		return
	}

	level := a.determineLevel(eventType)
	event := SecurityEvent{
		Timestamp: time.Now().UTC(),
		EventType: eventType,
		Level:     level,
		Message:   a.generateMessage(eventType, metadata),
		Metadata:  metadata,
	}

	// Add stack trace for critical events
	if level == SecurityLevelCritical || level == SecurityLevelError {
		event.StackTrace = a.captureStackTrace()
	}

	// Store event in memory (with rotation)
	a.storeEvent(event)

	// Log to structured logger
	a.logToStructuredLogger(event)
}

// LogSecurityEventWithContext records a security event with additional context
func (a *SecurityAuditor) LogSecurityEventWithContext(
	eventType string, 
	metadata map[string]interface{}, 
	sessionID string, 
	userContext map[string]string,
) {
	if !a.config.Enabled {
		return
	}

	level := a.determineLevel(eventType)
	event := SecurityEvent{
		Timestamp:   time.Now().UTC(),
		EventType:   eventType,
		Level:       level,
		Message:     a.generateMessage(eventType, metadata),
		Metadata:    metadata,
		SessionID:   sessionID,
		UserContext: userContext,
	}

	// Add stack trace for critical events
	if level == SecurityLevelCritical || level == SecurityLevelError {
		event.StackTrace = a.captureStackTrace()
	}

	// Store event in memory (with rotation)
	a.storeEvent(event)

	// Log to structured logger
	a.logToStructuredLogger(event)
}

// GetRecentEvents returns recent security events for monitoring
func (a *SecurityAuditor) GetRecentEvents(limit int) []SecurityEvent {
	a.eventsMu.RLock()
	defer a.eventsMu.RUnlock()

	start := 0
	if len(a.events) > limit {
		start = len(a.events) - limit
	}

	result := make([]SecurityEvent, len(a.events)-start)
	copy(result, a.events[start:])
	return result
}

// GetEventsByType returns events filtered by type
func (a *SecurityAuditor) GetEventsByType(eventType string, limit int) []SecurityEvent {
	a.eventsMu.RLock()
	defer a.eventsMu.RUnlock()

	var result []SecurityEvent
	count := 0

	// Search backwards through events for most recent matches
	for i := len(a.events) - 1; i >= 0 && count < limit; i-- {
		if a.events[i].EventType == eventType {
			result = append([]SecurityEvent{a.events[i]}, result...)
			count++
		}
	}

	return result
}

// ExportEvents exports all events as JSON for external analysis
func (a *SecurityAuditor) ExportEvents() ([]byte, error) {
	a.eventsMu.RLock()
	defer a.eventsMu.RUnlock()

	return json.MarshalIndent(a.events, "", "  ")
}

func (a *SecurityAuditor) determineLevel(eventType string) SecurityLevel {
	// Map event types to security levels
	criticalEvents := map[string]bool{
		"system_compromise":       true,
		"privilege_escalation":    true,
		"data_exfiltration":      true,
		"malicious_code_detected": true,
	}

	errorEvents := map[string]bool{
		"input_validation_failed":     true,
		"blocked_pattern_detected":    true,
		"file_access_denied":         true,
		"permission_violation":       true,
		"suspicious_content_detected": true,
		"environment_variable_blocked": true,
	}

	warningEvents := map[string]bool{
		"unusual_activity":           true,
		"rate_limit_approached":      true,
		"large_input_detected":       true,
		"multiple_failed_attempts":   true,
		"sensitive_data_detected":    true,
	}

	if criticalEvents[eventType] {
		return SecurityLevelCritical
	}
	if errorEvents[eventType] {
		return SecurityLevelError
	}
	if warningEvents[eventType] {
		return SecurityLevelWarning
	}

	return SecurityLevelInfo
}

func (a *SecurityAuditor) generateMessage(eventType string, metadata map[string]interface{}) string {
	// Generate human-readable messages for common event types
	switch eventType {
	case "input_validation_failed":
		if err, ok := metadata["error"]; ok {
			return fmt.Sprintf("Input validation failed: %v", err)
		}
		return "Input validation failed"

	case "blocked_pattern_detected":
		if pattern, ok := metadata["pattern"]; ok {
			return fmt.Sprintf("Blocked pattern detected: %v", pattern)
		}
		return "Blocked pattern detected in input"

	case "file_access_denied":
		if path, ok := metadata["path"]; ok {
			return fmt.Sprintf("File access denied: %v", path)
		}
		return "File access denied"

	case "environment_variable_blocked":
		if varName, ok := metadata["variable"]; ok {
			return fmt.Sprintf("Environment variable access blocked: %v", varName)
		}
		return "Environment variable access blocked"

	case "sensitive_data_detected":
		if dataType, ok := metadata["data_type"]; ok {
			return fmt.Sprintf("Sensitive data detected: %v", dataType)
		}
		return "Sensitive data detected in output"

	case "permission_violation":
		if operation, ok := metadata["operation"]; ok {
			return fmt.Sprintf("Permission violation: %v", operation)
		}
		return "Permission violation detected"

	default:
		return fmt.Sprintf("Security event: %s", eventType)
	}
}

func (a *SecurityAuditor) captureStackTrace() string {
	// Capture stack trace for debugging critical issues
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

func (a *SecurityAuditor) storeEvent(event SecurityEvent) {
	a.eventsMu.Lock()
	defer a.eventsMu.Unlock()

	// Add event to in-memory store
	a.events = append(a.events, event)

	// Rotate events if we exceed the maximum
	maxEvents := 1000 // Keep last 1000 events in memory
	if len(a.events) > maxEvents {
		// Remove oldest events
		copy(a.events, a.events[len(a.events)-maxEvents:])
		a.events = a.events[:maxEvents]
	}
}

func (a *SecurityAuditor) logToStructuredLogger(event SecurityEvent) {
	// Create log attributes
	attrs := []slog.Attr{
		slog.String("event_type", event.EventType),
		slog.String("level", string(event.Level)),
		slog.Time("timestamp", event.Timestamp),
	}

	if event.SessionID != "" {
		attrs = append(attrs, slog.String("session_id", event.SessionID))
	}

	// Add metadata as attributes (but be careful with sensitive data)
	if a.config.IncludePayloads && event.Metadata != nil {
		for key, value := range event.Metadata {
			// Skip potentially sensitive fields
			if a.isSensitiveField(key) {
				continue
			}
			attrs = append(attrs, slog.Any(key, value))
		}
	}

	// Log at appropriate level
	switch event.Level {
	case SecurityLevelCritical:
		a.logger.LogAttrs(nil, slog.LevelError, event.Message, attrs...)
	case SecurityLevelError:
		a.logger.LogAttrs(nil, slog.LevelError, event.Message, attrs...)
	case SecurityLevelWarning:
		a.logger.LogAttrs(nil, slog.LevelWarn, event.Message, attrs...)
	default:
		a.logger.LogAttrs(nil, slog.LevelInfo, event.Message, attrs...)
	}
}

func (a *SecurityAuditor) isSensitiveField(fieldName string) bool {
	sensitiveFields := map[string]bool{
		"password":     true,
		"secret":       true,
		"key":          true,
		"token":        true,
		"credential":   true,
		"auth":         true,
		"private":      true,
		"confidential": true,
	}

	return sensitiveFields[fieldName]
}

// AuditStats provides statistics about security events
type AuditStats struct {
	TotalEvents      int                      `json:"total_events"`
	EventsByLevel    map[SecurityLevel]int    `json:"events_by_level"`
	EventsByType     map[string]int           `json:"events_by_type"`
	RecentEvents     int                      `json:"recent_events_1h"`
	LastEventTime    time.Time                `json:"last_event_time"`
}

// GetStats returns audit statistics
func (a *SecurityAuditor) GetStats() AuditStats {
	a.eventsMu.RLock()
	defer a.eventsMu.RUnlock()

	stats := AuditStats{
		TotalEvents:   len(a.events),
		EventsByLevel: make(map[SecurityLevel]int),
		EventsByType:  make(map[string]int),
	}

	now := time.Now()
	oneHourAgo := now.Add(-time.Hour)

	for _, event := range a.events {
		stats.EventsByLevel[event.Level]++
		stats.EventsByType[event.EventType]++

		if event.Timestamp.After(oneHourAgo) {
			stats.RecentEvents++
		}

		if event.Timestamp.After(stats.LastEventTime) {
			stats.LastEventTime = event.Timestamp
		}
	}

	return stats
}