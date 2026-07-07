package workerlog

func TaskAttrs(taskID string, requestID string, attrs ...any) []any {
	result := []any{"task_id", taskID}
	if requestID != "" {
		result = append(result, "request_id", requestID)
	}
	return append(result, attrs...)
}
