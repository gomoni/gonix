// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// Contains portions of cksum.c from suckless sbase under MIT license
// https://git.suckless.org/sbase/file/LICENSE.html

/*
check is surprisingly complex problem

what is implemented
✅ check don't work with crc
✅ check works with recent hashes like md5 or sha
✅ untagged format
✅ untagged format requires explicit --algorithm switch
✅ tagged format - happy path
✅ --check --algorithm returns no properly formatted lines error for a different hash
🚀 autodetect hash for untagged format - this include sha512 and blake2b
🚀 parallel check or checksums from one file limited by -j/--threds, defaults to GOMAXPROC
 parallel generation of checksums
🚀 parallel check or checksums from one file limited by -j/--threds, defaults to GOMAXPROC


what is not (yet)
❌ GNU options:
    -l/--length
    -z/--zero
    --strict    - cksum will return 1 by default
    -w/--warn
    --debug
❌ everything around warns and counting various errors - this looks to be completely random at least

*/

package cksum

import (
	"bufio"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/gomoni/gio/pipe"
	"github.com/gomoni/gio/unix"
	"github.com/gomoni/gonix/internal"
	"github.com/gomoni/gonix/internal/dbg"
	"github.com/spf13/pflag"
	"golang.org/x/crypto/blake2b"
)

type Algorithm int

const (
	NONE Algorithm = 0
	//sysv algorithm = 1
	//bsd  algorithm = 2
	CRC     Algorithm = 3
	MD5     Algorithm = 4
	SHA1    Algorithm = 5
	SHA224  Algorithm = 6
	SHA256  Algorithm = 7
	SHA384  Algorithm = 8
	SHA512  Algorithm = 9
	BLAKE2B Algorithm = 10
)

// https://pkg.go.dev/github.com/spf13/pflag#Value
func (a Algorithm) String() string {
	switch a {
	case CRC:
		return `crc`
	case MD5:
		return `md5`
	case SHA1:
		return `sha1`
	case SHA224:
		return `sha224`
	case SHA256:
		return `sha256`
	case SHA384:
		return `sha1384`
	case SHA512:
		return `sha512`
	case BLAKE2B:
		return `blake2b`
	default:
		return `!unknown`
	}
}

func (a Algorithm) Type() string {
	return "algorithm"
}

func (a *Algorithm) Set(value string) error {
	switch value {
	case `crc`:
		*a = CRC
	case `md5`:
		*a = MD5
	case `sha1`:
		*a = SHA1
	case `sha224`:
		*a = SHA224
	case `sha256`:
		*a = SHA256
	case `sha384`:
		*a = SHA384
	case `sha512`:
		*a = SHA512
	case `blake2b`:
		*a = BLAKE2B
	default:
		return fmt.Errorf("invalid argument %q for --algorithm", value)
	}
	return nil
}

type CKSum struct {
	threads       uint
	debug         bool
	algorithm     Algorithm
	check         bool
	untagged      bool
	ignoreMissing bool
	quiet         bool
	status        bool
	files         []string
}

func New() CKSum {
	return CKSum{}
}

// Files are input files, where - denotes stdin
func (c CKSum) Files(f ...string) CKSum {
	c.files = append(c.files, f...)
	return c
}

func (c CKSum) Algorithm(algorithm Algorithm) CKSum {
	c.algorithm = algorithm
	return c
}

func (c CKSum) Check(check bool) CKSum {
	c.check = true
	return c
}

func (c CKSum) IgnoreMissing(ignoreMissing bool) CKSum {
	c.ignoreMissing = ignoreMissing
	return c
}

func (c CKSum) Parallel(limit uint) CKSum {
	c.threads = limit
	return c
}

func (c CKSum) Quiet(quiet bool) CKSum {
	c.quiet = quiet
	return c
}

func (c CKSum) Untagged(untagged bool) CKSum {
	c.untagged = untagged
	return c
}

func (c CKSum) Status(status bool) CKSum {
	c.status = status
	return c
}
func (c CKSum) SetDebug(debug bool) CKSum {
	c.debug = debug
	return c
}

