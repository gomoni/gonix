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
	testCases := []test.Case[Head, *Head]{
		{
			Name:     "default",
			Filter:   New(),
			Input:    "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			Expected: "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n",
		},
		{
			Name:     "--lines 2",
			Filter:   New().Lines(2),
			Input:    "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			Expected: "1\n2\n",
		},
		{
			Name:     "--lines -10",
			Filter:   New().Lines(-10),
			Input:    "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			Expected: "1\n2\n",
		},
		{
			Name:     "--lines 2 --zero-terminated",
			Filter:   New().Lines(2).ZeroTerminated(true),
			Input:    "1\x002\x003\x004\x00",
			Expected: "1\n2\n",
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
