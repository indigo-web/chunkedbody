package chunkedbodyparser

import "errors"

var (
	ErrBadRequest = errors.New("bad request")
	ErrTooLarge   = errors.New("chunk is too large")
)
