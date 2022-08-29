package pipe

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPipe(t *testing.T) {
	if !testing.Verbose() {
		t.Parallel()
	}

	pipe := New()

	a := filter{msg: "a\n"}
	b := filter{msg: "b\n"}

	var out strings.Builder
	stdio := Stdio{
		Stdin:  io.NopCloser(strings.NewReader("<stdio>\n")),
		Stdout: &out,
		Stderr: io.Discard,
	}

	ctx := context.Background()
	err := pipe.Run(ctx, stdio, a, b)
	require.NoError(t, err)
	require.Equal(t, "<stdio>\na\nb\n", out.String())
}

func TestPipefail(t *testing.T) {
	if !testing.Verbose() {
		t.Parallel()
	}

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
			if !testing.Verbose() {
				t.Parallel()
			}

			a := errFilter{err: NewErrorf(42, "error: a")}
			b := filter{msg: "b\n"}

			var out strings.Builder
			stdio := Stdio{
				Stdin:  io.NopCloser(strings.NewReader("<stdio>\n")),
				Stdout: &out,
				Stderr: os.Stderr,
			}

			run := New().SetDebug(true).Pipefail(tt.pipefail).Run

			ctx := context.Background()
			err := run(ctx, stdio, a, b)
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

type errFilter struct {
	err error
}

func (f errFilter) Run(_ context.Context, stdio Stdio) error {
	return f.err
}

type mockCloser struct {
	closeErr error
}

func (m mockCloser) Close() error {
	return m.closeErr
}
func (mockCloser) Read(_ []byte) (int, error) {
	panic("not implemented")
}
func (mockCloser) Write(_ []byte) (int, error) {
	panic("not implemented")
}

func TestAllwaysCloser(t *testing.T) {
	t.Parallel()

	m := mockCloser{closeErr: errors.New("Close() called")}

	var r io.Reader = m
	err := closeReader(r)
	require.Error(t, err)
	require.EqualError(t, err, "Close() called")

	var w io.Writer = m
	err = closeWriter(w)
	require.Error(t, err)
	require.EqualError(t, err, "Close() called")

	err = closeReader(io.NopCloser(r))
	require.NoError(t, err)
}
