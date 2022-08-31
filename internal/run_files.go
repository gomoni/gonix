package internal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"

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
		return l.doOne(ctx, 0, "", l.stdio.Stdout, l.stdio.Stderr)
	}
	for idx, name := range l.files {
		err := l.doOne(ctx, idx, name, l.stdio.Stdout, l.stdio.Stderr)
		if err != nil {
			return err
		}
	}
	return l.asPipeError()
}

type in struct {
	idx  int
	name string
}

type out struct {
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

// DoThreads runs individual tasks concurrently via PMap. Each command writes to the memory buffer
// first, so probably best to be used for a compute intensive operations like cksum is. As it uses
// PMap, outputs are in the same order as inputs.
func (l *RunFiles) DoThreads(ctx context.Context, threads uint) error {
	if threads == 0 {
		threads = uint(runtime.GOMAXPROCS(0))
	}
	if threads == 1 || len(l.files) == 0 {
		return l.Do(ctx)
	}

	one := func(ctx context.Context, in in) (out, error) {
		out := out{
			stdout: bytes.NewBuffer(nil),
			stderr: bytes.NewBuffer(nil),
		}
		err := l.doOne(ctx, in.idx, in.name, out.stdout, out.stderr)
		return out, err
	}

	inputs := make([]in, len(l.files))
	for idx, f := range l.files {
		inputs[idx] = in{idx: idx, name: f}
	}

	outputs, err := PMap(ctx, threads, inputs, one)
	if err != nil {
		return err
	}
	for _, out := range outputs {
		_, err = io.Copy(l.stdio.Stderr, out.stderr)
		if err != nil {
			l.errs = multierror.Append(l.errs, err)
		}
		_, err = io.Copy(l.stdio.Stdout, out.stdout)
		if err != nil {
			l.errs = multierror.Append(l.errs, err)
		}
	}
	return l.asPipeError()
}

func (l *RunFiles) doOne(ctx context.Context, idx int, name string, stdout, stderr io.Writer) error {
	var in io.Reader
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
		Stdout: stdout,
		Stderr: stderr},
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
