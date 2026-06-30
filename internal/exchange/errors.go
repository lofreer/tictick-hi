package exchange

import (
	"errors"
	"time"
)

type temporaryError struct {
	message    string
	err        error
	retryAfter time.Duration
}

func NewTemporaryError(message string, err error) error {
	return temporaryError{message: message, err: err}
}

func NewTemporaryErrorWithRetryAfter(message string, err error, retryAfter time.Duration) error {
	if retryAfter < 0 {
		retryAfter = 0
	}
	return temporaryError{message: message, err: err, retryAfter: retryAfter}
}

func (err temporaryError) Error() string {
	return err.message
}

func (err temporaryError) Unwrap() error {
	return err.err
}

func (err temporaryError) Temporary() bool {
	return true
}

func (err temporaryError) RetryAfter() (time.Duration, bool) {
	return err.retryAfter, err.retryAfter > 0
}

func IsTemporaryError(err error) bool {
	var temporary interface {
		Temporary() bool
	}
	return errors.As(err, &temporary) && temporary.Temporary()
}

func RetryAfter(err error) (time.Duration, bool) {
	var retryAfter interface {
		RetryAfter() (time.Duration, bool)
	}
	if !errors.As(err, &retryAfter) {
		return 0, false
	}
	return retryAfter.RetryAfter()
}
