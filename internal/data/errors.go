package data

import "errors"

var ErrNotFound = errors.New("not found")

var ErrUnauthorized = errors.New("unauthorized")

var ErrInvalidState = errors.New("invalid state")

type ErrorCode string

const (
	ErrorCodeDataSyncRetryRequiresFailed ErrorCode = "data_sync_retry_requires_failed"
	ErrorCodeDataSyncCommandInvalidState ErrorCode = "data_sync_command_invalid_state"
)

type DomainError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (err *DomainError) Error() string {
	return err.Message
}

func (err *DomainError) Unwrap() error {
	return err.Cause
}

func DomainErrorCode(err error) (ErrorCode, bool) {
	var domainErr *DomainError
	if errors.As(err, &domainErr) && domainErr.Code != "" {
		return domainErr.Code, true
	}
	return "", false
}

func DataSyncRetryRequiresFailedError() error {
	return &DomainError{
		Code:    ErrorCodeDataSyncRetryRequiresFailed,
		Message: "data sync task must be failed before retry",
		Cause:   ErrInvalidState,
	}
}

func DataSyncCommandInvalidStateError() error {
	return &DomainError{
		Code:    ErrorCodeDataSyncCommandInvalidState,
		Message: "data sync task status cannot be changed by this command",
		Cause:   ErrInvalidState,
	}
}
