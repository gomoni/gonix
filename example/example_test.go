// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package example_test

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/gomoni/gonix/cat"
	"github.com/gomoni/gonix/pipe"
	"github.com/gomoni/gonix/wc"

	shlex "github.com/desertbit/go-shlex"
)

// This example shows the pipe.Run
func ExampleRun() {
	stdio := pipe.Stdio{
		Stdin:  io.NopCloser(bytes.NewBufferString("three\nsmall\npigs\n")),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	ctx := context.Background()
	// printf "three\nsmall\npigs\n" | cat | wc -l
	err := pipe.Run(ctx, stdio, cat.New(), wc.New().Lines(true))
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// 3
}

// This example shows the pipe.Run with command arguments passed as string slice
func ExampleRun_from_args() {
	stdio := pipe.Stdio{
		Stdin:  io.NopCloser(bytes.NewBufferString("three\nsmall\npigs\n")),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	ctx := context.Background()
	cat, err := cat.New().FromArgs(nil)
	if err != nil {
		log.Fatal(err)
	}
	wc, err := wc.New().FromArgs([]string{"-l"})
	if err != nil {
		log.Fatal(err)
	}
	err = pipe.Run(ctx, stdio, cat, wc)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// 3
}

func ExampleRun_exec() {
	stdio := pipe.Stdio{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	ctx := context.Background()
	cmd := exec.Command("go", "version")
	goVersion := pipe.NewExec(cmd)
	wc, err := wc.New().FromArgs([]string{"-l"})
	if err != nil {
		log.Fatal(err)
	}
	// go version | wc -l
	err = pipe.Run(ctx, stdio, goVersion, wc)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// 1
}

// Parsing of the command line
func ExampleSh_Run() {
	builtins := map[string]func([]string) (pipe.Filter, error){
		"cat": func(a []string) (pipe.Filter, error) { return cat.New().FromArgs(a) },
		"wc":  func(a []string) (pipe.Filter, error) { return wc.New().FromArgs(a) },
	}
	splitfn := func(s string) ([]string, error) { return shlex.Split(s, true) }
	stdio := pipe.Stdio{
		Stdin:  io.NopCloser(bytes.NewBufferString("three\nsmall\npigs\n")),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	ctx := context.Background()

	sh := pipe.NewSh(builtins, splitfn)
	err := sh.Run(ctx, stdio, `cat | wc -l`)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// 3
}

// Parsing of the command line with exec enabled
func ExampleSh_Run_exec() {
	builtins := map[string]func([]string) (pipe.Filter, error){
		"wc": func(a []string) (pipe.Filter, error) { return wc.New().FromArgs(a) },
	}
	splitfn := func(s string) ([]string, error) { return shlex.Split(s, true) }
	stdio := pipe.Stdio{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	ctx := context.Background()

	sh := pipe.NewSh(builtins, splitfn).AllowExec(true)
	err := sh.Run(ctx, stdio, `go version | wc -l`)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// 1
}
