package tr

import (
	"testing"

	"github.com/gomoni/gonix/internal/test"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestTr(t *testing.T) {
	test.Parallel(t)
	testCases := []test.Case[Tr, *Tr]{
		{
			Name:     "tr -d aeiou",
			Filter:   New().Array1("aeiou").Delete(true),
			Input:    "three\nsmall\npigs\n",
			Expected: "thr\nsmll\npgs\n",
		},
		{
			Name:     "tr -d [:space:]",
			Filter:   New().Array1("[:space:]").Delete(true),
			Input:    "three\nsmall\npigs\n",
			Expected: "threesmallpigs",
		},
		{
			Name:     "tr -d \\n",
			Filter:   New().Array1("\\n").Delete(true),
			Input:    "three\nsmall\npigs\n",
			Expected: "threesmallpigs",
		},
		{
			Name:     "tr -d [=t=]\\n",
			Filter:   New().Array1("[=t=]\\n").Delete(true),
			Input:    "three\nsmall\npigs\n",
			Expected: "hreesmallpigs",
		},
		{
			Name:     "tr -d [:digit:][=t=]\\n",
			Filter:   New().Array1("[:digit:][=t=]\\n").Delete(true),
			Input:    "1:three\n2:small\n3:pigs\n",
			Expected: ":hree:small:pigs",
		},
		{
			Name:     "tr -d [:digit:][=t=]\\n\145",
			Filter:   New().Array1("[:digit:][=t=]\\n\\145").Delete(true),
			Input:    "1:three\n2:small\n3:pigs\n",
			Expected: ":hr:small:pigs",
		},
		{
			Name:     "tr -c -d aeiou",
			Filter:   New().Array1("aeiou").Delete(true).Complement(true),
			Input:    "three\nsmall\npigs\n",
			Expected: "eeai",
		},
		{
			Name:     "tr -c -d [:space:]",
			Filter:   New().Array1("[:space:]").Delete(true).Complement(true),
			Input:    "three\nsmall\npigs\n",
			Expected: "\n\n\n",
		},
		{
			Name:     "tr -c -d \\n",
			Filter:   New().Array1("\\n").Delete(true).Complement(true),
			Input:    "three\nsmall\npigs\n",
			Expected: "\n\n\n",
		},
		{
			Name:     "tr -c -d [=t=]\\n",
			Filter:   New().Array1("[=t=]\\n").Delete(true).Complement(true),
			Input:    "three\nsmall\npigs\n",
			Expected: "t\n\n\n",
		},
		{
			Name:     "tr -c -d [:digit:][=t=]\\n",
			Filter:   New().Array1("[:digit:][=t=]\\n").Delete(true).Complement(true),
			Input:    "1:three\n2:small\n3:pigs\n",
			Expected: "1t\n2\n3\n",
		},
		{
			Name:     "tr -c -d [:digit:][=t=]\\n\145",
			Filter:   New().Array1("[:digit:][=t=]\\n\\145").Delete(true).Complement(true),
			Input:    "1:three\n2:small\n3:pigs\n",
			Expected: "1tee\n2\n3\n",
		},
		{
			Name:     "tr e a",
			Filter:   New().Array1("e").Array2("a"),
			Input:    "three\nsmall\npigs\n",
			Expected: "thraa\nsmall\npigs\n",
		},
		{
			Name:     "tr el a",
			Filter:   New().Array1("el").Array2("a"),
			Input:    "three\nsmall\npigs\n",
			Expected: "thraa\nsmaaa\npigs\n",
		},
		{
			Name:     "tr el\\n a",
			Filter:   New().Array1("el\\n").Array2("a"),
			Input:    "three\nsmall\npigs\n",
			Expected: "thraaasmaaaapigsa",
		},
		{
			Name:     "tr el\\n aX",
			Filter:   New().Array1("el\\n").Array2("aX"),
			Input:    "three\nsmall\npigs\n",
			Expected: "thraaXsmaXXXpigsX",
		},
		{
			Name:     "tr e xy",
			Filter:   New().Array1("e").Array2("xy"),
			Input:    "three\nsmall\npigs\n",
			Expected: "thrxx\nsmall\npigs\n",
		},
		{
			Name:     "tr [=e=] xy",
			Filter:   New().Array1("e").Array2("xy"),
			Input:    "three\nsmall\npigs\n",
			Expected: "thrxx\nsmall\npigs\n",
		},
		{
			Name:     "tr [:digit:] X",
			Filter:   New().Array1("[:digit:]").Array2("X"),
			Input:    "1:three\n2:small\n3:pigs\n",
			Expected: "X:three\nX:small\nX:pigs\n",
		},
		{
			Name:     "tr [:digit:] XY",
			Filter:   New().Array1("[:digit:]").Array2("XY"),
			Input:    "1:three\n2:small\n3:pigs\n",
			Expected: "Y:three\nY:small\nY:pigs\n",
		},
		{
			Name:     "tr e[:digit:] XY",
			Filter:   New().Array1("e[:digit:]").Array2("XY"),
			Input:    "1:three\n2:small\n3:pigs\n",
			Expected: "Y:thrXX\nY:small\nY:pigs\n",
		},
		{
			Name:     "tr e[:digit:] X",
			Filter:   New().Array1("e[:digit:]").Array2("X"),
			Input:    "1:three\n2:small\n3:pigs\n",
			Expected: "X:thrXX\nX:small\nX:pigs\n",
		},
		{
			Name:     "tr [:digit:]e X",
			Filter:   New().Array1("[:digit:]e").Array2("X"),
			Input:    "1:three\n2:small\n3:pigs\n",
			Expected: "X:thrXX\nX:small\nX:pigs\n",
		},
		{
			Name:     "tr [:digit:]e XY",
			Filter:   New().Array1("[:digit:]e").Array2("XY"),
			Input:    "1:three\n2:small\n3:pigs\n",
			Expected: "Y:thrYY\nY:small\nY:pigs\n",
		},
	}
	test.RunAll(t, testCases)
}
