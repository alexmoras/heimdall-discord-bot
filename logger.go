package main

import (
	"log"
	"strings"
)

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelError LogLevel = iota
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

var (
	currentLogLevel LogLevel = LogLevelInfo
	logLevelNames   = map[LogLevel]string{
		LogLevelError: "ERROR",
		LogLevelWarn:  "WARN",
		LogLevelInfo:  "INFO",
		LogLevelDebug: "DEBUG",
	}
)

// InitLogger initializes the logging system based on config
func InitLogger(logLevelStr string) {
	// Default to INFO if not specified
	if logLevelStr == "" {
		logLevelStr = "INFO"
	}

	switch strings.ToUpper(logLevelStr) {
	case "ERROR":
		currentLogLevel = LogLevelError
	case "WARN", "WARNING":
		currentLogLevel = LogLevelWarn
	case "INFO":
		currentLogLevel = LogLevelInfo
	case "DEBUG":
		currentLogLevel = LogLevelDebug
	default:
		currentLogLevel = LogLevelInfo // Default to INFO
		log.Printf("⚠️  Unknown log level '%s', defaulting to INFO", logLevelStr)
	}

	if currentLogLevel == LogLevelDebug {
		log.Println("🔍 Debug logging enabled")
	}
}

// LogError logs error messages (always shown)
func LogError(format string, v ...interface{}) {
	if currentLogLevel >= LogLevelError {
		log.Printf("❌ [ERROR] "+format, v...)
	}
}

// LogWarn logs warning messages
func LogWarn(format string, v ...interface{}) {
	if currentLogLevel >= LogLevelWarn {
		log.Printf("⚠️  [WARN] "+format, v...)
	}
}

// LogInfo logs informational messages
func LogInfo(format string, v ...interface{}) {
	if currentLogLevel >= LogLevelInfo {
		log.Printf("ℹ️  [INFO] "+format, v...)
	}
}

// LogDebug logs debug messages (only in debug mode)
func LogDebug(format string, v ...interface{}) {
	if currentLogLevel >= LogLevelDebug {
		log.Printf("🔍 [DEBUG] "+format, v...)
	}
}

// LogSuccess logs success messages (same level as info)
func LogSuccess(format string, v ...interface{}) {
	if currentLogLevel >= LogLevelInfo {
		log.Printf("✅ [SUCCESS] "+format, v...)
	}
}

// GetLogLevel returns the current log level as a string
func GetLogLevel() string {
	return logLevelNames[currentLogLevel]
}