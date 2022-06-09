package cat

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNonPrinting(t *testing.T) {
	var out bytes.Buffer

	inp := []byte{0, 8, 9, 10, 31, 32}
	nonPrinting(inp, &out)
	require.Equal(t, "^@^H\t\n^_ ", out.String())
	inp = []byte{32, 42, 126, 127}
	nonPrinting(inp, &out)
	require.Equal(t, " *~^?", out.String())
	inp = []byte{128, 142, 159}
	nonPrinting(inp, &out)
	require.Equal(t, "M-BM-^@M-BM-^NM-BM-^_", out.String())
	inp = []byte{160, 180, 191}
	nonPrinting(inp, &out)
	require.Equal(t, "M-BM- M-BM-4M-BM-?", out.String())
	inp = []byte{192, 202, 223}
	nonPrinting(inp, &out)
	require.Equal(t, "M-CM-^@M-CM-^JM-CM-^_", out.String())
	inp = []byte{224, 242, 255}
	nonPrinting(inp, &out)
	require.Equal(t, "M-CM- M-CM-2M-CM-?", out.String())
}
