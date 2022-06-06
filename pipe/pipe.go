// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

/*
Package pipe provides a Go native equivalent of a unix pipe. The function run
accepts several Filter(s), which is a Go struct with a Run method accepting the
context and Stdio struct. This struct defines three standard io stream stdin,
stdout and stderr. When Run encounters more Filters, it automatically create a
colon connecting the stdout of previous command to stdin of a next one using
io.Pipe. Every Filter is then started in own goroutine so all commands works in
really streaming fashion.

Unlike real Unix goroutine supports the cooperative multitasking and can't be
killed externally. For this reason Filter implementation must check given
context and ends up accordingly.
*/

package pipe

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/go-multierror"
)

// Stdio describes standard io streams for commands
type Stdio struct {
	Stdin  io.ReadCloser
	Stdout io.Writer
	Stderr io.Writer
}

// Filter is an interface of a unix command - it consumes stdin and write to
// stdout. Context is used for a cancellation of a filter's run.
type Filter interface {
	Run(context.Context, Stdio) error
}

// Run executes the colon. It connects first Filter to stdin and last to stdout and connects
// stdin/stdout via io.Pipe. Standard error is passed through. Every filter runs inside own
// goroutine. If any returns an error, all are canceled too and an error is returned.
func Run(ctx context.Context, stdio Stdio, cmds ...Filter) error {
	var err error
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if len(cmds) == 0 {
		return fmt.Errorf("Can't run 0 commands")
	}
	if len(cmds) == 1 {
		return cmds[0].Run(ctx, stdio)
	}

	// else more cmds
	errs := make(chan error, 1)
	go func(errs <-chan error) {
		for {
			select {
			case e := <-errs:
				cancel()
				err = multierror.Append(err, e)
			case <-ctx.Done():
				return
			}
		}
	}(errs)

	var wg sync.WaitGroup

	in := stdio.Stdin
	for idx, cmd := range cmds {
		var nextIn io.ReadCloser
		var out io.WriteCloser
		if idx == len(cmds)-1 {
			// last one
			out = newNopCloser(stdio.Stdout)
		} else {
			pipeR, pipeW := io.Pipe()
			out = pipeW
			nextIn = pipeR
		}
		wg.Add(1)
		go func(cmd Filter, errs chan<- error, stdin io.ReadCloser, stdout io.WriteCloser) {
			defer wg.Done()
			defer stdin.Close()
			defer stdout.Close()
			err := cmd.Run(ctx, Stdio{stdin, stdout, stdio.Stderr})
			if err != nil {
				errs <- err
			}
		}(cmd, errs, in, out)
		in = nextIn
	}

	wg.Wait()
	return err
}

// newNopCloser returns a ReadCloser with a no-op Close method wrapping
// the provided Reader r.
func newNopCloser(w io.Writer) io.WriteCloser {
	return nopCloser{w}
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }
