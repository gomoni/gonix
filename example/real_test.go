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

	"github.com/gomoni/gio/unix"
	"github.com/gomoni/gonix/cat"
	"github.com/gomoni/gonix/wc"
	"github.com/stretchr/testify/require"

	"go.uber.org/goleak"
)

func TestGoleak(t *testing.T) {
	defer goleak.VerifyNone(t)
	var b bytes.Buffer
	stdio := unix.NewStdio(
		io.NopCloser(bytes.NewBufferString("three\nsmall\npigs\n")),
		&b,
		os.Stderr,
	)
	ctx := context.Background()
	err := unix.NewLine().Run(ctx, stdio, cat.New(), wc.New().Lines(true))
	if err != nil {
		log.Fatal(err)
	}
	require.Equal(t, "3\n", b.String())
}
