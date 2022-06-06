// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package cat_test

import (
	"io"
	"strings"
	"testing"

	. "github.com/gomoni/gonix/cat"
	"github.com/gomoni/gonix/internal/test"
	"github.com/gomoni/gonix/pipe"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestCat(t *testing.T) {
	test.Parallel(t)

	testCases := []testCase{
		{
			"cat",
			Cat(),
			"three\nsmall\npigs\n",
			"three\nsmall\npigs\n",
		},
		// --show-all
		{
			"cat -b",
			Cat().ShowNumber(NonBlank),
			"three\n\n\nsmall\npigs\n",
			"     1\tthree\n\n\n     2\tsmall\n     3\tpigs\n",
		},
		// -e   equivalent to -vE
		{
			"cat -E",
			Cat().ShowEnds(true),
			"three\nsmall\npigs\n",
			"three$\nsmall$\npigs$\n",
		},
		{
			"cat -n",
			Cat().ShowNumber(All),
			"three\nsmall\npigs\n",
			"     1\tthree\n     2\tsmall\n     3\tpigs\n",
		},
		{
			"cat -s",
			Cat().SqueezeBlanks(true),
			"three\n\n\nsmall\npigs\n",
			"three\n\nsmall\npigs\n",
		},
		// -t     equivalent to -vT
		{
			"cat -T",
			Cat().ShowTabs(true),
			"\tthree\nsmall\t\npi\tgs\n",
			"^Ithree\nsmall^I\npi^Igs\n",
		},
		// -s show-non-printing
		// must be brave enough to implement this
		// https://github.com/coreutils/coreutils/blob/master/src/cat.c#L415
		{
			"cat -ET",
			Cat().ShowEnds(true).ShowTabs(true),
			"\tthree\nsmall\t\npi\tgs\n",
			"^Ithree$\nsmall^I$\npi^Igs$\n",
		},
	}

	test.RunAll(t, testCases)
}

type testCase struct {
	name     string
	cmd      *CatFilter
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
