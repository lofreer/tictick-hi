package data

import "errors"

var ErrNotFound = errors.New("not found")

var ErrUnauthorized = errors.New("unauthorized")

var ErrInvalidState = errors.New("invalid state")
