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
	}
	test.RunAll(t, testCases)
}
