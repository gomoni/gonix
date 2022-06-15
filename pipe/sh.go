// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package pipe

import (
	"context"
	"fmt"
	"os/exec"
)

// SplitFn is a function used to split shell command line to individual elements
// github.com/desertbit/go-shlex is an example of a good parsing function
type SplitFn func(string) ([]string, error)

// Builtins is a typedef for a map of a command name to a builtin function
// which construct a command from list of arguments
type Builtins = map[string]func([]string) (Filter, error)

type FromArgsFn func([]string) (Filter, error)

// NotFoundFn is called on an unknown command
type NotFoundFn func(string) (FromArgsFn, error)

// Sh contains a configuration for command line parsing and running. It maps
// shell command line to individual Filters and allows one to execute a colon.
type Sh struct {
	builtins   Builtins
	splitfn    SplitFn
	notFoundFn NotFoundFn
	pipe       *Pipe
}

// NewSh creates an instance of a Sh with specified set of builtins and a split
// function
func NewSh(builtins Builtins, splitfn SplitFn) *Sh {
	return &Sh{
		builtins:   builtins,
		splitfn:    splitfn,
		notFoundFn: NotFoundFunc,
		pipe:       New(),
	}
}

// NotFoundFunc with true replaces every not found command by Exec filter,
// making it working similarly to a real shell. False means to throw an error
// if command not found.
func (s *Sh) NotFoundFunc(f NotFoundFn) *Sh {
	s.notFoundFn = f
	return s
}

// Pipefail is equivalent of bash set -o pipefail. Sh by default fails on error. Use
// false to get shell-like behavior.
func (s *Sh) Pipefail(b bool) *Sh {
	s.pipe.Pipefail(b)
	return s
}

// Parse returns a slice of filters based on given command line.
func (s Sh) Parse(cmdline string) ([]Filter, error) {
	args, err := s.splitfn(cmdline)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("nothing to do")
	}

	filters := make([]Filter, 0, 16)
	start := 0
	for {
		if start >= len(args) {
			break
		}
		arg0 := args[start]
		fromArgs, ok := s.builtins[arg0]
		if !ok {
			fromArgs, err = s.notFoundFn(arg0)
			if err != nil {
				return nil, err
			}
		}

		var argn []string
		if start+1 == len(args) {
			argn = []string{}
			start++
		} else {
			stop := start + nextPipe(args[start:])
			argn = args[start+1 : stop]
			start = stop + 1
		}
		filter, err := fromArgs(argn)
		if err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}
	return filters, nil
}

// Run does Parse and Run together
func (s Sh) Run(ctx context.Context, stdio Stdio, cmdline string) error {
	filters, err := s.Parse(cmdline)
	if err != nil {
		return err
	}
	return s.pipe.Run(ctx, stdio, filters...)
}

func nextPipe(args []string) int {
	for idx, a := range args {
		if a == "|" {
			return idx
		}
	}
	return len(args)
}

func NotFoundFunc(arg0 string) (FromArgsFn, error) {
	err := fmt.Errorf("can't run %q: %w", arg0, ErrBuiltinNotFound)
	return func([]string) (Filter, error) {
		return nil, Error{Code: NotFound, Err: err}
	}, err
}

// TODO: FIXME - this mixes pipe.Environ and exec.Command together. Shall it be
// an another struct?
// NotFoundFunc with initialized environment
func (e Environ) NotFoundFunc(arg0 string) (FromArgsFn, error) {
	fromArgs := func(args []string) (Filter, error) {
		cmd := exec.Command(arg0, args...)
		cmd.Env = e.Environ()
		return NewExec(cmd), nil
	}
	return fromArgs, nil
}
