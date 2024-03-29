package cksum_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gomoni/gio/unix"
	. "github.com/gomoni/gonix/cksum"
	"github.com/gomoni/gonix/internal/test"
	"github.com/stretchr/testify/require"
)

func TestCKSum(t *testing.T) {
	test.Parallel(t)
	testCases := []test.Case[CKSum]{
		{
			Name:     "default",
			Filter:   fromArgs(t, nil),
			Input:    "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			Expected: "1340348198 27 \n",
		},
		{
			Name:     "default untagged",
			Filter:   New().Untagged(false),
			FromArgs: fromArgs(t, nil),
			Input:    "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			Expected: "1340348198 27 \n",
		},
		{
			Name:     "md5",
			Filter:   New().Algorithm(MD5),
			FromArgs: fromArgs(t, []string{"--algorithm", "md5"}),
			Input:    "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			Expected: "MD5 (-) = f4699b80440c0403b31fce987f9cd8af\n",
		},
		{
			Name:     "md5 untagged",
			Filter:   New().Algorithm(MD5).Untagged(true),
			FromArgs: fromArgs(t, []string{"--algorithm", "md5", "--untagged"}),
			Input:    "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			Expected: "f4699b80440c0403b31fce987f9cd8af  -\n",
		},
	}

	test.RunAll(t, testCases)
}

func initTemp(t *testing.T, name string) string {
	t.Helper()
	temp, err := os.MkdirTemp("", "gonix-cksum-check*")
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

	spongef(t, name+".notag.md5", "%s  %s\n", "5f707e2a346cc0dac73e1323198a503c", name)
	spongef(t, name+".tag.md5", "MD5 (%s) = %s\n", name, "5f707e2a346cc0dac73e1323198a503c")
	spongef(t, name+".missing.file.tag.md5", "MD5 (%s) = %s\nMD5 (missing-file) = %s\n", name, "5f707e2a346cc0dac73e1323198a503c", "5f707e2a346cc0dac73e1323198a503c")
	spongef(t, name+".notag.broken.md5", "%s  %s\n", "1f707e2a346cc0dac73e1323198a503c", name)
	spongef(t, name+".tag.broken.md5", "MD5 (%s) = %s\n", name, "1f707e2a346cc0dac73e1323198a503c")

	return temp
}

func spongef(t *testing.T, name string, format string, a ...any) {
	t.Helper()
	f, err := os.Create(name)
	require.NoError(t, err)
	defer f.Close()
	_, err = fmt.Fprintf(f, format, a...)
	require.NoError(t, err)
	err = f.Sync()
	require.NoError(t, err)
}

type emptyStdio struct{}

func (e emptyStdio) Stdin() io.Reader {
	return noopReader{}
}
func (e emptyStdio) Stdout() io.Writer {
	return io.Discard
}
func (e emptyStdio) Stderr() io.Writer {
	return io.Discard
}

type noopReader struct{}

func (noopReader) Read([]byte) (int, error) {
	return 0, nil
}

func TestCheckCRC(t *testing.T) {
	test.Parallel(t)
	cksum := New().Check(true).Algorithm(CRC)
	err := cksum.Run(context.Background(), emptyStdio{})
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
		name           string
		cksum          CKSum
		expectedStdout string
	}{
		{
			name:           "md5 untagged",
			cksum:          New().Check(true).Algorithm(MD5).Files(tsp + ".notag.md5"),
			expectedStdout: "three-small-pigs: OK\n",
		},
		{
			name:           "md5 untagged autodetect",
			cksum:          New().SetDebug(testing.Verbose()).Check(true).Files(tsp + ".notag.md5"),
			expectedStdout: "three-small-pigs: OK\n",
		},
		{
			name:           "md5 tagged",
			cksum:          New().SetDebug(testing.Verbose()).Check(true).Untagged(false).Algorithm(MD5).Files(tsp + ".tag.md5"),
			expectedStdout: "three-small-pigs: OK\n",
		},
		{
			name:           "md5 ignore missing",
			cksum:          New().SetDebug(testing.Verbose()).Check(true).IgnoreMissing(true).Algorithm(MD5).Files(tsp + ".missing.file.tag.md5"),
			expectedStdout: "three-small-pigs: OK\n",
		},
		{
			name:           "md5 quiet",
			cksum:          New().SetDebug(testing.Verbose()).Check(true).Quiet(true).Algorithm(MD5).Files(tsp + ".tag.md5"),
			expectedStdout: "",
		},
		{
			name:           "md5 status",
			cksum:          New().SetDebug(testing.Verbose()).Check(true).Status(true).Algorithm(MD5).Files(tsp + ".tag.md5"),
			expectedStdout: "",
		},
		// TODO: sha512 and blake2b autodetect
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			test.Parallel(t)

			var stdout strings.Builder
			var stderr strings.Builder
			stdio := unix.NewStdio(
				nil,
				&stdout,
				&stderr,
			)

			err := tt.cksum.Run(context.Background(), stdio)
			t.Logf("stderr=%q", stderr.String())
			t.Logf("stdout=%q", stdout.String())
			require.NoError(t, err)

			require.Equal(t, tt.expectedStdout, stdout.String())
		})
	}

	// test errors
	errCases := []struct {
		name           string
		cksum          CKSum
		expectedError  string
		expectedStdout string
	}{
		{
			name:          "error --algorithm mismatch",
			cksum:         New().Check(true).Algorithm(SHA224).Files(tsp + ".tag.md5"),
			expectedError: "BadLineFormatError",
		},
		{
			name:          "error not found file",
			cksum:         New().Check(true).Algorithm(SHA224).Files(tsp + ".notfound.md5"),
			expectedError: "notfound.md5: no such file or directory",
		},
		{
			name:           "error mismatch tagged --algorithm NONE",
			cksum:          New().Check(true).Algorithm(NONE).Files(tsp + ".tag.broken.md5"),
			expectedStdout: "three-small-pigs: FAILED\n",
		},
		{
			name:           "error mismatch tagged --algorithm MD5",
			cksum:          New().Check(true).Algorithm(MD5).Files(tsp + ".tag.broken.md5"),
			expectedStdout: "three-small-pigs: FAILED\n",
		},
		{
			name:           "error mismatch untagged --algorithm NONE",
			cksum:          New().Check(true).Algorithm(NONE).Files(tsp + ".notag.broken.md5"),
			expectedStdout: "three-small-pigs: FAILED\n",
		},
		{
			name:           "error mismatch untagged --algorithm MD5",
			cksum:          New().Check(true).Algorithm(MD5).Files(tsp + ".notag.broken.md5"),
			expectedStdout: "three-small-pigs: FAILED\n",
		},
	}

	for _, tt := range errCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			test.Parallel(t)
			tt.cksum.SetDebug(testing.Verbose())

			var stdout strings.Builder
			var stderr strings.Builder
			stdio := unix.NewStdio(
				nil,
				&stdout,
				&stderr,
			)

			err := tt.cksum.Run(context.Background(), stdio)
			require.Error(t, err)

			if tt.expectedError != "" {
				require.True(t, strings.Contains(err.Error(), tt.expectedError))
			} else if tt.expectedStdout != "" {
				require.Equal(t, tt.expectedStdout, stdout.String())
			} else {
				t.Fatalf("test case %q does not check error neither stdout", tt.name)
			}
		})
	}

}

func fromArgs(t *testing.T, argv []string) CKSum {
	t.Helper()
	n := New()
	f, err := n.FromArgs(argv)
	require.NoError(t, err)
	return f
}
