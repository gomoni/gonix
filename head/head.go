// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package head

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/benhoyt/goawk/parser"
	"github.com/gomoni/gio/pipe"
	"github.com/gomoni/gio/unix"
	"github.com/gomoni/gonix/awk"
	"github.com/gomoni/gonix/internal"
	"github.com/gomoni/gonix/internal/dbg"
	"github.com/spf13/pflag"

	_ "embed"
)

//go:embed head.awk
var headAwk []byte

//go:embed head_negative.awk
var headNegative []byte

type Head struct {
	debug          bool
	lines          int
	zeroTerminated bool
	files          []string
}

func New() Head {
	return Head{}
}

func (c Head) FromArgs(argv []string) (Head, error) {
	if len(argv) == 0 {
		c = c.Lines(10)
		return c, nil
	}

	flag := pflag.FlagSet{}

	var lines internal.Byte = internal.Byte(c.lines)
	flag.VarP(&lines, "lines", "n", "print at least n lines, -n means everything except last n lines")

	zeroTerminated := flag.BoolP("zero-terminated", "z", false, "line delimiter is NUL")

	err := flag.Parse(argv)
	if err != nil {
		return Head{}, pipe.NewErrorf(1, "head: parsing failed: %w", err)
	}
	if len(flag.Args()) > 0 {
		c.files = flag.Args()
	}

	// TODO: deal with more than int64 lines
	c.lines = int(math.Round(float64(lines)))
	c.zeroTerminated = *zeroTerminated

	return c, nil
}

// Files are input files, where - denotes stdin
func (c Head) Files(f ...string) Head {
	c.files = append(c.files, f...)
	return c
}

func (c Head) Lines(lines int) Head {
	c.lines = lines
	return c
}

func (c Head) ZeroTerminated(zeroTerminated bool) Head {
	c.zeroTerminated = zeroTerminated
	return c
}

func (c Head) SetDebug(debug bool) Head {
	c.debug = debug
	return c
}

func (c Head) Run(ctx context.Context, stdio unix.StandardIO) error {
	debug := dbg.Logger(c.debug, "cat", stdio.Stderr())
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
	debug.Printf("head: zero-terminated=%t", c.zeroTerminated)

	config := awk.NewConfig()
	if c.zeroTerminated {
		config.Vars = append(config.Vars, []string{"RS", "\x00"}...)
	}

	prog, err := parser.ParseProgram([]byte(src), nil)
	if err != nil {
		return err
	}
	awk := awk.New(prog, config)
	config.Vars = append(config.Vars, []string{"lines", strconv.Itoa(lines)}...)

	var head func(context.Context, unix.StandardIO, int, string) error
	if len(c.files) <= 1 {
		head = func(ctx context.Context, stdio unix.StandardIO, _ int, _ string) error {
			err := awk.Run(ctx, stdio)
			if err != nil {
				return pipe.NewError(1, fmt.Errorf("head: fail to run: %w", err))
			}
			return nil
		}
	} else {
		head = func(ctx context.Context, stdio unix.StandardIO, _ int, name string) error {
			fmt.Fprintf(stdio.Stdout(), "==> %s <==\n", name)
			err := awk.Run(ctx, stdio)
			if err != nil {
				return pipe.NewError(1, fmt.Errorf("head: fail to run: %w", err))
			}
			fmt.Fprintln(stdio.Stdout())
			return nil
		}
	}

	runFiles := internal.NewRunFiles(
		c.files,
		stdio,
		head,
	)
	return runFiles.Do(ctx)
}
