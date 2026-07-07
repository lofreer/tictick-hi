package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewLoggerFromEnvWritesJSONDebugLogs(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "json")
	var output bytes.Buffer

	logger, err := newLoggerFromEnv(&output)
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	logger.Debug("debug message", "component", "test")

	logged := output.String()
	if !strings.Contains(logged, `"level":"DEBUG"`) ||
		!strings.Contains(logged, `"msg":"debug message"`) ||
		!strings.Contains(logged, `"component":"test"`) {
		t.Fatalf("unexpected json log output: %s", logged)
	}
}

func TestNewLoggerFromEnvDefaultsToInfoText(t *testing.T) {
	clearCommandEnv(t)
	var output bytes.Buffer

	logger, err := newLoggerFromEnv(&output)
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	logger.Debug("hidden")
	logger.Info("visible")

	logged := output.String()
	if strings.Contains(logged, "hidden") || !strings.Contains(logged, "visible") || !strings.Contains(logged, "level=INFO") {
		t.Fatalf("unexpected text log output: %s", logged)
	}
}

func TestNewLoggerFromEnvRejectsInvalidLevelWithoutEchoingValue(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("LOG_LEVEL", "stage8_config_secret")

	_, err := newLoggerFromEnv(&bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "LOG_LEVEL") {
		t.Fatalf("expected LOG_LEVEL error, got %v", err)
	}
	if strings.Contains(err.Error(), "stage8_config_secret") {
		t.Fatalf("error leaked invalid log level value: %v", err)
	}
}

func TestNewLoggerFromEnvRejectsInvalidFormatWithoutEchoingValue(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("LOG_FORMAT", "stage8_config_secret")

	_, err := newLoggerFromEnv(&bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "LOG_FORMAT") {
		t.Fatalf("expected LOG_FORMAT error, got %v", err)
	}
	if strings.Contains(err.Error(), "stage8_config_secret") {
		t.Fatalf("error leaked invalid log format value: %v", err)
	}
}
