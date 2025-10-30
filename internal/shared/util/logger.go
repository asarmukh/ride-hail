package util

import (
	"encoding/json"
	"os"
	"time"
)

// Logger provides structured JSON logging
type Logger struct {
	service  string
	hostname string
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp   string                 `json:"timestamp"`
	Level       string                 `json:"level"`
	Service     string                 `json:"service"`
	Action      string                 `json:"action,omitempty"`
	Message     string                 `json:"message"`
	Hostname    string                 `json:"hostname"`
	RequestID   string                 `json:"request_id,omitempty"`
	RideID      string                 `json:"ride_id,omitempty"`
	DriverID    string                 `json:"driver_id,omitempty"`
	PassengerID string                 `json:"passenger_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	Error       *ErrorDetails          `json:"error,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

// ErrorDetails contains error information
type ErrorDetails struct {
	Message string `json:"msg"`
	Stack   string `json:"stack,omitempty"`
}

// NewLogger creates a new structured JSON logger
func NewLogger(serviceName string) *Logger {
	hostname, _ := os.Hostname()
	return &Logger{
		service:  serviceName,
		hostname: hostname,
	}
}

// New creates a logger with default service name "unknown"
func New() *Logger {
	return NewLogger("unknown")
}

// Info logs an informational message
func (l *Logger) Info(action, message string, fields ...map[string]interface{}) {
	l.log("INFO", action, message, nil, fields...)
}

// Debug logs a debug message
func (l *Logger) Debug(action, message string, fields ...map[string]interface{}) {
	l.log("DEBUG", action, message, nil, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(action, message string, fields ...map[string]interface{}) {
	l.log("WARN", action, message, nil, fields...)
}

// Error logs an error message
func (l *Logger) Error(action, message string, err error, fields ...map[string]interface{}) {
	var errDetails *ErrorDetails
	if err != nil {
		errDetails = &ErrorDetails{
			Message: err.Error(),
		}
	}
	l.log("ERROR", action, message, errDetails, fields...)
}

// Fatal logs a fatal error message and exits
func (l *Logger) Fatal(action, message string, err error, fields ...map[string]interface{}) {
	var errDetails *ErrorDetails
	if err != nil {
		errDetails = &ErrorDetails{
			Message: err.Error(),
		}
	}
	l.log("FATAL", action, message, errDetails, fields...)
	os.Exit(1)
}

// OK logs a success message (alias for Info)
func (l *Logger) OK(action, message string, fields ...map[string]interface{}) {
	l.log("INFO", action, message, nil, fields...)
}

// log is the internal logging function that outputs JSON
func (l *Logger) log(level, action, message string, errDetails *ErrorDetails, fields ...map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Service:   l.service,
		Action:    action,
		Message:   message,
		Hostname:  l.hostname,
		Error:     errDetails,
	}

	// Merge extra fields if provided
	if len(fields) > 0 {
		extraFields := make(map[string]interface{})
		for k, v := range fields[0] {
			switch k {
			case "request_id":
				if s, ok := v.(string); ok {
					entry.RequestID = s
				}
			case "ride_id":
				if s, ok := v.(string); ok {
					entry.RideID = s
				}
			case "driver_id":
				if s, ok := v.(string); ok {
					entry.DriverID = s
				}
			case "passenger_id":
				if s, ok := v.(string); ok {
					entry.PassengerID = s
				}
			case "user_id":
				if s, ok := v.(string); ok {
					entry.UserID = s
				}
			default:
				extraFields[k] = v
			}
		}
		if len(extraFields) > 0 {
			entry.Extra = extraFields
		}
	}

	json.NewEncoder(os.Stdout).Encode(entry)
}

// HTTP logs HTTP request information
func (l *Logger) HTTP(status int, method, path string, fields ...map[string]interface{}) {
	l.Info("http_request", "HTTP request processed", append(fields, map[string]interface{}{
		"status": status,
		"method": method,
		"path":   path,
	})...)
}
