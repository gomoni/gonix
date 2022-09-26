// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gomoni/gonix/cat"
	"github.com/gomoni/gonix/cksum"
	"github.com/gomoni/gonix/head"
	"github.com/gomoni/gonix/pipe"
	"github.com/gomoni/gonix/wc"
	"github.com/spf13/pflag"
)

var tools map[string]func([]string) (pipe.Filter, error)

func main() {
	tools = map[string]func([]string) (pipe.Filter, error){
		"cat":   mkFilterFunc[cat.Cat](cat.New),
		"cksum": mkFilterFunc[cksum.CKSum](cksum.New),
		"head":  mkFilterFunc[head.Head](head.New),
		"wc":    mkFilterFunc[wc.Wc](wc.New),
	}

	args := os.Args
	if len(args) == 0 {
		panic("what to do here")
	}
	// strip gonix
	if filepath.Base(args[0]) == "gonix" {
		args = args[1:]
	}

	if len(args) == 0 {
		fmt.Printf("TODO: usage\n")
		os.Exit(1)
	}

	fromArgs, found := tools[args[0]]
	if !found {
		panic("exec not yet implemented")
	}
	filter, err := fromArgs(args[1:])
	if err != nil {
		var perr pipe.Error
		if errors.As(err, &perr) {
			if errors.Is(perr.Err, pflag.ErrHelp) {
				os.Exit(1)
				return
			}
		}
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
		cksum.CKSum |
		head.Head |
		wc.Wc

	pipe.Filter
}

type arger[T filter] interface {
	FromArgs([]string) (*T, error)
}

func mkFilterFunc[F filter, AF arger[F]](newArger func() AF) func([]string) (pipe.Filter, error) {
	return func(args []string) (pipe.Filter, error) {
		f, err := newArger().FromArgs(args)
		if err != nil {
			return nil, err
		}
		return *f, err
	}
}
