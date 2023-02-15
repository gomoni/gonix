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
	"fmt"
	"io"

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

// AWK is a thin wrapper on top of github.com/benhoyt/goawk
type AWK struct {
	program *parser.Program
	config  *interp.Config
}

func New(prog *parser.Program, config *interp.Config) AWK {
	return AWK{
		program: prog,
		config:  config,
	}
}

func Compile(src []byte, config *interp.Config) (AWK, error) {
	if config == nil {
		return AWK{}, fmt.Errorf("nil config")
	}
	pconfig := parser.ParserConfig{
		DebugTypes:  false,
		DebugWriter: io.Discard,
		Funcs:       config.Funcs,
	}
	prog, err := parser.ParseProgram(src, &pconfig)
	if err != nil {
		return AWK{}, err
	}
	return AWK{
		program: prog,
		config:  config,
	}, nil
}

func (c AWK) Run(ctx context.Context, stdio unix.StandardIO) error {
	if c.config == nil {
		return fmt.Errorf("nil config")
	}
	if c.program == nil {
		return fmt.Errorf("nil prog")
	}
	config := *c.config
	config.Stdin = stdio.Stdin()
	config.Output = stdio.Stdout()
	config.Error = stdio.Stderr()
	_, err := interp.ExecProgram(c.program, &config)
	return err
}
