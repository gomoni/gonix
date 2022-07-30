// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package pipe

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ExecFilter implements [Filter] interface for [exec.Cmd] so ordinary commands
// can be injected into gonix pipe
type ExecFilter struct {
	cmd exec.Cmd
}

// NewExec returns new ExecFilter from exec.Cmd
func NewExec(cmd *exec.Cmd) *ExecFilter {
	if cmd == nil {
		panic("cmd is nil")
	}
	return &ExecFilter{
		cmd: *cmd,
	}
}

// Environ sets the environment of a process, check documentation of exec.Cmd.Env
func (e *ExecFilter) Environ(env []string) *ExecFilter {
	e.cmd.Env = env
	return e
}

func (e *ExecFilter) Run(ctx context.Context, stdio Stdio) error {
	cmd := exec.CommandContext(ctx, e.cmd.Path, e.cmd.Args[1:]...)
	cmd.Stdin = stdio.Stdin
	cmd.Stdout = stdio.Stdout
	cmd.Stderr = stdio.Stderr
	return cmd.Run()
}

// Environ is a helper on top of []string implementing basic manipulation with
// os.Environ() compatible data type
type Environ []string

// DuplicateEnviron duplicates process environment
func DuplicateEnviron() *Environ {
	e := os.Environ()
	ret := Environ(e)
	return &ret
}

// NewEnviron creates new environment and inherits given variable names
func NewEnviron(names ...string) *Environ {
	e := make([]string, 0, len(names))
	for _, name := range names {
		if v, exists := os.LookupEnv(name); exists {
			e = append(e, fmt.Sprintf("%s=%s", name, v))
		}
	}
	ret := Environ(e)
	return &ret
}

// Set sets the environment variable, uses the format of setenv(3)
// always overwrite.
func (e *Environ) Set(name, v string) *Environ {
	*e = append(*e, fmt.Sprintf("%s=%s", name, v))
	return e
}

// Unset removes all occurrences of name from Environ. Do nothing if not present.
func (e *Environ) Unset(name string) *Environ {
	idx := 0
	a := []string(*e)
	for _, line := range a {
		varName, _, ok := strings.Cut(line, "=")
		if !ok || name == varName {
			// pass
		} else {
			a[idx] = line
			idx++
		}
	}
	a = a[:idx]
	ret := Environ(a)
	*e = ret
	return &ret
}

// Environ exports content as []string for compatibility with exec.Cmd API
func (e Environ) Environ() []string {
	return []string(e)
}

// ExecFunc calls exec.Command for each not builtin command and
// assigns an environment there
func (e Environ) ExecFunc(arg0 string) (FromArgsFunc, error) {
	fromArgs := func(args []string) (Filter, error) {
		cmd := exec.Command(arg0, args...)
		cmd.Env = e.Environ()
		return NewExec(cmd), nil
	}
	return fromArgs, nil
}
