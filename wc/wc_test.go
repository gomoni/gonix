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

// TODO: convert to table test
func TestWcFromArgs(t *testing.T) {

	wc1 := New().Bytes(true).Lines(true).Words(true)
	wc2, err := New().FromArgs([]string{})
	require.NoError(t, err)
	require.Equal(t, wc1, wc2)

	wc1 = New().Lines(true)
	wc2, err = New().FromArgs([]string{"--lines"})
	require.NoError(t, err)
	require.Equal(t, wc1, wc2)
}

func TestWc(t *testing.T) {
	threeSmallPigs := test.Testdata(t, "three-small-pigs")
	test.Parallel(t)
	dflt, err := New().FromArgs(nil)
	require.NoError(t, err)
	testCases := []test.Case[Wc, *Wc]{
		{
			Name:     "default",
			Filter:   dflt,
			Input:    "The three\nsmall\npigs\n",
			Expected: " 3 4 21\n",
		},
		{
			Name:     "wc -l",
			Filter:   New().Lines(true),
			Input:    "three\nsmall\npigs\n",
			Expected: "3\n",
		},
		{
			Name:     "wc -cmlLw",
			Filter:   New().Bytes(true).Chars(true).Lines(true).MaxLineLength(true).Words(true),
			Input:    "The three žluťoučká\nsmall\npigs\n",
			Expected: " 3 5 31 35 19\n",
		},
		{
			Name:     "wc - three-small-pigs",
			Filter:   New().Lines(true).Files("-", threeSmallPigs),
			Input:    "1\n2\n3\n4\n",
			Expected: fmt.Sprintf(" 4 -\n 3 %s\n 7 total\n", threeSmallPigs),
		},
	}

	test.RunAll(t, testCases)
}
