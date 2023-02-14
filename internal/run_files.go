package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/gomoni/gio/pipe"
	"github.com/gomoni/gio/unix"
)

// RunFiles is a helper run gonix commands with inputs from more files
// failure in file opening does not break the loop, but returns exit code 1
// "" or "-" are treated as stdin
type RunFiles struct {
	files []string
	errs  error
	stdio unix.StandardIO
	fun   func(context.Context, unix.StandardIO, int, string) error
}

func NewRunFiles(files []string, stdio unix.StandardIO, fun func(context.Context, unix.StandardIO, int, string) error) RunFiles {
	return RunFiles{
		files: files,
		stdio: stdio,
		errs:  nil,
		fun:   fun,
	}
}

func (l RunFiles) Do(ctx context.Context) error {
	errs := make([]error, 0, len(l.files))
	if len(l.files) == 0 {
		return l.doOne(ctx, 0, "", l.stdio.Stdout(), l.stdio.Stderr(), &errs)
	}
	for idx, name := range l.files {
		err := l.doOne(ctx, idx, name, l.stdio.Stdout(), l.stdio.Stderr(), &errs)
		if err != nil {
			return err
		}
	}
	return asPipeError(errs)
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
func (l RunFiles) DoThreads(ctx context.Context, threads uint) error {
	if threads == 0 {
		threads = uint(runtime.GOMAXPROCS(0))
	}
	if threads == 1 || len(l.files) == 0 {
		return l.Do(ctx)
	}

	errs := make([]error, 0, len(l.files))
	one := func(ctx context.Context, in in) (out, error) {
		out := out{
			stdout: bytes.NewBuffer(nil),
			stderr: bytes.NewBuffer(nil),
		}
		err := l.doOne(ctx, in.idx, in.name, out.stdout, out.stderr, &errs)
		if err != nil {
			errs = append(errs, err)
		}
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
		_, err = io.Copy(l.stdio.Stderr(), out.stderr)
		if err != nil {
			errs = append(errs, err)
		}
		_, err = io.Copy(l.stdio.Stdout(), out.stdout)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return asPipeError(errs)
}

func (l RunFiles) doOne(ctx context.Context, idx int, name string, stdout, stderr io.Writer, errsp *[]error) error {
	var in io.Reader
	if name == "" || name == "-" {
		in = l.stdio.Stdin()
	} else {
		f, err := os.Open(name)
		if err != nil {
			fmt.Fprintf(l.stdio.Stderr(), "%s\n", err)
			*errsp = append(*errsp, err)
			return nil
		}
		defer f.Close()
		in = f
	}
	return l.fun(ctx, unix.NewStdio(
		in,
		stdout,
		stderr),
		idx,
		name,
	)
}

func asPipeError(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	err := pipe.NewError(1, errors.Join(errs...))
	return err
}
