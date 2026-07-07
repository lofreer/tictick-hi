package data

import "errors"

var ErrNotFound = errors.New("not found")

var ErrUnauthorized = errors.New("unauthorized")

var ErrInvalidState = errors.New("invalid state")

type ErrorCode string

const (
	ErrorCodeMarketInstrumentNotActive      ErrorCode = "market_instrument_not_active"
	ErrorCodeDataSyncRetryRequiresFailed    ErrorCode = "data_sync_retry_requires_failed"
	ErrorCodeDataSyncCommandInvalidState    ErrorCode = "data_sync_command_invalid_state"
	ErrorCodeTradingTaskCommandInvalidState ErrorCode = "trading_task_command_invalid_state"
	ErrorCodeOperatorLastEnabledRequired    ErrorCode = "operator_last_enabled_required"
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

func MarketInstrumentNotActiveError() error {
	return &DomainError{
		Code:    ErrorCodeMarketInstrumentNotActive,
		Message: "market instrument is not active in catalog",
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

func TradingTaskCommandInvalidStateError() error {
	return &DomainError{
		Code:    ErrorCodeTradingTaskCommandInvalidState,
		Message: "trading task status cannot be changed by this command",
		Cause:   ErrInvalidState,
	}
}

func OperatorLastEnabledError() error {
	return &DomainError{
		Code:    ErrorCodeOperatorLastEnabledRequired,
		Message: "at least one operator must remain enabled",
		Cause:   ErrInvalidState,
	}
}
