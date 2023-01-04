//go:generate go run ../../wasm/internal/gen/gen.go -i tr.yaml -p tr -o tr.go
package tr_test

import (
	"context"
	"testing"

	"github.com/gomoni/gonix/internal/test"
	. "github.com/gomoni/gonix/sbase/tr"
	"github.com/gomoni/gonix/wasm"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func TestTr(t *testing.T) {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	t.Cleanup(func() {
		err := r.Close(ctx)
		require.NoError(t, err)
	})
	wasi_snapshot_preview1.MustInstantiate(ctx, r)
	code, err := Compile(ctx, r)
	require.NoError(t, err)

	w := func(c wasm.Configer) *wasm.Filter {
		return wasm.New(r, code, c)
	}

	test.Parallel(t)
	testCases := []test.Case[wasm.Filter, *wasm.Filter]{
		{
			Name:     "tr -d aeiou",
			Filter:   w(New().Set1("aeiou").Delete(true)),
			Input:    "three\nsmall\npigs\n",
			Expected: "thr\nsmll\npgs\n",
		},
		{
			Name:     "tr -d [:space:]",
			Filter:   w(New().Set1("[:space:]").Delete(true)),
			Input:    "three\nsmall\npigs\n",
			Expected: "threesmallpigs",
		},
		{
			Name:     "tr -d \\n",
			Filter:   w(New().Set1("\\n").Delete(true)),
			Input:    "three\nsmall\npigs\n",
			Expected: "threesmallpigs",
		},
		{
			Name:     "tr -d [=t=]\\n",
			Filter:   w(New().Set1("[=t=]\\n").Delete(true)),
			Input:    "three\nsmall\npigs\n",
			Expected: "hreesmallpigs",
		},
		{
			Name:     "tr -c -d aeiou",
			Filter:   w(New().Set1("aeiou").Delete(true).Complement(true)),
			Input:    "three\nsmall\npigs\n",
			Expected: "eeai",
		},
		{
			Name:     "tr -c -d [:space:]",
			Filter:   w(New().Set1("[:space:]").Delete(true).Complement(true)),
			Input:    "three\nsmall\npigs\n",
			Expected: "\n\n\n",
		},
		{
			Name:     "tr -c -d \\n",
			Filter:   w(New().Set1("\\n").Delete(true).Complement(true)),
			Input:    "three\nsmall\npigs\n",
			Expected: "\n\n\n",
		},
		{
			Name:     "tr -c -d [=t=]\\n",
			Filter:   w(New().Set1("[=t=]\\n").Delete(true).Complement(true)),
			Input:    "three\nsmall\npigs\n",
			Expected: "t\n\n\n",
		},
		{
			Name:     "tr e a",
			Filter:   w(New().Set1("e").Set2("a")),
			Input:    "three\nsmall\npigs\n",
			Expected: "thraa\nsmall\npigs\n",
		},
		{
			Name:     "tr el a",
			Filter:   w(New().Set1("el").Set2("a")),
			Input:    "three\nsmall\npigs\n",
			Expected: "thraa\nsmaaa\npigs\n",
		},
		{
			Name:     "tr el\\n a",
			Filter:   w(New().Set1("el\\n").Set2("a")),
			Input:    "three\nsmall\npigs\n",
			Expected: "thraaasmaaaapigsa",
		},
		{
			Name:     "tr el\\n aX",
			Filter:   w(New().Set1("el\\n").Set2("aX")),
			Input:    "three\nsmall\npigs\n",
			Expected: "thraaXsmaXXXpigsX",
		},
		{
			Name:     "tr e xy",
			Filter:   w(New().Set1("e").Set2("xy")),
			Input:    "three\nsmall\npigs\n",
			Expected: "thrxx\nsmall\npigs\n",
		},
		{
			Name:     "tr [=e=] xy",
			Filter:   w(New().Set1("e").Set2("xy")),
			Input:    "three\nsmall\npigs\n",
			Expected: "thrxx\nsmall\npigs\n",
		},
		{
			Name:     "tr [:digit:] X",
			Filter:   w(New().Set1("[:digit:]").Set2("X")),
			Input:    "1:three\n2:small\n3:pigs\n",
			Expected: "X:three\nX:small\nX:pigs\n",
		},
		{
			Name:     "tr [:digit:] XY",
			Filter:   w(New().Set1("[:digit:]").Set2("XY")),
			Input:    "1:three\n2:small\n3:pigs\n",
			Expected: "Y:three\nY:small\nY:pigs\n",
		},
	}
	test.RunAll(t, testCases)
}