func (c CKSum) FromArgs(argv []string) (CKSum, error) {
	flag := pflag.FlagSet{}
	var algorithm Algorithm = NONE
	flag.VarP(&algorithm, "algorithm", "a", "checksum algorithm to use, crc is default")
	check := flag.BoolP("check", "c", false, "check checksums from file(s)")
	_ = flag.Bool("tag", true, "create BSD style checksum (default)")
	untagged := flag.Bool("untagged", false, "create checksum without digest type")
	ignoreMissing := flag.Bool("ignore-missing", false, "ignore missing files")
	quiet := flag.Bool("quiet", false, "do not print OK for every verified file")
	status := flag.Bool("status", false, "report status code only")

	// GNU is not consistent with parallel naming (make uses -j/--jobs, xargs -P and so
	// used -j/--threads as ripgrep does
	var threads uint
	flag.UintVarP(&threads, "threads", "j", 0, "generate or check using N goroutines, 0 equals GOMAXPROCS")
	err := flag.Parse(argv)
	if err != nil {
		return CKSum{}, pipe.NewErrorf(1, "cksum: parsing failed: %w", err)
	}

	if len(flag.Args()) > 0 {
		c.files = flag.Args()
	}

	c.algorithm = algorithm
	if c.algorithm == CRC || *untagged {
		c.untagged = true
	}
	c.check = *check
	c.ignoreMissing = *ignoreMissing
	c.quiet = *quiet
	c.status = *status
	c.threads = threads
	return c, nil
}

func (c CKSum) Run(ctx context.Context, stdio unix.StandardIO) error {
	debug := dbg.Logger(c.debug, "cksum", stdio.Stderr())
	if c.threads == 0 {
		c.threads = uint(runtime.GOMAXPROCS(0))
	}
	debug.Printf("running with --threads %d", c.threads)

	if c.check {
		debug.Printf("about to call c.checkSum")
		return c.checkSum(ctx, stdio, debug)
	}
	return c.makeSum(ctx, stdio, debug)
}

func (c CKSum) makeSum(ctx context.Context, stdio unix.StandardIO, _ *log.Logger) error {
	if c.algorithm == NONE {
		c.algorithm = CRC
	}

	var makeSum func(context.Context, unix.StandardIO, int, string) error

	switch c.algorithm {
	case CRC:
		makeSum = func(ctx context.Context, stdio unix.StandardIO, _ int, name string) error {
			cksum, size, err := docrc(ctx, func() simpleHash { return &crc{} }, stdio.Stdin())
			if err != nil {
				return err
			}
			fmt.Fprintf(stdio.Stdout(), "%s %d %s\n", cksum, size, name)
			return nil
		}
	default:
		hash, name, ok := c.algorithm.hashFunc()
		if !ok {
			return fmt.Errorf("invalid argument %q for --algorithm", c.algorithm)
		}
		makeSum = newDigestFunc(hash, name, c.untagged)
	}

	runFiles := internal.NewRunFiles(
		c.files,
		stdio,
		makeSum,
	)
	return runFiles.DoThreads(ctx, c.threads)
}

func (c CKSum) checkSum(ctx context.Context, stdio unix.StandardIO, debug *log.Logger) error {
	if c.check && c.algorithm == CRC {
		return fmt.Errorf("--check is not supported with algorithm=%s", c.algorithm)
	}

	ckSumOne := func(_ context.Context, line string) (checkResult, error) {
		res, err := c.checkLine(line, debug)
		if !c.ignoreMissing && err != nil {
			return res, err
		}
		return res, nil
	}

	ckSum := func(ctx context.Context, stdio unix.StandardIO, _ int, name string) error {
		r := bufio.NewScanner(stdio.Stdin())
		input := make([]string, 0, 16)
		for r.Scan() {
			if r.Err() != nil {
				return pipe.Error{Code: 1, Err: r.Err()}
			}
			input = append(input, r.Text())
		}

		results, err := internal.PMap(ctx, c.threads, input, ckSumOne)
		var ret int
		if err != nil {
			ret = 1
		}

		for _, result := range results {
			switch result.state {
			case stNONE:
				continue
			case stOK:
				if !c.quiet && !c.status {
					fmt.Fprintf(stdio.Stdout(), "%s: OK\n", result.name)
				}
			case stFAILED:
				if !c.status {
					fmt.Fprintf(stdio.Stdout(), "%s: FAILED\n", result.name)
					ret = 1
				}
			case stIO:
				if c.ignoreMissing {
					continue
				}
				if !c.status {
					fmt.Fprintf(stdio.Stdout(), "%s: FAILED open or read error\n", result.name)
					ret = 1
				}
			default:
				panic("unknown result state")
			}
		}

		if ret != 0 {
			return pipe.NewError(ret, err)
		}
		return nil
	}

	runFiles := internal.NewRunFiles(
		c.files,
		stdio,
		ckSum,
	)
	return runFiles.Do(ctx)
}

