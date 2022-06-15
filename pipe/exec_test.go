// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package pipe

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExec(t *testing.T) {
	e := NewExec(exec.Command("go", "version"))
	var out strings.Builder
	stdio := Stdio{
		Stdin:  os.Stdin,
		Stdout: &out,
		Stderr: os.Stderr,
	}
	ctx := context.Background()
	err := e.Run(ctx, stdio)
	require.NoError(t, err)
}

func TestExecNotFound(t *testing.T) {
	ex := NewExec(exec.Command("xgox", "version"))
	var out strings.Builder
	stdio := Stdio{
		Stdin:  os.Stdin,
		Stdout: &out,
		Stderr: os.Stderr,
	}
	ctx := context.Background()
	err := ex.Run(ctx, stdio)
	require.Error(t, err)
	require.EqualError(t, err, `exec: "xgox": executable file not found in $PATH`)
	e := AsError(err)
	require.EqualValues(t, NotFound, e.Code)
	require.EqualError(t, e.Err, `exec: "xgox": executable file not found in $PATH`)
}

func TestExecNotExecutable(t *testing.T) {
	ex := NewExec(exec.Command("/dev/null", "version"))
	var out strings.Builder
	stdio := Stdio{
		Stdin:  os.Stdin,
		Stdout: &out,
		Stderr: os.Stderr,
	}
	ctx := context.Background()
	err := ex.Run(ctx, stdio)
	require.Error(t, err)

	e := AsError(err)
	require.EqualValues(t, NotExecutable, e.Code)
	require.EqualError(t, err, "fork/exec /dev/null: permission denied")
}

func TestEnviron(t *testing.T) {

	e := DuplicateEnviron()
	require.ElementsMatch(t, e.Environ(), os.Environ())

	e = NewEnviron("USER", "HOME", "$SHELL")
	require.LessOrEqual(t, len(e.Environ()), 3)

	e = NewEnviron()
	e.Set("FOO", "BAR")
	require.Len(t, e.Environ(), 1)
	e.Set("BAR", "BAZ")
	require.Len(t, e.Environ(), 2)
	e.Unset("NONE")
	require.Len(t, e.Environ(), 2)
	e.Unset("FOO")
	require.Len(t, e.Environ(), 1)

}
