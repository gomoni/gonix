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

type filter struct {
	msg string
}

func (f filter) Run(_ context.Context, stdio Stdio) error {
	_, err := io.Copy(stdio.Stdout, stdio.Stdin)
	if err != nil {
		return err
	}

	_, err = stdio.Stdout.Write([]byte(f.msg))
	return err
}
