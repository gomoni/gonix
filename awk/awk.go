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

func NewConfig() *interp.Config {
	return &interp.Config{
		NoArgVars:    true,
		NoExec:       true,
		NoFileWrites: true,
		NoFileReads:  true,
		ShellCommand: []string{"/bin/true"},
	}
}

// AWK - maybe this will morph to bigger awk command, but for know lets
// keep it here in order to reuse Run functionality of a multiple awk programs
type AWK struct {
	prog   *parser.Program
	config *interp.Config
}

func New(prog *parser.Program, config *interp.Config) AWK {
	return AWK{
		prog:   prog,
		config: config,
	}
}

func (c AWK) Run(ctx context.Context, stdio unix.StandardIO) error {
	// not safe to use via different goroutines
	config := *c.config
	config.Stdin = stdio.Stdin()
	config.Output = stdio.Stdout()
	config.Error = stdio.Stderr()
	_, err := interp.ExecProgram(c.prog, &config)
	return err
}
