// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/gomoni/gio/unix"
	"github.com/stretchr/testify/require"
)

// Parallel enables parallel tests only if testing is not verbose
// this prevents the debug logs from being mixed together
func Parallel(t *testing.T) {
	if !testing.Verbose() {
		t.Parallel()
	}
}

// Case is a single test case testing gonix filter
// It contains a pointer (PF) to type implementing the pipe.Filter interface
type Case[F unix.Filter] struct {
	Name     string // Name is test case name
	Input    string // Input is test case input
	Expected string // Expected is what filter is expected to produce
	Filter   F      // Filter is a pointer to type implementing the pipe.Filter
	FromArgs F      // Optional Filter constructed via FromArgs helper, expected to be equal to Filter
}

func RunAll[F unix.Filter](t *testing.T, testCases []Case[F]) {
	t.Helper()

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.Name, func(t *testing.T) {
			Parallel(t)

			var out strings.Builder
			stdio := unix.NewStdio(
				bytes.NewBufferString(tt.Input),
				&out,
				os.Stderr,
			)
			ctx := context.Background()

			var zero F
			if !reflect.DeepEqual(tt.FromArgs, zero) {
				require.Equal(t, tt.FromArgs, tt.Filter)
			}

			// call SetDebug(true) if present and test if verbose
			x := reflect.ValueOf(tt.Filter)
			setDebug := x.MethodByName("SetDebug")
			if setDebug.Kind() == reflect.Func {
				setDebug.Call([]reflect.Value{reflect.ValueOf(testing.Verbose())})
			}

			err := unix.NewLine().Run(ctx, stdio, tt.Filter)
			require.NoError(t, err)
			require.Equal(t, tt.Expected, out.String())
		})
	}
}

var (
	testDataDir  string
	testDataOnce sync.Once
)

// Testdata returns and (absolute) path to internal/test/testdata file
func Testdata(t *testing.T, key string) string {
	t.Helper()
	testDataOnce.Do(func() {
		_, f, _, ok := runtime.Caller(0)
		require.Truef(t, ok, "can't call runtime.Caller")
		testDataDir = filepath.Join(filepath.Dir(f), "testdata")
	})

	path := filepath.Join(
		testDataDir,
		key)
	st, err := os.Stat(path)
	require.NoError(t, err)
	require.True(t, st.Mode().IsRegular())
	return path
}

type IOError struct {
	Reads    [][]byte
	Writes   int
	Err      error
	CloseErr error
}

func (i *IOError) Read(p []byte) (int, error) {
	if len(i.Reads) == 0 {
		if i.Err != nil {
			return 0, i.Err
		}
		return 0, io.EOF
	}
	copy(p, i.Reads[0])
	i.Reads = i.Reads[1:]
	return len(p), nil
}
func (i *IOError) Write(p []byte) (int, error) {
	if i.Writes == 0 {
		return 0, i.Err
	}
	i.Writes -= 1
	return len(p), nil
}
func (i IOError) Close() error {
	return i.CloseErr
}
