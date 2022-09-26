// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package cat_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	. "github.com/gomoni/gonix/cat"
	"github.com/gomoni/gonix/internal/test"
	"github.com/gomoni/gonix/pipe"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestCat(t *testing.T) {
	test.Parallel(t)

	testCases := []test.Case[Cat, *Cat]{
		{
			Name:     "cat",
			Filter:   New(),
			Input:    "three\nsmall\npigs\n",
			Expected: "three\nsmall\npigs\n",
		},
		// --show-all
		{
			Name:     "cat -b",
			Filter:   New().ShowNumber(NonBlank),
			Input:    "three\n\n\nsmall\npigs\n",
			Expected: "     1\tthree\n\n\n     2\tsmall\n     3\tpigs\n",
		},
		// -e   equivalent to -vE
		{
			Name:     "cat -E",
			Filter:   New().ShowEnds(true),
			Input:    "three\nsmall\npigs\n",
			Expected: "three$\nsmall$\npigs$\n",
		},
		{
			Name:     "cat -n",
			Filter:   New().ShowNumber(All),
			Input:    "three\nsmall\npigs\n",
			Expected: "     1\tthree\n     2\tsmall\n     3\tpigs\n",
		},
		{
			Name:     "cat -s",
			Filter:   New().SqueezeBlanks(true),
			Input:    "three\n\n\nsmall\npigs\n",
			Expected: "three\n\nsmall\npigs\n",
		},
		// -t     equivalent to -vT
		{
			Name:     "cat -T",
			Filter:   New().ShowTabs(true),
			Input:    "\tthree\nsmall\t\npi\tgs\n",
			Expected: "^Ithree\nsmall^I\npi^Igs\n",
		},
		{
			Name:     "cat -ET",
			Filter:   New().ShowEnds(true).ShowTabs(true),
			Input:    "\tthree\nsmall\t\npi\tgs\n",
			Expected: "^Ithree$\nsmall^I$\npi^Igs$\n",
		},
		{
			Name:     "cat -A",
			Filter:   New().ShowNonPrinting(true).ShowEnds(true).ShowTabs(true),
			Input:    string(rune(127)) + "\tthree\nsmall\t\npi\tgs\n",
			Expected: "^?^Ithree$\nsmall^I$\npi^Igs$\n",
		},
	}

	test.RunAll(t, testCases)
}

// TODO: think about how this can be more generic
func TestError(t *testing.T) {
	ctx := context.Background()

	t.Run("FromArgs error", func(t *testing.T) {
		_, err := New().FromArgs([]string{"-x"})
		require.Error(t, err)
		e := pipe.AsError(err)
		require.EqualValues(t, 1, e.Code)
	})
	t.Run("read error", func(t *testing.T) {
		cat := New()
		stdio := pipe.Stdio{
			Stdin:  &test.IOError{Err: fmt.Errorf("stdin crashed")},
			Stdout: io.Discard,
			Stderr: io.Discard,
		}
		err := cat.Run(ctx, stdio)
		require.Error(t, err)
		e := pipe.AsError(err)
		require.EqualValues(t, 1, e.Code)
		require.EqualError(t, e.Err, "cat: fail to run: stdin crashed")
	})
	t.Run("write error", func(t *testing.T) {
		cat := New()
		stdio := pipe.Stdio{
			Stdin:  &test.IOError{Reads: [][]byte{{0xd, 0xe, 0xa, 0xd, 0xb, 0xe, 0xe, 0xe, 0xf}}},
			Stdout: &test.IOError{Err: fmt.Errorf("stdout crashed")},
			Stderr: io.Discard,
		}
		err := cat.Run(ctx, stdio)
		require.Error(t, err)
		e := pipe.AsError(err)
		require.EqualValues(t, 1, e.Code)
		require.EqualError(t, e.Err, "cat: fail to run: stdout crashed")
	})
	t.Run("close error", func(t *testing.T) {
		t.Skipf("TODO: must redefine this ReadCloser usage")
		cat := New()
		stdio := pipe.Stdio{
			Stdin: &test.IOError{
				Reads:    [][]byte{{0xd, 0xe, 0xa, 0xd, 0xb, 0xe, 0xe, 0xe, 0xf}},
				CloseErr: fmt.Errorf("close crashed"),
			},
			Stdout: &test.IOError{Writes: 1},
			Stderr: io.Discard,
		}
		err := cat.Run(ctx, stdio)
		require.Error(t, err)
		e := pipe.AsError(err)
		require.EqualValues(t, 1, e.Code)
		require.EqualError(t, e.Err, "cat: fail to run: close crashed")
	})
	t.Run("file not found", func(t *testing.T) {
		// main.c is guaranteed to not exists, because this is pure Go and compiler
		// will complain otherwise
		// package github.com/gomoni/gonix/cat: C source files not allowed when not using cgo or SWIG: main.c
		cat := New().Files("main.c")
		stdio := pipe.Stdio{
			Stdin:  io.NopCloser(strings.NewReader("")),
			Stdout: io.Discard,
			Stderr: io.Discard,
		}
		err := cat.Run(ctx, stdio)
		require.Error(t, err)
		e := pipe.AsError(err)
		require.EqualValues(t, 1, e.Code)
		require.Contains(t, e.Err.Error(), "main.c")
	})

}
