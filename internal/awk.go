package internal

import (
	"context"

	"github.com/benhoyt/goawk/interp"
	"github.com/benhoyt/goawk/parser"
	"github.com/gomoni/gio/unix"
)

// Awk - maybe this will morph to bigger awk command, but for know lets
// keep it here in order to reuse Run functionality of a multiple awk programs
type Awk struct {
	prog   *parser.Program
	config *interp.Config
}

func NewAWK(prog *parser.Program) *Awk {
	return &Awk{
		prog:   prog,
		config: &interp.Config{},
	}
}

func (c *Awk) SetVariable(name, value string) *Awk {
	c.config.Vars = append(c.config.Vars, []string{name, value}...)
	return c
}

func (c Awk) Run(ctx context.Context, stdio unix.StandardIO) error {
	// not safe to use via different goroutines
	c.config.Stdin = stdio.Stdin()
	c.config.Output = stdio.Stdout()
	c.config.Error = stdio.Stderr()
	_, err := interp.ExecProgram(c.prog, c.config)
	return err
}
