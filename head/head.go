// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package head

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"

	"github.com/benhoyt/goawk/parser"
	"github.com/gomoni/gonix/internal"
	"github.com/gomoni/gonix/internal/dbg"
	"github.com/gomoni/gonix/pipe"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/pflag"

	_ "embed"
)

//go:embed head.awk
var headAwk []byte

//go:embed head_negative.awk
var headNegative []byte

type Head struct {
	debug bool
	lines int
	files []string
}

func New() *Head {
	return &Head{
		debug: false,
		lines: 10}
}

func (c *Head) FromArgs(argv []string) (*Head, error) {
	flag := pflag.FlagSet{}

	an := flag.String("n", "10", "print at least n lines, -n means everything except last n lines")

	err := flag.Parse(argv)
	if err != nil {
		return nil, pipe.NewErrorf(1, "head: parsing failed: %w", err)
	}
	c.files = flag.Args()

	n, err := internal.ParseByte(*an)
	if err != nil {
		return nil, pipe.NewErrorf(1, "head: %w", err)
	}
	if float64(n) > float64(1<<63) {
		return nil, pipe.NewErrorf(1, "head: size overflow %f", math.Round(float64(n)))
	}
	// TODO: deal with more than int64 lines?
	c.lines = int(math.Round(float64(n)))

	return c, nil
}

// Files are input files, where - denotes stdin
func (c *Head) Files(f ...string) *Head {
	c.files = append(c.files, f...)
	return c
}

func (c *Head) Lines(lines int) *Head {
	c.lines = lines
	return c
}

func (c *Head) SetDebug(debug bool) *Head {
	c.debug = debug
	return c
}

func (c Head) Run(ctx context.Context, stdio pipe.Stdio) error {
	debug := dbg.Logger(c.debug, "cat", stdio.Stderr)
	if c.lines == 0 {
		return nil
	}
	var src []byte
	var lines int
	if c.lines > 0 {
		lines = c.lines
		src = headAwk
	} else {
		lines = -1 * c.lines
		src = headNegative
	}

	debug.Printf("head: src=`%s`", src)
	debug.Printf("head: lines=%d", lines)

	prog, err := parser.ParseProgram([]byte(src), nil)
	if err != nil {
		return err
	}
	awk := internal.NewAWK(prog)
	awk.SetVariable("lines", strconv.Itoa(lines))

	var head func(context.Context, pipe.Stdio, int, string) error
	if len(c.files) <= 1 {
		head = func(ctx context.Context, stdio pipe.Stdio, _ int, _ string) error {
			err := awk.Run(ctx, stdio)
			if err != nil {
				return pipe.NewError(1, fmt.Errorf("head: fail to run: %w", err))
			}
			return nil
		}
	} else {
		head = func(ctx context.Context, stdio pipe.Stdio, _ int, name string) error {
			fmt.Fprintf(stdio.Stdout, "==> %s <==\n", name)
			err := awk.Run(ctx, stdio)
			if err != nil {
				return pipe.NewError(1, fmt.Errorf("head: fail to run: %w", err))
			}
			fmt.Fprintln(stdio.Stdout)
			return nil
		}
	}

	runFiles := newRunFiles(
		c.files,
		stdio,
		head,
	)
	err = runFiles.do(ctx)
	if err != nil {
		return err
	}
	return runFiles.AsPipeError()
}

// runFiles is a helper run gonix commands with inputs from more files
// failure in file opening does not break the loop, but returns exit code 1
// - or empty name are treated as stdin
type runFiles struct {
	files []string
	errs  error
	stdio pipe.Stdio
	fun   func(context.Context, pipe.Stdio, int, string) error
}

func newRunFiles(files []string, stdio pipe.Stdio, fun func(context.Context, pipe.Stdio, int, string) error) runFiles {
	return runFiles{
		files: files,
		stdio: stdio,
		errs:  nil,
		fun:   fun,
	}
}

func (l *runFiles) do(ctx context.Context) error {
	if len(l.files) == 0 {
		return l.doOne(ctx, 0, "")
	}
	for idx, name := range l.files {
		err := l.doOne(ctx, idx, name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *runFiles) doOne(ctx context.Context, idx int, name string) error {
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

func (l runFiles) AsPipeError() error {
	if l.errs != nil {
		return pipe.NewError(1, l.errs)
	}
	return nil
}
