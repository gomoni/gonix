// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package wc

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/gomoni/gonix/internal"
	"github.com/gomoni/gonix/pipe"
	"github.com/spf13/pflag"
)

/*
   Print  newline,  word, and byte counts for each FILE, and a total line if more than one FILE is specified.  A word is a non-zero-length sequence of printable characters delimited
   by white space.

   With no FILE, or when FILE is -, read standard input.

   The options below may be used to select which counts are printed, always in the following order: newline, word, character, byte, maximum line length.

   --files0-from=F
          read input from the files specified by NUL-terminated names in file F; If F is - then read names from standard input

   --version
          output version information and exit

*/

type WcFilter struct {
	debug         bool
	bytes         bool
	chars         bool
	lines         bool
	maxLineLength bool
	words         bool
	files         []string
}

func Wc() *WcFilter {
	return &WcFilter{}
}

// FromArgs builds a WcFilter from standard argv except the command name (os.Argv[1:])
func FromArgs(argv []string) (*WcFilter, error) {
	cmd := &WcFilter{}

	if len(argv) == 0 {
		cmd = cmd.Bytes(true).Lines(true).Words(true)
		return cmd, nil
	}

	flag := pflag.FlagSet{}
	flag.BoolVarP(&cmd.bytes, "bytes", "c", false, "print number of bytes")
	flag.BoolVarP(&cmd.chars, "chars", "m", false, "print number of characters (runes)")
	flag.BoolVarP(&cmd.lines, "lines", "l", false, "print number of lines")
	flag.BoolVarP(&cmd.maxLineLength, "max-line-length", "L", false, "print maximum display width")
	flag.BoolVarP(&cmd.words, "words", "w", false, "print number of words")

	err := flag.Parse(argv)
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

func (w *WcFilter) Bytes(b bool) *WcFilter {
	w.bytes = b
	return w
}

func (w *WcFilter) Chars(b bool) *WcFilter {
	w.chars = b
	return w
}

func (w *WcFilter) Lines(lines bool) *WcFilter {
	w.lines = lines
	return w
}

func (w *WcFilter) MaxLineLength(b bool) *WcFilter {
	w.maxLineLength = b
	return w
}

func (w *WcFilter) Words(b bool) *WcFilter {
	w.words = b
	return w
}

// Files adds files into a list of files
func (w *WcFilter) Files(files ...string) *WcFilter {
	w.files = append(w.files, files...)
	return w
}

func (w *WcFilter) SetDebug(debug bool) *WcFilter {
	w.debug = debug
	return w
}

func (c WcFilter) Run(ctx context.Context, stdio pipe.Stdio) error {
	debug := internal.Logger(c.debug, "wc", stdio.Stderr)

	files := c.files
	if len(files) == 0 {
		files = []string{""}
	}
	stat := make([]stats, 0, len(c.files))
	total := stats{fileName: "total"}

	for _, f := range files {
		var in io.ReadCloser
		if f == "" || f == "-" {
			in = stdio.Stdin
		} else {
			f, err := os.Open(f)
			if err != nil {
				fmt.Fprintf(stdio.Stderr, "%s\n", err)
				continue
			}
			defer f.Close()
			in = f
		}
		st, err := c.runFile(ctx, in, debug)
		if err != nil {
			fmt.Fprintf(stdio.Stderr, "%s\n", err)
			continue
		}
		st.fileName = f
		total.add(st)
		stat = append(stat, st)
	}

	percents, argsFn := c.percentsArgsFn()
	stdinOnly := len(files) == 1 && files[0] == ""
	var template string
	if stdinOnly {
		template = fmt.Sprintf("%s\t\n", strings.Join(percents, "\t"))
	} else {
		template = fmt.Sprintf("%s\t %%s\n", strings.Join(percents, "\t"))
	}
	minWidth := total.maxLen()
	padding := 1
	if len(stat) == 1 && len(argsFn) == 1 {
		padding = 0
	}
	debug.Printf("template=%q", template)
	debug.Printf("minWidth=%+v, tabwith=8, padding=%+v", minWidth, padding)
	w := tabwriter.NewWriter(stdio.Stdout, minWidth-padding, 8, padding, ' ', tabwriter.AlignRight)

	if stdinOnly {
		args := make([]any, 0, len(argsFn))
		for _, fn := range argsFn {
			args = append(args, fn(stat[0]))
		}
		fmt.Fprintf(w, template, args...)
		return w.Flush()
	}

	stat = append(stat, total)
	for _, st := range stat {
		args := make([]any, 0, len(argsFn))
		for _, fn := range argsFn {
			args = append(args, fn(st))
		}
		args = append(args, st.fileName)
		fmt.Fprintf(w, template, args...)
	}

	err := w.Flush()
	if err != nil {
		return err
	}

	debug.Printf("exiting")
	return nil
}

func (c WcFilter) runFile(ctx context.Context, in io.Reader, debug *log.Logger) (stats, error) {
	var stat stats
	s := bufio.NewScanner(in)
	for s.Scan() {
		if s.Err() != nil {
			return stat, s.Err()
		}
		if ctx.Err() != nil {
			return stat, ctx.Err()
		}
		if c.bytes {
			// TODO: windows has two(?)
			stat.bytes += len(s.Bytes()) + 1
		}
		if c.chars || c.maxLineLength {
			count := utf8.RuneCount(s.Bytes())
			// TODO: windows has two(?)
			stat.chars += count + 1
			if count > stat.maxLineLength {
				// \n does not count to maxLineLength
				stat.maxLineLength = count
			}
		}
		if c.words {
			ws := bufio.NewScanner(bytes.NewReader(s.Bytes()))
			ws.Split(bufio.ScanWords)
			for ws.Scan() {
				stat.words += 1
			}
		}
		stat.lines++
	}
	return stat, nil
}

// percentsArgsFn ensures wc prints in following order: newline, word,
// character, byte, maximum line length.
func (c WcFilter) percentsArgsFn() ([]string, []func(stats) int) {
	percents := make([]string, 0, 5)
	argsFn := make([]func(stat stats) int, 0, 5)
	if c.lines {
		argsFn = append(argsFn, func(stat stats) int { return stat.lines })
		percents = append(percents, "%d")
	}
	if c.words {
		argsFn = append(argsFn, func(stat stats) int { return stat.words })
		percents = append(percents, "%d")
	}
	if c.chars {
		argsFn = append(argsFn, func(stat stats) int { return stat.chars })
		percents = append(percents, "%d")
	}
	if c.bytes {
		argsFn = append(argsFn, func(stat stats) int { return stat.bytes })
		percents = append(percents, "%d")
	}
	if c.maxLineLength {
		argsFn = append(argsFn, func(stat stats) int { return stat.maxLineLength })
		percents = append(percents, "%d")
	}
	return percents, argsFn
}

type stats struct {
	bytes         int
	chars         int
	lines         int
	maxLineLength int
	words         int
	fileName      string
}

func (s *stats) add(t stats) {
	s.bytes += t.bytes
	s.chars += t.chars
	s.lines += t.lines
	s.maxLineLength += t.maxLineLength
	s.words += t.words
}

func (s stats) maxLen() int {
	foo := [5]int{
		s.bytes,
		s.chars,
		s.lines,
		s.maxLineLength,
		s.words,
	}
	sort.Ints(foo[:])
	return len(strconv.Itoa(foo[4]))
}
