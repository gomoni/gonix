// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package head_test

import (
	"testing"

	. "github.com/gomoni/gonix/head"
	"github.com/gomoni/gonix/internal/test"
	"github.com/stretchr/testify/require"
)

func TestHead(t *testing.T) {
	test.Parallel(t)
	testCases := []test.Case[Head]{
		{
			Name:     "default",
			Filter:   fromArgs(t, []string{}),
			Input:    "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			Expected: "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n",
		},
		{
			Name:     "--lines 2",
			Filter:   New().Lines(2),
			FromArgs: fromArgs(t, []string{"-n", "2"}),
			Input:    "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			Expected: "1\n2\n",
		},
		{
			Name:     "--lines -10",
			Filter:   New().Lines(-10),
			FromArgs: fromArgs(t, []string{"-n", "-10"}),
			Input:    "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			Expected: "1\n2\n",
		},
		{
			Name:     "--lines 2 --zero-terminated",
			Filter:   New().Lines(2).ZeroTerminated(true),
			FromArgs: fromArgs(t, []string{"-n", "2", "--zero-terminated"}),
			Input:    "1\x002\x003\x004\x00",
			Expected: "1\n2\n",
		},
	}
	test.RunAll(t, testCases)
}

func fromArgs(t *testing.T, argv []string) Head {
	t.Helper()
	n := New()
	f, err := n.FromArgs(argv)
	require.NoError(t, err)
	return f
}
