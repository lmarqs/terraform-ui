package sdk

import "errors"

// ErrReadOnly is returned by mutating operations on a read-only service.
var ErrReadOnly = errors.New("operation not available in read-only mode")
