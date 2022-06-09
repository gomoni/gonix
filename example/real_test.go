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
	"testing"

	"github.com/gomoni/gonix/cat"
	"github.com/gomoni/gonix/pipe"
	"github.com/gomoni/gonix/wc"
	"github.com/stretchr/testify/require"

	"go.uber.org/goleak"
)

func TestGoleak(t *testing.T) {
	defer goleak.VerifyNone(t)
	var b bytes.Buffer
	stdio := pipe.Stdio{
		Stdin:  io.NopCloser(bytes.NewBufferString("three\nsmall\npigs\n")),
		Stdout: &b,
		Stderr: os.Stderr,
	}
	ctx := context.Background()
	err := pipe.Run(ctx, stdio, cat.New(), wc.New().Lines(true))
	if err != nil {
		log.Fatal(err)
	}
	require.Equal(t, "3\n", b.String())
}
