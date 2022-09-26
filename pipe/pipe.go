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
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/gomoni/gonix/internal/dbg"
	"github.com/hashicorp/go-multierror"
)

// EmptyStdio is a good default which does not read neither prints anything from/to stdin/stdout/stderr
var EmptyStdio = Stdio{
	zeroReader{},
	io.Discard,
	io.Discard,
}

// Stdio describes standard io streams for commands
type Stdio struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Filter is an interface of a unix command - it consumes stdin and write to
// stdout. Context is used for a cancellation of a filter's run.
type Filter interface {
	Run(context.Context, Stdio) error
}

type Pipe struct {
	noPipeFail bool
	debug      bool
}

func New() *Pipe {
	return &Pipe{
		noPipeFail: false,
	}
}

// Pipefail false continues the colon even in a case of an error. If true the only errors which
// stops colon are context.Canceled and context.DeadlineExceeded.
func (p *Pipe) Pipefail(b bool) *Pipe {
	p.noPipeFail = !b
	return p
}

func (p *Pipe) SetDebug(b bool) *Pipe {
	p.debug = b
	return p
}

// Run executes the colon. It connects first Filter to stdin and last to stdout and connects
// stdin/stdout via io.Pipe. Standard error is passed through. Every filter runs inside own
// goroutine. If any returns an error, all are canceled too and an error is returned.
//
// On default Pipefail(true) is setup, Run returns {firstNonZeroExitCode, all errors}.
// On Pipefail(false), Run returns {lastExitCode, all errors}. So {Code: 0, Err: something}
// is possible in this case.
func (p Pipe) Run(ctx context.Context, stdio Stdio, cmds ...Filter) error {
	debug := dbg.Logger(p.debug, "gonix.pipe", stdio.Stderr)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if len(cmds) == 0 {
		return fmt.Errorf("Can't run 0 commands")
	}
	if len(cmds) == 1 {
		return cmds[0].Run(ctx, stdio)
	}

	var wg sync.WaitGroup
	var lastCode uint8 = 0     // exit code of a last started Filter
	var firstNonZero uint8 = 0 // first non zero exit code
	var nonzeroOnce sync.Once

	errChan := make(chan error, len(cmds))

	in := io.NopCloser(stdio.Stdin)
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
		isLast := idx == len(cmds)-1
		go func(cancel context.CancelFunc, noPipeFail bool, cmd Filter, errChan chan<- error, stdin io.Reader, stdout io.WriteCloser) {
			defer wg.Done()
			defer closePipe(stdin)
			defer closePipe(stdout)

			// XXX TODO FIXME: his it possible for go to deal with a failed predecessor?
			// use atomics?
			/*
				if !p.noPipeFail && firstNonZero != 0 {
					// TODO: return an error here?
					return
				}
			*/

			err := cmd.Run(ctx, Stdio{stdin, stdout, stdio.Stderr})
			errChan <- err
			if isLast && err != nil {
				lastCode = AsError(err).Code
			}
			if err != nil {
				nonzeroOnce.Do(func() {
					firstNonZero = AsError(err).Code
				})
				if noPipeFail {
					if errors.Is(err, context.Canceled) ||
						errors.Is(err, context.DeadlineExceeded) {
						debug.Printf("noPipeFail=true, calling cancel on non nil err=%+v", err)
						cancel()
					}
				} else {
					debug.Printf("noPipeFail=false, calling cancel on non nil err=%+v", err)
					cancel()
				}
			}

		}(cancel, p.noPipeFail, cmd, errChan, in, out)
		in = nextIn
	}

	wg.Wait()

	var errs error
	for i := 0; i != len(cmds); i++ {
		select {
		case e := <-errChan:
			if e != nil {
				errs = multierror.Append(errs, e)
			}
		default:
			break
		}
	}
	close(errChan)

	if firstNonZero != 0 {
		debug.Printf("pipe.Run: firstNonZero != 0")
		if p.noPipeFail {
			// no pipe fail: does not cancel colons, returns exit code of last Filter
			err := NewError(lastCode, errs)
			debug.Printf("pipe.Run: noPipeFail: lastCode=%d, err=%T %+v", lastCode, err, err)
			return err
		} else {
			// set -o pipefail: return first non zero error
			err := NewError(firstNonZero, errs)
			debug.Printf("pipe.Run: PipeFail: firstNonZero=%d, err=%T %+v", firstNonZero, err, err)
			return err
		}
	}
	debug.Printf("pipe.Run: firstNonZero == 0")
	return nil
}

// Run executes the colon through default Pipe. Colon is canceled on error.
func Run(ctx context.Context, stdio Stdio, cmds ...Filter) error {
	return Pipe{}.Run(ctx, stdio, cmds...)
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

// zeroReader reads nothing from stdin and ends with io.EOF
type zeroReader struct{}

func (zeroReader) Read([]byte) (int, error) {
	return 0, io.EOF
}

// closePipe if an argument is *io.PipeWriter or *io.PipeReader returned by io.Pipe()
func closePipe(x any) {
	switch y := x.(type) {
	case *io.PipeReader:
		_ = y.Close()
	case *io.PipeWriter:
		_ = y.Close()
	}
}
