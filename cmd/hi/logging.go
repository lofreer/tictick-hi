package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

const (
	generatedCorrelationIDBytes = 16
	generatedTraceIDBytes       = 16
	generatedSpanIDBytes        = 8
)

func configureLoggerFromEnv() error {
	logger, err := newLoggerFromEnv(os.Stderr)
	if err != nil {
		return err
	}
	slog.SetDefault(logger)
	return nil
}

func newLoggerFromEnv(output io.Writer) (*slog.Logger, error) {
	level, err := logLevelFromEnv()
	if err != nil {
		return nil, err
	}
	format, err := logFormatFromEnv()
	if err != nil {
		return nil, err
	}
	correlationID, err := logCorrelationIDFromEnv()
	if err != nil {
		return nil, err
	}
	traceparent, err := logTraceParentFromEnv()
	if err != nil {
		return nil, err
	}

	options := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	switch format {
	case "text":
		handler = slog.NewTextHandler(output, options)
	case "json":
		handler = slog.NewJSONHandler(output, options)
	default:
		return nil, fmt.Errorf("LOG_FORMAT must be one of text, json")
	}
	return slog.New(handler).With(
		"correlation_id", correlationID,
		"run_traceparent", traceparent,
		"run_trace_id", traceIDFromTraceParent(traceparent),
	), nil
}

func logLevelFromEnv() (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) {
	case "":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error")
	}
}

func logFormatFromEnv() (string, error) {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT"))) {
	case "", "text":
		return "text", nil
	case "json":
		return "json", nil
	default:
		return "", fmt.Errorf("LOG_FORMAT must be one of text, json")
	}
}

func logCorrelationIDFromEnv() (string, error) {
	value := strings.TrimSpace(os.Getenv("LOG_CORRELATION_ID"))
	if value == "" {
		return generateCorrelationID()
	}
	if !isValidCorrelationID(value) {
		return "", fmt.Errorf("LOG_CORRELATION_ID must be 8 to 128 characters using letters, digits, dot, underscore, colon, or dash")
	}
	return value, nil
}

func logTraceParentFromEnv() (string, error) {
	value := strings.TrimSpace(os.Getenv("LOG_TRACEPARENT"))
	if value == "" {
		return generateTraceParent()
	}
	if !isValidTraceParent(value) {
		return "", fmt.Errorf("LOG_TRACEPARENT must be a valid W3C traceparent")
	}
	return strings.ToLower(value), nil
}

func generateCorrelationID() (string, error) {
	data := make([]byte, generatedCorrelationIDBytes)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("generate LOG_CORRELATION_ID: %w", err)
	}
	return hex.EncodeToString(data), nil
}

func generateTraceParent() (string, error) {
	traceID, err := randomNonZeroHex(generatedTraceIDBytes)
	if err != nil {
		return "", fmt.Errorf("generate LOG_TRACEPARENT trace id: %w", err)
	}
	spanID, err := randomNonZeroHex(generatedSpanIDBytes)
	if err != nil {
		return "", fmt.Errorf("generate LOG_TRACEPARENT span id: %w", err)
	}
	return "00-" + traceID + "-" + spanID + "-00", nil
}

func randomNonZeroHex(size int) (string, error) {
	for {
		data := make([]byte, size)
		if _, err := rand.Read(data); err != nil {
			return "", err
		}
		value := hex.EncodeToString(data)
		if !isAllZeroHex(value) {
			return value, nil
		}
	}
}

func isValidCorrelationID(value string) bool {
	if len(value) < 8 || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if char >= 'a' && char <= 'z' {
			continue
		}
		if char >= 'A' && char <= 'Z' {
			continue
		}
		if char >= '0' && char <= '9' {
			continue
		}
		switch char {
		case '.', '_', ':', '-':
			continue
		default:
			return false
		}
	}
	return true
}

func isValidTraceParent(value string) bool {
	if len(value) != 55 {
		return false
	}
	if value[2] != '-' || value[35] != '-' || value[52] != '-' {
		return false
	}
	version := value[0:2]
	traceID := value[3:35]
	spanID := value[36:52]
	flags := value[53:55]
	if !strings.EqualFold(version, "00") {
		return false
	}
	if !isHex(version) || !isHex(traceID) || !isHex(spanID) || !isHex(flags) {
		return false
	}
	if isAllZeroHex(traceID) || isAllZeroHex(spanID) {
		return false
	}
	return true
}

func traceIDFromTraceParent(traceparent string) string {
	if len(traceparent) != 55 {
		return ""
	}
	return strings.ToLower(traceparent[3:35])
}

func isHex(value string) bool {
	for _, char := range value {
		if char >= '0' && char <= '9' {
			continue
		}
		if char >= 'a' && char <= 'f' {
			continue
		}
		if char >= 'A' && char <= 'F' {
			continue
		}
		return false
	}
	return true
}

func isAllZeroHex(value string) bool {
	for _, char := range value {
		if char != '0' {
			return false
		}
	}
	return true
}
