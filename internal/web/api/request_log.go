package api

import (
	"log/slog"
	"net/http"
	"time"
)

type accessLogResponseWriter struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (writer *accessLogResponseWriter) WriteHeader(statusCode int) {
	if writer.statusCode != 0 {
		return
	}
	writer.statusCode = statusCode
	writer.ResponseWriter.WriteHeader(statusCode)
}

func (writer *accessLogResponseWriter) Write(data []byte) (int, error) {
	if writer.statusCode == 0 {
		writer.WriteHeader(http.StatusOK)
	}
	written, err := writer.ResponseWriter.Write(data)
	writer.bytes += written
	return written, err
}

func (writer *accessLogResponseWriter) StatusCode() int {
	if writer.statusCode == 0 {
		return http.StatusOK
	}
	return writer.statusCode
}

func logHTTPRequest(r *http.Request, writer *accessLogResponseWriter, startedAt time.Time) {
	slog.Info(
		"http request",
		"request_id", RequestIDFromContext(r.Context()),
		"method", r.Method,
		"path", r.URL.Path,
		"status", writer.StatusCode(),
		"bytes", writer.bytes,
		"duration_ms", time.Since(startedAt).Milliseconds(),
	)
}
