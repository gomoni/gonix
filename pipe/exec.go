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
	cmd := exec.CommandContext(ctx, e.cmd.Path, e.cmd.Args...)
	cmd.Stdin = stdio.Stdin
	cmd.Stdout = stdio.Stdout
	cmd.Stderr = stdio.Stderr
	return cmd.Run()
}
