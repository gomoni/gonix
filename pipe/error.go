package pipe

import (
	"errors"
	"fmt"
)

// Error contains exit code and error - simulates the cli exit codes
type Error struct {
	Code uint8
	Err  error
}

func (e Error) Error() string {
	return fmt.Sprintf("Error{Code: %d, Err: %+v}", e.Code, e.Err)
}

// NewError returns a new error with code and error inside
func NewError(code uint8, err error) Error {
	return Error{Code: code, Err: err}
}

// NewErrorf returns formatted new error with code and error inside
func NewErrorf(code uint8, format string, args ...any) Error {
	return Error{Code: code, Err: fmt.Errorf(format, args...)}
}

// AsError unpacks error into Error. If it can't be unpacked, it assigns code 255
func AsError(x error) Error {
	var err Error
	if !errors.As(x, &err) {
		return Error{Code: 255, Err: x}
	}
	return err
}