// copy from https://git.suckless.org/sbase/file/cksum.c.html#l11
var crctab = [256]uint32{0x00000000,
	0x04c11db7, 0x09823b6e, 0x0d4326d9, 0x130476dc, 0x17c56b6b,
	0x1a864db2, 0x1e475005, 0x2608edb8, 0x22c9f00f, 0x2f8ad6d6,
	0x2b4bcb61, 0x350c9b64, 0x31cd86d3, 0x3c8ea00a, 0x384fbdbd,
	0x4c11db70, 0x48d0c6c7, 0x4593e01e, 0x4152fda9, 0x5f15adac,
	0x5bd4b01b, 0x569796c2, 0x52568b75, 0x6a1936c8, 0x6ed82b7f,
	0x639b0da6, 0x675a1011, 0x791d4014, 0x7ddc5da3, 0x709f7b7a,
	0x745e66cd, 0x9823b6e0, 0x9ce2ab57, 0x91a18d8e, 0x95609039,
	0x8b27c03c, 0x8fe6dd8b, 0x82a5fb52, 0x8664e6e5, 0xbe2b5b58,
	0xbaea46ef, 0xb7a96036, 0xb3687d81, 0xad2f2d84, 0xa9ee3033,
	0xa4ad16ea, 0xa06c0b5d, 0xd4326d90, 0xd0f37027, 0xddb056fe,
	0xd9714b49, 0xc7361b4c, 0xc3f706fb, 0xceb42022, 0xca753d95,
	0xf23a8028, 0xf6fb9d9f, 0xfbb8bb46, 0xff79a6f1, 0xe13ef6f4,
	0xe5ffeb43, 0xe8bccd9a, 0xec7dd02d, 0x34867077, 0x30476dc0,
	0x3d044b19, 0x39c556ae, 0x278206ab, 0x23431b1c, 0x2e003dc5,
	0x2ac12072, 0x128e9dcf, 0x164f8078, 0x1b0ca6a1, 0x1fcdbb16,
	0x018aeb13, 0x054bf6a4, 0x0808d07d, 0x0cc9cdca, 0x7897ab07,
	0x7c56b6b0, 0x71159069, 0x75d48dde, 0x6b93dddb, 0x6f52c06c,
	0x6211e6b5, 0x66d0fb02, 0x5e9f46bf, 0x5a5e5b08, 0x571d7dd1,
	0x53dc6066, 0x4d9b3063, 0x495a2dd4, 0x44190b0d, 0x40d816ba,
	0xaca5c697, 0xa864db20, 0xa527fdf9, 0xa1e6e04e, 0xbfa1b04b,
	0xbb60adfc, 0xb6238b25, 0xb2e29692, 0x8aad2b2f, 0x8e6c3698,
	0x832f1041, 0x87ee0df6, 0x99a95df3, 0x9d684044, 0x902b669d,
	0x94ea7b2a, 0xe0b41de7, 0xe4750050, 0xe9362689, 0xedf73b3e,
	0xf3b06b3b, 0xf771768c, 0xfa325055, 0xfef34de2, 0xc6bcf05f,
	0xc27dede8, 0xcf3ecb31, 0xcbffd686, 0xd5b88683, 0xd1799b34,
	0xdc3abded, 0xd8fba05a, 0x690ce0ee, 0x6dcdfd59, 0x608edb80,
	0x644fc637, 0x7a089632, 0x7ec98b85, 0x738aad5c, 0x774bb0eb,
	0x4f040d56, 0x4bc510e1, 0x46863638, 0x42472b8f, 0x5c007b8a,
	0x58c1663d, 0x558240e4, 0x51435d53, 0x251d3b9e, 0x21dc2629,
	0x2c9f00f0, 0x285e1d47, 0x36194d42, 0x32d850f5, 0x3f9b762c,
	0x3b5a6b9b, 0x0315d626, 0x07d4cb91, 0x0a97ed48, 0x0e56f0ff,
	0x1011a0fa, 0x14d0bd4d, 0x19939b94, 0x1d528623, 0xf12f560e,
	0xf5ee4bb9, 0xf8ad6d60, 0xfc6c70d7, 0xe22b20d2, 0xe6ea3d65,
	0xeba91bbc, 0xef68060b, 0xd727bbb6, 0xd3e6a601, 0xdea580d8,
	0xda649d6f, 0xc423cd6a, 0xc0e2d0dd, 0xcda1f604, 0xc960ebb3,
	0xbd3e8d7e, 0xb9ff90c9, 0xb4bcb610, 0xb07daba7, 0xae3afba2,
	0xaafbe615, 0xa7b8c0cc, 0xa379dd7b, 0x9b3660c6, 0x9ff77d71,
	0x92b45ba8, 0x9675461f, 0x8832161a, 0x8cf30bad, 0x81b02d74,
	0x857130c3, 0x5d8a9099, 0x594b8d2e, 0x5408abf7, 0x50c9b640,
	0x4e8ee645, 0x4a4ffbf2, 0x470cdd2b, 0x43cdc09c, 0x7b827d21,
	0x7f436096, 0x7200464f, 0x76c15bf8, 0x68860bfd, 0x6c47164a,
	0x61043093, 0x65c52d24, 0x119b4be9, 0x155a565e, 0x18197087,
	0x1cd86d30, 0x029f3d35, 0x065e2082, 0x0b1d065b, 0x0fdc1bec,
	0x3793a651, 0x3352bbe6, 0x3e119d3f, 0x3ad08088, 0x2497d08d,
	0x2056cd3a, 0x2d15ebe3, 0x29d4f654, 0xc5a92679, 0xc1683bce,
	0xcc2b1d17, 0xc8ea00a0, 0xd6ad50a5, 0xd26c4d12, 0xdf2f6bcb,
	0xdbee767c, 0xe3a1cbc1, 0xe760d676, 0xea23f0af, 0xeee2ed18,
	0xf0a5bd1d, 0xf464a0aa, 0xf9278673, 0xfde69bc4, 0x89b8fd09,
	0x8d79e0be, 0x803ac667, 0x84fbdbd0, 0x9abc8bd5, 0x9e7d9662,
	0x933eb0bb, 0x97ffad0c, 0xafb010b1, 0xab710d06, 0xa6322bdf,
	0xa2f33668, 0xbcb4666d, 0xb8757bda, 0xb5365d03, 0xb1f740b4,
}

