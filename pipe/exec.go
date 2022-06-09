// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package pipe

import (
	"context"
	"os/exec"
)

type ExecFilter struct {
	cmd exec.Cmd
}

func NewExec(cmd *exec.Cmd) *ExecFilter {
	if cmd == nil {
		panic("cmd is nil")
	}
	return &ExecFilter{
		cmd: *cmd,
	}
}

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
