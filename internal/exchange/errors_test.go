package exchange

import (
	"errors"
	"testing"
	"time"
)

func TestTemporaryErrorRetryAfter(t *testing.T) {
	cause := errors.New("429")
	err := NewTemporaryErrorWithRetryAfter("temporary", cause, 12*time.Second)

	if !IsTemporaryError(err) {
		t.Fatal("temporary error should be temporary")
	}
	if !errors.Is(err, cause) {
		t.Fatal("temporary error should unwrap cause")
	}
	retryAfter, ok := RetryAfter(err)
	if !ok || retryAfter != 12*time.Second {
		t.Fatalf("RetryAfter = %s, %t; want 12s, true", retryAfter, ok)
	}
}

func TestTemporaryErrorWithoutRetryAfter(t *testing.T) {
	err := NewTemporaryError("temporary", nil)

	retryAfter, ok := RetryAfter(err)
	if ok || retryAfter != 0 {
		t.Fatalf("RetryAfter = %s, %t; want 0, false", retryAfter, ok)
	}
}
