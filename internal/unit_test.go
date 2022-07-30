package internal

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseByte(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input    string
		expected Byte
	}{
		{
			"0",
			Byte(0),
		},
		{
			"00008",
			Byte(8),
		},
		{
			"42",
			Byte(42),
		},
		{
			"5b",
			5 * Block,
		},
		{
			"7kB",
			7 * KiloByte,
		},
		{
			"8K",
			8 * KibiByte,
		},
		{
			"9KiB",
			9 * KibiByte,
		},
		{
			"1MB500kB",
			1*MegaByte + 500*KiloByte,
		},
		{
			"1024KiB",
			1024 * KibiByte,
		},
		{
			"1024KiB",
			1 * MebiByte,
		},
		{
			"1.5G",
			1.5 * GibiByte,
		},
		{
			"-1.5G",
			-1.5 * GibiByte,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			s, err := ParseByte(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.expected, s)
		})
	}
}

func TestParseByteErr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			input:    "",
			expected: `invalid size "": empty`,
		},
		{
			input:    " ",
			expected: `invalid size " ": expected number or decimal separator`,
		},
		{
			input:    "x",
			expected: `invalid size "x": expected number or decimal separator`,
		},
		{
			input:    "3x",
			expected: `unknown unit "x" in size "3x"`,
		},
		{
			name:     "maxfloat640",
			input:    fmt.Sprintf("%f", math.MaxFloat64) + "0",
			expected: fmt.Sprintf(`invalid size %q: overflow`, fmt.Sprintf("%f", math.MaxFloat64)+"0"),
		},
		{
			name:     "maxfloat64b",
			input:    fmt.Sprintf("%f", math.MaxFloat64) + "b",
			expected: fmt.Sprintf(`invalid size %q: overflow`, fmt.Sprintf("%f", math.MaxFloat64)+"b"),
		},
		{
			input:    "\xf22000000",
			expected: `invalid size "\xf22000000": expected number or decimal separator`,
		},
		{
			input:    "A",
			expected: `invalid size "A": expected number or decimal separator`,
		},
		{
			input:    "\t\t",
			expected: `invalid size "\t\t": expected number or decimal separator`,
		},
		{
			input:    ".",
			expected: `invalid size ".": no digits`,
		},
		{
			input:    "-.s",
			expected: `invalid size "-.s": no digits`,
		},
	}

	for _, tt := range testCases {
		tt := tt
		name := func() string {
			if tt.name != "" {
				return tt.name
			}
			return tt.input
		}
		t.Run(name(), func(t *testing.T) {
			t.Parallel()
			_, err := ParseByte(tt.input)
			require.Error(t, err)
			require.EqualError(t, err, tt.expected)
		})
	}
}
