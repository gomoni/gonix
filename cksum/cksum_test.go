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
	f, err := os.Create(name)
	require.NoError(t, err)
	defer f.Close()
	_, err = fmt.Fprintf(f, format, a...)
	require.NoError(t, err)
	err = f.Sync()
	require.NoError(t, err)
}

func TestCheckCRC(t *testing.T) {
	test.Parallel(t)
	cksum := New().Check(true).Algorithm(CRC)
	err := cksum.Run(context.Background(), pipe.EmptyStdio)
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
		cksum          *CKSum
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
			stdio := pipe.Stdio{
				Stdin:  nil,
				Stdout: &stdout,
				Stderr: &stderr,
			}

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
		cksum          *CKSum
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
			stdio := pipe.Stdio{
				Stdin:  nil,
				Stdout: &stdout,
				Stderr: &stderr,
			}

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

func TestFromArgs(t *testing.T) {
	test.Parallel(t)
	testCases := []struct {
		name     string
		args     []string
		expected *CKSum
	}{
		{
			"default",
			nil,
			New(),
		},
		{
			"--check",
			[]string{"--check"},
			New().Check(true),
		},
		{
			"--tag",
			[]string{"--tag"},
			New(),
		},
		{
			"--algorithm crc",
			[]string{"--algorithm", "crc"},
			New().Algorithm(CRC).Untagged(true),
		},
		{
			"--algorithm sha1",
			[]string{"--algorithm", "sha1"},
			New().Algorithm(SHA1),
		},
		{
			"--algorithm blake2b --untagged",
			[]string{"--algorithm", "blake2b", "--untagged"},
			New().Algorithm(BLAKE2B).Untagged(true),
		},
		{
			"--ignore-missing",
			[]string{"--ignore-missing"},
			New().IgnoreMissing(true),
		},
		{
			"--quiet",
			[]string{"--quiet"},
			New().Quiet(true),
		},
		{
			"--status",
			[]string{"--status"},
			New().Status(true),
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			test.Parallel(t)
			cksum, err := New().FromArgs(tt.args)
			require.NoError(t, err)
			require.Equal(t, tt.expected, cksum)
		})
	}
}
