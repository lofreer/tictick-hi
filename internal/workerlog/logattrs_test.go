package workerlog

import (
	"reflect"
	"testing"
)

func TestTaskAttrsIncludesRequestIDWhenPresent(t *testing.T) {
	got := TaskAttrs("task-1", "request-1", "error", "boom")
	want := []any{"task_id", "task-1", "request_id", "request-1", "error", "boom"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("attrs = %#v, want %#v", got, want)
	}
}

func TestTaskAttrsOmitsEmptyRequestID(t *testing.T) {
	got := TaskAttrs("task-1", "", "error", "boom")
	want := []any{"task_id", "task-1", "error", "boom"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("attrs = %#v, want %#v", got, want)
	}
}

func TestTaskTraceAttrsIncludesTraceIDWhenPresent(t *testing.T) {
	got := TaskTraceAttrs(
		"task-1",
		"request-1",
		"00-4BF92F3577B34DA6A3CE929D0E0E4736-00F067AA0BA902B7-01",
		"error",
		"boom",
	)
	want := []any{
		"task_id", "task-1",
		"request_id", "request-1",
		"trace_id", "4bf92f3577b34da6a3ce929d0e0e4736",
		"error", "boom",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("attrs = %#v, want %#v", got, want)
	}
}

func TestTaskTraceAttrsOmitsInvalidTraceParent(t *testing.T) {
	got := TaskTraceAttrs("task-1", "request-1", "stage8_config_secret", "error", "boom")
	want := []any{"task_id", "task-1", "request_id", "request-1", "error", "boom"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("attrs = %#v, want %#v", got, want)
	}
}

func TestTaskTraceAttrsOmitsAllZeroTraceParent(t *testing.T) {
	got := TaskTraceAttrs(
		"task-1",
		"request-1",
		"00-00000000000000000000000000000000-00f067aa0ba902b7-01",
		"error",
		"boom",
	)
	want := []any{"task_id", "task-1", "request_id", "request-1", "error", "boom"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("attrs = %#v, want %#v", got, want)
	}
}
