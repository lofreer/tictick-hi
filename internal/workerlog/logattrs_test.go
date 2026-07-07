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
