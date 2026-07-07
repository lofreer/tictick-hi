package datasync

import "testing"

func TestRealtimeSyncModeDocumentsRESTPolling(t *testing.T) {
	if RealtimeSyncModeRESTPolling != "rest_polling" {
		t.Fatalf("RealtimeSyncModeRESTPolling = %q, want rest_polling", RealtimeSyncModeRESTPolling)
	}
}