func docrc(_ context.Context, hashFunc func() simpleHash, stdin io.Reader) (string, int64, error) {
	var buf [4096]byte
	hash := hashFunc()
	n, err := io.CopyBuffer(hash, stdin, buf[:])
	if err != nil {
		return "", -1, err
	}
	return string(hash.Sum(nil)), n, nil
}

type crc struct {
	ck   uint32
	size int
}

func (c *crc) Write(buf []byte) (int, error) {
	n := len(buf)
	c.size += n
	ck := c.ck
	// https://git.suckless.org/sbase/file/cksum.c.html#l74
	for i := 0; i < n; i++ {
		ck = (ck << 8) ^ crctab[(ck>>24)^uint32(buf[i])]
	}
	c.ck = ck
	return n, nil
}

func (c crc) Sum(_ []byte) []byte {
	ck := c.ck
	for i := c.size; i != 0; i >>= 8 {
		ck = (ck << 8) ^ crctab[(ck>>24)^uint32((i&0xFF))]
	}
	return []byte(fmt.Sprintf("%d", ^ck))
}

func (a Algorithm) Size() int {
	switch a {
	case MD5:
		return 32
	case SHA1:
		return 40
	case SHA224:
		return 56
	case SHA256:
		return 64
	case SHA384:
		return 96
	case SHA512:
		return 128
	case BLAKE2B:
		return 128
	default:
		return -1
	}
}

