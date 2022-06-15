// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package pipe

import (
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
)

const (
	// NotExecutable is from POSIX and indicates tool was found, but not executable
	NotExecutable = 126
	// NotFound is from POSIX and indicate a tool was not found
	NotFound = 127
	// UnknownError is a code used for unpacking other than pipe.Error
	UnknownError = 250
)

var (
	ErrBuiltinNotFound = errors.New("builtin not found")
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

// AsError unpacks error into Error. If it can't be unpacked, it assigns code 250
// error fs.ErrPermission will get code NotExecutable (126)
// error exec.ErrNotFound will get code NotFound (code 127)
func AsError(x error) Error {
	if errors.Is(x, exec.ErrNotFound) {
		return Error{Code: NotFound, Err: x}
	}

	var fsErr *fs.PathError
	if errors.As(x, &fsErr) {
		if fsErr.Op == "fork/exec" && errors.Is(fsErr.Err, fs.ErrPermission) {
			return Error{Code: NotExecutable, Err: x}
		}
	}

	var err Error
	if !errors.As(x, &err) {
		return Error{Code: UnknownError, Err: x}
	}
	return err
}
