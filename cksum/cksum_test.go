package cksum_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/gomoni/gonix/cksum"
	"github.com/gomoni/gonix/internal/test"
	"github.com/gomoni/gonix/pipe"
	"github.com/stretchr/testify/require"
)

func TestCKSum(t *testing.T) {
	test.Parallel(t)
	testCases := []testCase{
		{
			"default",
			New(),
			"1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			"1340348198 27 \n",
		},
		{
			"default untagged",
			New().Untagged(false),
			"1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			"1340348198 27 \n",
		},
		{
			"md5",
			New().Algorithm(MD5),
			"1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			"MD5 (-) = f4699b80440c0403b31fce987f9cd8af\n",
		},
		{
			"md5 untagged",
			New().Algorithm(MD5).Untagged(true),
			"1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			"f4699b80440c0403b31fce987f9cd8af  -\n",
		},
	}

	test.RunAll(t, testCases)
}

type testCase struct {
	name     string
	cmd      *CKSum
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

func initTemp(t *testing.T, name string) string {
	t.Helper()
	temp, err := os.MkdirTemp("", "cksum-check*")
	require.NoError(t, err)
	err = os.Chdir(temp)
	require.NoError(t, err)
	dest, err := os.Create(name)
	require.NoError(t, err)
	defer dest.Close()
	src, err := os.Open(test.Testdata(t, name))
	require.NoError(t, err)
	defer src.Close()
	_, err = io.Copy(dest, src)
	require.NoError(t, err)
	err = dest.Sync()
	require.NoError(t, err)

	md5, err := os.Create(name + ".notag.md5")
	require.NoError(t, err)
	defer md5.Close()
	fmt.Fprintf(md5, "%s  %s\n", "5f707e2a346cc0dac73e1323198a503c", name)
	err = md5.Sync()
	require.NoError(t, err)

	md5t, err := os.Create(name + ".tag.md5")
	require.NoError(t, err)
	defer md5.Close()
	fmt.Fprintf(md5t, "MD5 (%s) = %s\n", name, "5f707e2a346cc0dac73e1323198a503c")
	err = md5t.Sync()
	require.NoError(t, err)

	return temp
}

func TestCheckCRC(t *testing.T) {
	test.Parallel(t)
	cksum := New().Check(true).Algorithm(CRC)
	err := cksum.Run(context.Background(), pipe.Stdio{})
	require.Error(t, err)
	require.EqualError(t, err, "--check is not supported with algorithm=crc")
}

func TestCheck(t *testing.T) {
	test.Parallel(t)

	temp := initTemp(t, "three-small-pigs")
	tsp := filepath.Join(temp, "three-small-pigs")
	t.Logf("temp=%q", temp)
	t.Cleanup(func() {
		err := os.RemoveAll(temp)
		require.NoError(t, err)
	})

	testCases := []struct {
		name  string
		cksum *CKSum
	}{
		{
			name:  "md5 untagged",
			cksum: New().Check(true).Algorithm(MD5).Files(tsp + ".notag.md5"),
		},
		{
			name:  "md5 untagged autodetect",
			cksum: New().SetDebug(testing.Verbose()).Check(true).Files(tsp + ".notag.md5"),
		},
		// TODO: sha512 and blake2b autodetect
		{
			name:  "md5 tagged",
			cksum: New().SetDebug(testing.Verbose()).Check(true).Untagged(false).Algorithm(MD5).Files(tsp + ".tag.md5"),
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			test.Parallel(t)

			var stdout strings.Builder
			var stderr strings.Builder
			stdio := pipe.Stdio{
				Stdin:  nil,
				Stdout: &stdout,
				Stderr: &stderr,
			}

			err := tt.cksum.Run(context.Background(), stdio)
			t.Logf("err=%q", stderr.String())
			require.NoError(t, err)

			t.Logf("out=%q", stdout.String())

		})
	}

}