func (a Algorithm) hashFunc() (func() simpleHash, string, bool) {
	switch a {
	case MD5:
		return func() simpleHash { return md5.New() }, "MD5", true
	case SHA1:
		return func() simpleHash { return sha1.New() }, "SHA1", true
	case SHA224:
		return func() simpleHash { return sha256.New224() }, "SHA224", true
	case SHA256:
		return func() simpleHash { return sha256.New() }, "SHA256", true
	case SHA384:
		return func() simpleHash { return sha512.New384() }, "SHA338", true
	case SHA512:
		return func() simpleHash { return sha512.New() }, "SHA512", true
	case BLAKE2B:
		hash, err := blake2b.New(64, nil)
		if err != nil {
			return nil, "", false
		}
		return func() simpleHash { return hash }, "BLAKE2b", true
	default:
		return nil, "", false
	}
}

func parseAlgorithm(s string) (Algorithm, error) {
	var a Algorithm
	switch strings.ToUpper(s) {
	case "MD5":
		a = MD5
	case "SHA1":
		a = SHA1
	case "SHA224":
		a = SHA224
	case "SHA256":
		a = SHA256
	case "SHA384":
		a = SHA384
	case "SHA512":
		a = SHA512
	case "BLAKE2B":
		a = BLAKE2B
	default:
		return NONE, fmt.Errorf("invalid argument %q for --algorithm", s)
	}
	return a, nil
}

// simpleHash implements just enough from hash.Hash interface
// so suckless crc does not need to implement all methods
type simpleHash interface {
	io.Writer
	Sum([]byte) []byte
}

