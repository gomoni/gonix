// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package wc_test

import (
	"fmt"
	"testing"

	"github.com/gomoni/gonix/internal/test"
	. "github.com/gomoni/gonix/wc"

	"github.com/stretchr/testify/require"
)

func TestWc(t *testing.T) {
	test.Parallel(t)
	threeSmallPigs := test.Testdata(t, "three-small-pigs")
	testCases := []test.Case[Wc, *Wc]{
		{
			Name:     "default",
			Filter:   fromArgs(t, []string{}),
			Input:    "The three\nsmall\npigs\n",
			Expected: " 3 4 21\n",
		},
		{
			Name:     "wc -l",
			Filter:   New().Lines(true),
			FromArgs: fromArgs(t, []string{"-l"}),
			Input:    "three\nsmall\npigs\n",
			Expected: "3\n",
		},
		{
			Name:     "wc --lines",
			Filter:   New().Lines(true),
			FromArgs: fromArgs(t, []string{"--lines"}),
			Input:    "three\nsmall\npigs\n",
			Expected: "3\n",
		},
		{
			Name:     "wc -cmlLw",
			Filter:   New().Bytes(true).Chars(true).Lines(true).MaxLineLength(true).Words(true),
			FromArgs: fromArgs(t, []string{"-cmlLw"}),
			Input:    "The three žluťoučká\nsmall\npigs\n",
			Expected: " 3 5 31 35 19\n",
		},
		{
			Name:     "wc -l - three-small-pigs",
			Filter:   New().Lines(true).Files("-", threeSmallPigs),
			FromArgs: fromArgs(t, []string{"-l", "-", threeSmallPigs}),
			Input:    "1\n2\n3\n4\n",
			Expected: fmt.Sprintf(" 4 -\n 3 %s\n 7 total\n", threeSmallPigs),
		},
	}

	test.RunAll(t, testCases)
}

func fromArgs(t *testing.T, argv []string) *Wc {
	t.Helper()
	n := New()
	f, err := n.FromArgs(argv)
	require.NoError(t, err)
	return f
}
