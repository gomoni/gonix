package tr

import (
	"strings"
	"testing"

	"github.com/gomoni/gonix/internal/test"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestCat(t *testing.T) {
	test.Parallel(t)
	testCases := []test.Case[Tr, *Tr]{
		{
			Name:     "tr -d aeiou",
			Filter:   New().Array1("aeiou").Delete(true),
			Input:    "three\nsmall\npigs\n",
			Expected: "thr\nsmll\npgs\n",
		},
		{
			Name:     "tr -d -c aeiou",
			Filter:   New().Array1("aeiou").Delete(true).Complement(true),
			Input:    "three\nsmall\npigs\n",
			Expected: "eeai",
		},
	}
	test.RunAll(t, testCases)
}

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
