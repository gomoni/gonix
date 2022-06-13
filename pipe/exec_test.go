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
	e := NewExec(exec.Command("xgox", "version"))
	var out strings.Builder
	stdio := Stdio{
		Stdin:  os.Stdin,
		Stdout: &out,
		Stderr: os.Stderr,
	}
	ctx := context.Background()
	err := e.Run(ctx, stdio)
	require.Error(t, err)
	// TODO: convert to pipe.Error - check the right error codes
	require.EqualError(t, err, `exec: "xgox": executable file not found in $PATH`)
}
