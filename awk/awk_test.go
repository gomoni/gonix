// Copyright 2023 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package awk_test

import (
	"testing"

	. "github.com/gomoni/gonix/awk"
	"github.com/gomoni/gonix/internal/test"

	"github.com/benhoyt/goawk/interp"
	"github.com/stretchr/testify/require"
)

func TestAWK(t *testing.T) {
	test.Parallel(t)

	testCases := []test.Case[AWK]{
		{
			Name:     "cat",
			Filter:   compile(t, newCfg(), `{print $1;}`),
			Input:    "01\tthree\n02\tsmall\n03\tpigs\n",
			Expected: "01\n02\n03\n",
		},
		{
			Name:     "cat FS ;",
			Filter:   compile(t, newCfg().FS(";"), `{print $2;}`),
			Input:    "01;three\n02;small\n03;pigs\n",
			Expected: "three\nsmall\npigs\n",
		},
	}
	test.RunAll(t, testCases)
}

func compile(t *testing.T, c *cfg, src string) AWK {
	t.Helper()
	awk, err := Compile([]byte(src), c.config)
	require.NoError(t, err)
	return awk
}

type cfg struct {
	config *interp.Config
}

func newCfg() *cfg {
	return &cfg{config: &interp.Config{}}
}
func (c *cfg) FS(value string) *cfg {
	c.config.Vars = append(c.config.Vars, []string{"FS", value}...)
	return c
}
