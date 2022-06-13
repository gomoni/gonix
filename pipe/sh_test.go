// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package pipe

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSh(t *testing.T) {
	builtins := Builtins{
		"a": func([]string) (Filter, error) {
			return filter{msg: "a\n"}, nil
		},
		"b": func([]string) (Filter, error) {
			return filter{msg: "b\n"}, nil
		},
	}
	splitfn := func(_ string) ([]string, error) {
		return []string{"a", "|", "b"}, nil
	}

	sh := NewSh(builtins, splitfn)
	ctx := context.Background()
	var out strings.Builder
	stdio := Stdio{
		Stdin:  io.NopCloser(strings.NewReader("")),
		Stdout: &out,
		Stderr: io.Discard,
	}

	const cmdline = `a | b`

	t.Run("parse", func(t *testing.T) {
		filters, err := sh.Parse(cmdline)
		require.NoError(t, err)
		require.Len(t, filters, 2)
	})

	t.Run("run", func(t *testing.T) {
		err := sh.Run(ctx, stdio, `a|b`)
		require.NoError(t, err)
		require.Equal(t, "a\nb\n", out.String())
	})
}

func TestShPipefail(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		pipefail bool
		expError bool
	}{
		{
			"no pipefail",
			false,
			false,
		},
		{
			"pipefail",
			true,
			true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {

			a := errFilter{err: NewErrorf(42, "error: a")}
			b := filter{msg: "b\n"}

			var builtins Builtins = Builtins{
				"a": func(_ []string) (Filter, error) { return a, nil },
				"b": func(_ []string) (Filter, error) { return b, nil },
			}
			splitFn := func(_ string) ([]string, error) { return []string{"a", "|", "b"}, nil }

			sh := NewSh(builtins, splitFn).Pipefail(tt.pipefail)

			var out strings.Builder
			stdio := Stdio{
				Stdin:  io.NopCloser(strings.NewReader("<stdio>\n")),
				Stdout: &out,
				Stderr: io.Discard,
			}

			ctx := context.Background()
			err := sh.Run(ctx, stdio, `a|b`)
			require.Error(t, err)
			e := AsError(err)
			if !tt.expError {
				require.EqualValues(t, 0, e.Code)
			} else {
				require.EqualValues(t, 42, e.Code)
			}
			require.EqualError(t, e.Err, "1 error occurred:\n\t* Error{Code: 42, Err: error: a}\n\n")
			require.Equal(t, "b\n", out.String())

		})
	}
}

type filter struct {
	msg string
}

func (f filter) Run(ctx context.Context, stdio Stdio) error {

	if ctx.Err() != nil {
		return ctx.Err()
	}

	_, err := io.Copy(stdio.Stdout, stdio.Stdin)
	if err != nil {
		return err
	}

	_, err = stdio.Stdout.Write([]byte(f.msg))
	return err
}
