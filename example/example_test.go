// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package example_test

import (
	"bytes"
	"context"
	"log"
	"os"

	"github.com/gomoni/gio/unix"
	"github.com/gomoni/gonix/cat"
	"github.com/gomoni/gonix/head"
	"github.com/gomoni/gonix/wc"
)

// This example shows the unix.NewLine().Run with cat and wc
func Example() {
	stdio := unix.NewStdio(
		bytes.NewBufferString("three\nsmall\npigs\n"),
		os.Stdout,
		os.Stderr,
	)
	ctx := context.Background()
	// printf "three\nsmall\npigs\n" | cat | wc -l
	err := unix.NewLine().Run(ctx, stdio, cat.New(), wc.New().Lines(true))
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// 3
}

// This example shows the unix.NewLine().Run with cat and wc with arguments passed as []string
func Example_from_args() {
	stdio := unix.NewStdio(
		bytes.NewBufferString("three\nsmall\npigs\n"),
		os.Stdout,
		os.Stderr,
	)
	ctx := context.Background()
	cat, err := cat.New().FromArgs(nil)
	if err != nil {
		log.Fatal(err)
	}
	wc, err := wc.New().FromArgs([]string{"-l"})
	if err != nil {
		log.Fatal(err)
	}
	err = unix.NewLine().Run(ctx, stdio, cat, wc)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// 3
}

/* FIXME: NewExec shall be ported back to gio
func ExampleRun_exec() {
	stdio := unix.NewStdio(
		os.Stdin,
		os.Stdout,
		os.Stderr,
	)
	ctx := context.Background()
	cmd := exec.Command("go", "version")
	goVersion := pipe.NewExec(cmd)
	wc, err := wc.New().FromArgs([]string{"-l"})
	if err != nil {
		log.Fatal(err)
	}
	// go version | wc -l
	err = unix.NewLine().Run(ctx, stdio, goVersion, wc)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// 1
}
*/

func ExampleHead_Run() {
	head := head.New().Lines(2)
	err := head.Run(context.TODO(), unix.NewStdio(
		bytes.NewBufferString("three\nsmall\npigs\n"),
		os.Stdout,
		os.Stderr,
	))
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// three
	// small
}
