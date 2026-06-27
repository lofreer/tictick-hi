package exchange

import "errors"

type temporaryError struct {
	message string
	err     error
}

func NewTemporaryError(message string, err error) error {
	return temporaryError{message: message, err: err}
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

func IsTemporaryError(err error) bool {
	var temporary interface {
		Temporary() bool
	}
	return errors.As(err, &temporary) && temporary.Temporary()
}
