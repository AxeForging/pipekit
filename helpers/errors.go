package helpers

import "errors"

var (
	ErrFileNotFound    = errors.New("file not found")
	ErrInvalidInput    = errors.New("invalid input")
	ErrMissingRequired = errors.New("missing required argument")
	ErrAssertionFailed = errors.New("assertion failed")
	ErrTimeout         = errors.New("timeout exceeded")
	ErrCommandFailed   = errors.New("command failed")
)
