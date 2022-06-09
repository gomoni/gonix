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

type fromArgsFn func([]string) (Filter, error)
type missingFn func(string) (fromArgsFn, error)

// Sh contains a configuration for command line parsing and running. It maps
// shell command line to individual Filters and allows one to execute a colon.
type Sh struct {
	builtins  Builtins
	splitfn   SplitFn
	missingfn missingFn
}

// NewSh creates an instance of a Sh with specified set of builtins and a split
// function
func NewSh(builtins Builtins, splitfn SplitFn) *Sh {
	return &Sh{
		builtins:  builtins,
		splitfn:   splitfn,
		missingfn: notFound,
	}
}

// AllowExec with true replaces every not found command by Exec filter,
// making it working similarly to a real shell. False means to throw an error
// if command not found.
func (s *Sh) AllowExec(b bool) *Sh {
	s.missingfn = execFn
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
			fromArgs, err = s.missingfn(arg0)
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
	return Run(ctx, stdio, filters...)
}

func nextPipe(args []string) int {
	for idx, a := range args {
		if a == "|" {
			return idx
		}
	}
	return len(args)
}

func notFound(arg0 string) (fromArgsFn, error) {
	err := fmt.Errorf("builtin %q not found", arg0)
	return func([]string) (Filter, error) {
		return nil, err
	}, err
}

func execFn(arg0 string) (fromArgsFn, error) {
	fromArgs := func(args []string) (Filter, error) {
		cmd := exec.Command(arg0, args...)
		return NewExec(cmd), nil
	}
	return fromArgs, nil
}
