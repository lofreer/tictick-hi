package workerlog

import "strings"

func TaskAttrs(taskID string, requestID string, attrs ...any) []any {
	return taskAttrs(taskID, requestID, "", attrs...)
}

func TaskTraceAttrs(taskID string, requestID string, traceparent string, attrs ...any) []any {
	return taskAttrs(taskID, requestID, traceparent, attrs...)
}

func taskAttrs(taskID string, requestID string, traceparent string, attrs ...any) []any {
	result := []any{"task_id", taskID}
	if requestID != "" {
		result = append(result, "request_id", requestID)
	}
	if traceID := traceIDFromTraceParent(traceparent); traceID != "" {
		result = append(result, "trace_id", traceID)
	}
	return append(result, attrs...)
}

func traceIDFromTraceParent(traceparent string) string {
	value := strings.ToLower(strings.TrimSpace(traceparent))
	if len(value) != 55 || value[:2] != "00" || value[2] != '-' || value[35] != '-' || value[52] != '-' {
		return ""
	}
	traceID := value[3:35]
	spanID := value[36:52]
	if !isLowerHex(traceID) || !isLowerHex(spanID) || !isLowerHex(value[53:55]) {
		return ""
	}
	if isAllZero(traceID) || isAllZero(spanID) {
		return ""
	}
	return traceID
}

func isLowerHex(value string) bool {
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func isAllZero(value string) bool {
	for _, char := range value {
		if char != '0' {
			return false
		}
	}
	return true
}
