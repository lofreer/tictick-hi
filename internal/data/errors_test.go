package data

import (
	"errors"
	"testing"
)

func TestDataSyncDomainErrorsPreserveInvalidStateCause(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code ErrorCode
	}{
		{
			name: "retry requires failed",
			err:  DataSyncRetryRequiresFailedError(),
			code: ErrorCodeDataSyncRetryRequiresFailed,
		},
		{
			name: "market instrument not active",
			err:  MarketInstrumentNotActiveError(),
			code: ErrorCodeMarketInstrumentNotActive,
		},
		{
			name: "command invalid state",
			err:  DataSyncCommandInvalidStateError(),
			code: ErrorCodeDataSyncCommandInvalidState,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !errors.Is(test.err, ErrInvalidState) {
				t.Fatalf("error %v does not unwrap ErrInvalidState", test.err)
			}
			code, ok := DomainErrorCode(test.err)
			if !ok || code != test.code {
				t.Fatalf("DomainErrorCode = %q, %t; want %q, true", code, ok, test.code)
			}
		})
	}
}

func TestOperatorLastEnabledErrorPreservesInvalidStateCause(t *testing.T) {
	err := OperatorLastEnabledError()
	if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("error %v does not unwrap ErrInvalidState", err)
	}
	if code, ok := DomainErrorCode(err); ok || code != "" {
		t.Fatalf("DomainErrorCode = %q, %t; want empty, false", code, ok)
	}
}