// digest implements a digest for hash.Hash compatible stuff
// md5, sha256
func digest(hash simpleHash, stdin io.Reader) (string, error) {
	var buf [4096]byte
	_, err := io.CopyBuffer(hash, stdin, buf[:])
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func newDigestFunc(hashFunc func() simpleHash, hashName string, untagged bool) func(ctx context.Context, stdio unix.StandardIO, _ int, name string) error {
	return func(ctx context.Context, stdio unix.StandardIO, _ int, name string) error {
		hash := hashFunc()
		cksum, err := digest(hash, stdio.Stdin())
		if err != nil {
			return err
		}
		if name == "" {
			name = "-"
		}
		if untagged {
			fmt.Fprintf(stdio.Stdout(), "%s  %s\n", cksum, name)
		} else {
			fmt.Fprintf(stdio.Stdout(), "%s (%s) = %s\n", hashName, name, cksum)
		}
		return nil
	}
}

var (
	tagged = regexp.MustCompile("[A-Z]")
)

type BadLineFormatError string

func badLineFormatErrorf(temp string, args ...any) error {
	return BadLineFormatError(fmt.Sprintf(temp, args...))
}
func (e BadLineFormatError) Error() string {
	return fmt.Sprintf("BadLineFormatError(%q)", string(e))
}

type checkState int

const (
	stNONE   checkState = 0
	stOK     checkState = 1
	stFAILED checkState = 2
	stIO     checkState = 3
)

type checkResult struct {
	name  string
	state checkState
}

func stateOK(name string) checkResult {
	return checkResult{name: name, state: stOK}
}

func stateFAILED(name string) checkResult {
	return checkResult{name: name, state: stFAILED}
}

func stateIO(name string) checkResult {
	return checkResult{name: name, state: stIO}
}

// parse untagged and tagged formats
//   - untagged  hash name
//   - tagged HASH(name) = hash
//
// returns errors
// 1. BadLineFormatError for untagged format and algorithm NONE (unless autodetected)
// 2. BadLineFormatError for tagged format and a different hash
// 3. BadLineFormatError for wrong size of a hash
// 4. MismatchError for mismatched hash
func (c CKSum) checkLine(line string, debug *log.Logger) (checkResult, error) {
	var zero checkResult

	if len(line) == 0 {
		return zero, BadLineFormatError("empty")
	}

	if !tagged.MatchString(line[0:1]) {
		debug.Printf("checkLine: detected --untagged format")
		algorithms := make([]Algorithm, 0, 2)
		if c.algorithm == NONE {

			debug.Printf("checkLine: try to autodetect checksum")
			expected, _, ok := strings.Cut(line, " ")
			if !ok {
				debug.Printf("checLine: no space in --untagged format")
				goto cantDetect
			}
			switch len(expected) {
			case MD5.Size():
				c.algorithm = MD5
			case SHA1.Size():
				c.algorithm = SHA1
			case SHA224.Size():
				c.algorithm = SHA224
			case SHA256.Size():
				c.algorithm = SHA256
			case SHA384.Size():
				c.algorithm = SHA384
			case SHA512.Size():
				c.algorithm = SHA512 // or blake2b
				algorithms = []Algorithm{SHA512, BLAKE2B}
				debug.Printf("checLine: detected 512 bytes, trying SHA512 or BLAKE2b")
			default:
				goto cantDetect
			}
			goto detected

		cantDetect:
			return zero, BadLineFormatError("--algorithm must be specified with --untagged")
		}

	detected:
		checkSum := func(algorithm Algorithm) (checkResult, error) {
			// untagged format is hash<space><space>name: check there are two spaces there
			if line[algorithm.Size()] != ' ' || line[algorithm.Size()+1] != ' ' {
				return zero, BadLineFormatError("--untagged must have two spaces between sum and file name")
			}

			hash, _, ok := algorithm.hashFunc()
			if !ok {
				return zero, fmt.Errorf("unsupported --algorithm %q", c.algorithm)
			}

			name := line[algorithm.Size()+2:]
			err := checkSum(name, hash, line[:algorithm.Size()])
			if err == nil {
				return stateOK(name), nil
			}
			if errors.Is(err, errMismatch) {
				return stateFAILED(name), nil
			}
			return stateIO(name), nil
		}
		if len(algorithms) == 0 {
			return checkSum(c.algorithm)
		}
		res, err := checkSum(algorithms[0])
		if err == nil {
			return res, nil
		}
		var blferr BadLineFormatError
		if errors.As(err, &blferr) {
			res, err = checkSum(algorithms[1])
			if err == nil {
				return res, nil
			}
		}
		return res, err
	}

	debug.Printf("checkLine: detected --tag format")
	//TAG (file) = <hash>
	tag, rest, ok := strings.Cut(line, " ")
	if !ok {
		return zero, BadLineFormatError("no space after digest tag")
	}
	algorithm, err := parseAlgorithm(tag)
	if err != nil {
		return zero, badLineFormatErrorf("unsupported --algorithm tag %q", tag)
	}

	if c.algorithm != NONE && !strings.EqualFold(c.algorithm.String(), tag) {
		return zero, badLineFormatErrorf("line tag %q does not match --algorithm %q", tag, c.algorithm.String())
	}

	if len(rest) <= algorithm.Size() {
		return zero, badLineFormatErrorf("wrong size of hash: expected %d, got %d", algorithm.Size(), len(rest))
	}
	expected := rest[len(rest)-algorithm.Size():]

	// rest is now (name) =
	rest = rest[:len(rest)-algorithm.Size()]
	// so check and remove all remaining bytes
	lr := len(rest)
	if lr <= 5 || rest[0] != '(' || rest[lr-4:] != ") = " {
		return zero, badLineFormatErrorf("missing `() = ` around file name")
	}
	name := rest[1 : lr-4]

	hash, _, ok := algorithm.hashFunc()
	if !ok {
		return zero, fmt.Errorf("unsupported --algorithm %q", c.algorithm)
	}
	err = checkSum(name, hash, expected)
	if err == nil {
		return stateOK(name), nil
	}
	if errors.Is(err, errMismatch) {
		return stateFAILED(name), nil
	}
	if err != nil {
		return stateIO(name), err
	}
	panic("checkLine: tagged: should never go there")
}

var errMismatch = errors.New("checksum mismatch") // never returned upper

func checkSum(name string, hashFunc func() simpleHash, expected string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	checkSum, err := digest(hashFunc(), f)
	if err != nil {
		return err
	}

	if expected == checkSum {
		return nil
	}
	return errMismatch
}
