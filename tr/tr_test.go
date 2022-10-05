package tr

import (
	"testing"

	"github.com/gomoni/gonix/internal/test"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestFoo(t *testing.T) {
	var s = trMap{
		'a': -1,
		'b': -1,
	}
	tr := delTr{pred: s.in}
	rn, ok := tr.Tr('a')
	require.True(t, ok)
	require.EqualValues(t, -1, rn)
	rn, ok = tr.Tr('x')
	require.False(t, ok)
	require.Equal(t, 'x', rn)

	tr = delTr{pred: s.notIn}
	rn, ok = tr.Tr('a')
	require.False(t, ok)
	require.Equal(t, 'a', rn)
	rn, ok = tr.Tr('x')
	require.True(t, ok)
	require.EqualValues(t, -1, rn)

	tr = delTr{pred: xdigit}
	_, ok = tr.Tr('x')
	require.False(t, ok)
	_, ok = tr.Tr('b')
	require.True(t, ok)
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
	}
	test.RunAll(t, testCases)
}

/*
func TestTrString(t *testing.T) {
	t.Parallel()
	i2Hash := mapTr(map[rune]rune{'i': '#'})

	testCases := []struct {
		name     string
		chain    chain
		input    string
		expected string
	}{
		{
			name:     `[:lower:]i [:upper:]#`,
			chain:    newChain(i2Hash, trFunc(lowerToUpper)),
			input:    "i žluťoučký",
			expected: "# ŽLUŤOUČKÝ",
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var out strings.Builder
			err := trString(tt.input, tt.chain, &out)
			require.NoError(t, err)
			require.Equal(t, tt.expected, out.String())
		})
	}
}
*/
