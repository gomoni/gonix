// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/gomoni/gonix/cat"
	"github.com/gomoni/gonix/head"
	"github.com/gomoni/gonix/pipe"
	"github.com/gomoni/gonix/wc"
)

var tools map[string]func([]string) (pipe.Filter, error)

func main() {
	tools = map[string]func([]string) (pipe.Filter, error){
		"cat":  mkFilterFunc[cat.Cat](cat.New),
		"head": mkFilterFunc[head.Head](head.New),
		"wc":   mkFilterFunc[wc.Wc](wc.New),
	}

	args := os.Args
	if len(args) == 0 {
		panic("what to do here")
	}
	// strip gonix
	if filepath.Base(args[0]) == "gonix" {
		args = args[1:]
	}

	fromArgs, found := tools[args[0]]
	if !found {
		panic("exec not yet implemented")
	}
	filter, err := fromArgs(args[1:])
	if err != nil {
		log.Fatal(err)
	}
	// implement ctrl+c support
	err = filter.Run(
		context.Background(),
		pipe.Stdio{
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		},
	)

	if err != nil {
		log.Fatal(err)
	}

}

type filter interface {
	cat.Cat |
		head.Head |
		wc.Wc

	Run(context.Context, pipe.Stdio) error
}

type arger[T filter] interface {
	FromArgs([]string) (*T, error)
}

func mkFilterFunc[F filter, AF arger[F]](newArger func() AF) func([]string) (pipe.Filter, error) {
	return func(args []string) (pipe.Filter, error) {
		f, err := newArger().FromArgs(args)
		return *f, err
	}
}
