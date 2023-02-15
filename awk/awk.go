// Copyright 2023 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

/*
awk is a thin wrapper on top of github.com/benhoyt/goawk/interp and github.com/benhoyt/goawk/parser
providing a compatible [unix.Filter] interface for goawk.
*/

package awk

import (
	"context"

	"github.com/benhoyt/goawk/interp"
	"github.com/benhoyt/goawk/parser"
	"github.com/gomoni/gio/unix"
)

// AWK - maybe this will morph to bigger awk command, but for know lets
// keep it here in order to reuse Run functionality of a multiple awk programs
type AWK struct {
	prog   *parser.Program
	config *interp.Config
}

func New(prog *parser.Program, config *interp.Config) *AWK {
	return &AWK{
		prog:   prog,
		config: config,
	}
}

func (c *AWK) SetVariable(name, value string) *AWK {
	c.config.Vars = append(c.config.Vars, []string{name, value}...)
	return c
}

func (c AWK) Run(ctx context.Context, stdio unix.StandardIO) error {
	// not safe to use via different goroutines
	c.config.Stdin = stdio.Stdin()
	c.config.Output = stdio.Stdout()
	c.config.Error = stdio.Stderr()
	_, err := interp.ExecProgram(c.prog, c.config)
	return err
}
