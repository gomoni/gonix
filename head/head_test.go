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
	"github.com/stretchr/testify/require"
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
		{
			"--lines 2 --zero-terminated",
			New().Lines(2).ZeroTerminated(true),
			"1\x002\x003\x004\x00",
			"1\n2\n",
		},
	}

	test.RunAll(t, testCases)
}

func TestFromArgs(t *testing.T) {
	test.Parallel(t)
	testCases := []struct {
		name     string
		args     []string
		expected *Head
	}{
		{
			"default",
			nil,
			New(),
		},
		{
			"lines",
			[]string{"--lines", "10KiB"},
			New().Lines(10240),
		},
		{
			"zero terminated",
			[]string{"--zero-terminated"},
			New().ZeroTerminated(true),
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			test.Parallel(t)
			head, err := New().FromArgs(tt.args)
			require.NoError(t, err)
			require.Equal(t, tt.expected, head)
		})
	}
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
