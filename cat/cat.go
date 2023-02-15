// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package cat

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"

	"github.com/gomoni/gio/pipe"
	"github.com/gomoni/gio/unix"
	"github.com/gomoni/gonix/awk"
	"github.com/gomoni/gonix/internal"
	"github.com/gomoni/gonix/internal/dbg"

	"github.com/benhoyt/goawk/interp"
	"github.com/benhoyt/goawk/parser"
	"github.com/spf13/pflag"

	_ "embed"
)

type number int

const (
	None     number = 0
	NonBlank number = 1
	All      number = 2
)

var (
	ErrNothingToDo = pipe.NewErrorf(1, "cat: nothing to do")
)

//go:embed show_ends.awk
var showEndsAwk []byte

//go:embed show_number.awk
var showNumberAwk []byte

//go:embed show_number_nonblank.awk
var showNumberNonBlankAwk []byte

//go:embed squeeze_blanks.awk
var squeezeBlanksAwk []byte

//go:embed show_tabs.awk
var showTabsAwk []byte

type Cat struct {
	debug           bool
	files           []string
	showNumber      number
	showEnds        bool
	squeezeBlanks   bool
	showTabs        bool
	showNonPrinting bool
}

func New() Cat {
	return Cat{}
}

// FromArgs build a Cat from standard argv except the command name (os.Argv[1:])
func (c Cat) FromArgs(argv []string) (Cat, error) {
	var zero Cat
	flag := pflag.FlagSet{}

	nb := flag.BoolP("number-nonblank", "b", false, "number non blank lines only")
	flag.BoolVarP(&c.showEnds, "show-ends", "E", false, "print $ at the end of each line")
	na := flag.BoolP("number", "n", false, "number all lines")
	flag.BoolVarP(&c.squeezeBlanks, "squeeze-blanks", "s", false, "ignore repeated blank lines")
	flag.Bool("u", false, "ignored, for compatibility with POSIX")
	flag.BoolVarP(&c.showTabs, "show-tabs", "T", false, "print TAB as ^I")
	flag.BoolVarP(&c.showNonPrinting, "show-nonprinting", "v", false, "use ^ and M- notation for non printing characters")

	// compound options
	var all, e, t bool
	flag.BoolVarP(&all, "show-all", "A", false, "equivalent of -vET")
	// TODO FIXME - single dash options only - this accepts -e and --e
	flag.BoolVarP(&e, "e", "e", false, "equivalent of -vE")
	flag.BoolVarP(&t, "t", "t", false, "equivalent of -vT")

	err := flag.Parse(argv)
	if err != nil {
		return zero, pipe.NewErrorf(1, "cat: parsing failed: %w", err)
	}

	if all {
		c.ShowNonPrinting(true).ShowEnds(true).ShowTabs(true)
	}
	if e {
		c.ShowNonPrinting(true).ShowEnds(true)
	}
	if t {
		c.ShowNonPrinting(true).ShowTabs(true)
	}

	if len(flag.Args()) > 0 {
		c.files = flag.Args()
	}

	// post process
	if *nb {
		c.ShowNumber(NonBlank)
	} else if *na {
		c.ShowNumber(All)
	}

	return c, nil
}

// Files are input files, where - denotes stdin
func (c Cat) Files(f ...string) Cat {
	c.files = append(c.files, f...)
	return c
}

// ShowNumber adds none all or non empty output lines
func (c Cat) ShowNumber(n number) Cat {
	c.showNumber = n
	return c
}

// ShowEnds add $ to the end of each line
func (c Cat) ShowEnds(b bool) Cat {
	c.showEnds = b
	return c
}

// SqueezeBlanks - supress repeated empty lines
func (c Cat) SqueezeBlanks(b bool) Cat {
	c.squeezeBlanks = b
	return c
}

// ShowTabs display TAB as ^I
func (c Cat) ShowTabs(b bool) Cat {
	c.showTabs = b
	return c
}

// ShowNonPrinting use ^ and M- notation, except for LFD and TAB
func (c Cat) ShowNonPrinting(b bool) Cat {
	c.showNonPrinting = b
	return c
}

// SetDebug additional debugging messages on stderr
func (c Cat) SetDebug(debug bool) Cat {
	c.debug = debug
	return c
}

