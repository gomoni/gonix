package internal

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/gomoni/gonix/pipe"
	"github.com/hashicorp/go-multierror"
)

// RunFiles is a helper run gonix commands with inputs from more files
// failure in file opening does not break the loop, but returns exit code 1
// "" or "-" are treated as stdin
type RunFiles struct {
	files []string
	errs  error
	stdio pipe.Stdio
	fun   func(context.Context, pipe.Stdio, int, string) error
}

func NewRunFiles(files []string, stdio pipe.Stdio, fun func(context.Context, pipe.Stdio, int, string) error) RunFiles {
	return RunFiles{
		files: files,
		stdio: stdio,
		errs:  nil,
		fun:   fun,
	}
}

func (l *RunFiles) Do(ctx context.Context) error {
	if len(l.files) == 0 {
		return l.doOne(ctx, 0, "")
	}
	for idx, name := range l.files {
		err := l.doOne(ctx, idx, name)
		if err != nil {
			return err
		}
	}
	return l.asPipeError()
}

func (l *RunFiles) doOne(ctx context.Context, idx int, name string) error {
	var in io.ReadCloser
	if name == "" || name == "-" {
		in = l.stdio.Stdin
	} else {
		f, err := os.Open(name)
		if err != nil {
			fmt.Fprintf(l.stdio.Stderr, "%s\n", err)
			l.errs = multierror.Append(l.errs, err)
			return nil
		}
		defer f.Close()
		in = f
	}
	return l.fun(ctx, pipe.Stdio{
		Stdin:  in,
		Stdout: l.stdio.Stdout,
		Stderr: l.stdio.Stderr},
		idx,
		name,
	)
}

func (l RunFiles) asPipeError() error {
	if l.errs != nil {
		return pipe.NewError(1, l.errs)
	}
	return nil
}
