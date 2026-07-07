package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
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

	options := &slog.HandlerOptions{Level: level}
	switch format {
	case "text":
		return slog.New(slog.NewTextHandler(output, options)), nil
	case "json":
		return slog.New(slog.NewJSONHandler(output, options)), nil
	default:
		return nil, fmt.Errorf("LOG_FORMAT must be one of text, json")
	}
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