func (c Cat) modifyStdout() bool {
	return c.showNumber != None || c.showEnds || c.squeezeBlanks || c.showTabs || c.showNonPrinting
}

func (c Cat) Run(ctx context.Context, stdio unix.StandardIO) error {
	debug := dbg.Logger(c.debug, "cat", stdio.Stderr())
	var filters []unix.Filter
	if !c.modifyStdout() {
		filters = []unix.Filter{cat{debug: c.debug}}
	} else {
		progs, err := c.awk(debug)
		if err != nil {
			return err
		}
		filters = make([]unix.Filter, len(progs))
		for idx, prog := range progs {
			filters[idx] = awk.New(prog, &interp.Config{})
		}
	}
	if c.showNonPrinting {
		filters = append(filters, catNonPrinting{})
	}
	if len(filters) == 0 {
		return ErrNothingToDo
	}

	cat := func(ctx context.Context, stdio unix.StandardIO, _ int, _ string) error {
		err := unix.NewLine().Run(ctx, stdio, filters...)
		if err != nil {
			return pipe.NewError(1, fmt.Errorf("cat: fail to run: %w", err))
		}
		return nil
	}

	runFiles := internal.NewRunFiles(c.files, stdio, cat)
	return runFiles.Do(ctx)
}

func (c Cat) awk(debug *log.Logger) ([]*parser.Program, error) {
	debug.Printf("c=%+v", c)
	var sources [][]byte
	if c.showEnds {
		sources = append(sources, showEndsAwk)
	}
	if c.showNumber == All {
		sources = append(sources, showNumberAwk)
	} else if c.showNumber == NonBlank {
		sources = append(sources, showNumberNonBlankAwk)
	}
	if c.squeezeBlanks {
		sources = append(sources, squeezeBlanksAwk)
	}
	if c.showTabs {
		sources = append(sources, showTabsAwk)
	}

	progs := make([]*parser.Program, len(sources))
	for idx, src := range sources {
		debug.Printf("goawk src[%d] = %q", idx, src)
		var err error
		progs[idx], err = parser.ParseProgram(src, nil)
		if err != nil {
			return nil, err
		}
	}
	return progs, nil
}

type cat struct {
	debug bool
}

func (c cat) Run(ctx context.Context, stdio unix.StandardIO) error {
	debug := dbg.Logger(c.debug, "cat", stdio.Stderr())
	const n = 8192
	for {
		wb, err := io.CopyN(stdio.Stdout(), stdio.Stdin(), n)
		debug.Printf("written %d bytes", wb)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	debug.Printf("found io.EOF, exiting")
	return nil
}

// catNonPrinting converts non printable characters to ^ M- codes
type catNonPrinting struct{}

func (catNonPrinting) Run(ctx context.Context, stdio unix.StandardIO) error {
	var inp [4096]byte
	var out bytes.Buffer
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		n, err := stdio.Stdin().Read(inp[:])
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		nonPrinting(inp[:n], &out)
		_, err = out.WriteTo(stdio.Stdout())
		if err != nil {
			return err
		}
	}
}

func nonPrinting(inp []byte, out *bytes.Buffer) {
	out.Reset()
	for _, ch := range inp {
		if ch < 32 {
			// print TAB and \n
			if ch == 9 || ch == 10 {
				out.WriteByte(ch)
				continue
			}
			out.WriteByte('^')
			out.WriteByte(ch + 64)
			continue
		} else if ch == 127 {
			out.WriteByte('^')
			out.WriteByte('?')
			continue
		} else if ch >= 128 && ch < 160 {
			out.WriteString(`M-BM-^`)
			out.WriteByte(ch - 128 + 64)
			continue
		} else if ch >= 160 && ch < 192 {
			out.WriteString(`M-BM-`)
			out.WriteByte(ch - 128)
			continue
		} else if ch >= 192 && ch < 224 {
			out.WriteString(`M-CM-^`)
			out.WriteByte(ch - 128)
			continue
		} else if ch >= 224 {
			out.WriteString(`M-CM-`)
			out.WriteByte(ch - 192)
			continue
		}
		out.WriteByte(ch)
	}
}
