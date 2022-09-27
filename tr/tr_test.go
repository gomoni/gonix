package tr

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrString(t *testing.T) {
	t.Parallel()
	i2Hash := map[rune]rune{'i': '#'}

	testCases := []struct {
		name     string
		chain    chain
		input    string
		expected string
	}{
		{
			name:     `[:lower:]i [:upper:]#`,
			chain:    newChain(i2Hash, lowerToUpper),
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
