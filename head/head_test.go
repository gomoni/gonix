// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package head_test

import (
	"io"
	"strings"
	"testing"

	. "github.com/gomoni/gonix/head"
	"github.com/gomoni/gonix/internal/test"
	"github.com/gomoni/gonix/pipe"
)

func TestHead(t *testing.T) {
	test.Parallel(t)
	testCases := []testCase{
		{
			"default",
			New(),
			"1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			"1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n",
		},
		{
			"--lines 2",
			New().Lines(2),
			"1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			"1\n2\n",
		},
		{
			"--lines -10",
			New().Lines(-10),
			"1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			"1\n2\n",
		},
	}

	test.RunAll(t, testCases)
}

type testCase struct {
	name     string
	cmd      *Head
	input    string
	expected string
}

func (tt testCase) Name() string {
	return tt.name
}

func (tt testCase) Filter() pipe.Filter {
	return tt.cmd
}

func (tt testCase) Input() io.ReadCloser {
	return io.NopCloser(strings.NewReader(tt.input))
}

func (tt testCase) Expected() string {
	return tt.expected
}
