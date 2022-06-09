// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package wc_test

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/gomoni/gonix/internal/test"
	"github.com/gomoni/gonix/pipe"
	. "github.com/gomoni/gonix/wc"

	"github.com/stretchr/testify/require"
)

// TODO: convert to table test
func TestWcFromArgs(t *testing.T) {

	wc1 := New().Bytes(true).Lines(true).Words(true)
	wc2, err := FromArgs([]string{})
	require.NoError(t, err)
	require.Equal(t, wc1, wc2)

	wc1 = New().Lines(true)
	wc2, err = FromArgs([]string{"--lines"})
	require.NoError(t, err)
	require.Equal(t, wc1, wc2)
}

func TestWc(t *testing.T) {
	threeSmallPigs := test.Testdata(t, "three-small-pigs")
	test.Parallel(t)
	dflt, err := FromArgs(nil)
	require.NoError(t, err)
	testCases := []testCase{
		{
			"default",
			dflt,
			"The three\nsmall\npigs\n",
			" 3 4 21\n",
		},
		{
			"wc -l",
			New().Lines(true),
			"three\nsmall\npigs\n",
			"3\n",
		},
		{
			"wc -cmlLw",
			New().Bytes(true).Chars(true).Lines(true).MaxLineLength(true).Words(true),
			"The three žluťoučká\nsmall\npigs\n",
			" 3 5 31 35 19\n",
		},
		{
			"wc - three-small-pigs",
			New().Lines(true).Files("-", threeSmallPigs),
			"1\n2\n3\n4\n",
			fmt.Sprintf(" 4 -\n 3 %s\n 7 total\n", threeSmallPigs),
		},
	}

	test.RunAll(t, testCases)
}

type testCase struct {
	name     string
	cmd      *Wc
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
