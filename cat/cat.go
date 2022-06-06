// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package cat

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/gomoni/gonix/internal"
	"github.com/gomoni/gonix/pipe"

	"github.com/benhoyt/goawk/interp"
	"github.com/benhoyt/goawk/parser"
	"github.com/spf13/pflag"
)

/*
   -A, --show-all
          equivalent to -vET

   -e     equivalent to -vE

   -t     equivalent to -vT

   -v, --show-nonprinting
          use ^ and M- notation, except for LFD and TAB
*/
type number int

const (
	None     number = 0
	NonBlank number = 1
	All      number = 2
)

type CatFilter struct {
	debug           bool
	files           []string
	showNumber      number
	showEnds        bool
	squeezeBlanks   bool
	showTabs        bool
	showNonPrinting bool
}

func Cat() *CatFilter {
	return &CatFilter{}
}

// FromArgs build a CatFilter from standard argv except the command name (os.Argv[1:])
func FromArgs(argv []string) (*CatFilter, error) {
	cmd := &CatFilter{}
	flag := pflag.FlagSet{}

	nb := flag.BoolP("number-nonblank", "b", false, "number non blank lines only")
	flag.BoolVarP(&cmd.showEnds, "show-ends", "E", false, "print $ at the end of each line")
	na := flag.BoolP("number", "n", false, "number all lines")
	flag.BoolVarP(&cmd.squeezeBlanks, "squeeze-blanks", "s", false, "ignore repeated blank lines")
	flag.Bool("u", false, "ignored, for compatibility with POSIX")
	flag.BoolVarP(&cmd.showTabs, "show-tabs", "T", false, "print TAB as ^I")

	err := flag.Parse(argv)
	if err != nil {
		return nil, err
	}
	cmd.files = flag.Args()

	// post process
	if *nb {
		cmd.ShowNumber(NonBlank)
	} else if *na {
		cmd.ShowNumber(NonBlank)
	}

	return cmd, nil
}

// Files are input files, where - denotes stdin
func (c *CatFilter) Files(f ...string) *CatFilter {
	c.files = append(c.files, f...)
	return c
}

// ShowNumber adds none all or non empty output lines
func (c *CatFilter) ShowNumber(n number) *CatFilter {
	c.showNumber = n
	return c
}

// ShowEnds add $ to the end of each line
func (c *CatFilter) ShowEnds(b bool) *CatFilter {
	c.showEnds = b
	return c
}

// SqueezeBlanks - supress repeated empty lines
func (c *CatFilter) SqueezeBlanks(b bool) *CatFilter {
	c.squeezeBlanks = b
	return c
}

// ShowTabs display TAB as ^I
func (c *CatFilter) ShowTabs(b bool) *CatFilter {
	c.showTabs = b
	return c
}

// ShowNonPrinting use ^ and M- notation, except for LFD and TAB
func (c *CatFilter) ShowNonPrinting(b bool) *CatFilter {
	c.showNonPrinting = b
	panic("not yet implemented")
}

// SetDebug additional debugging messages on stderr
func (c *CatFilter) SetDebug(debug bool) *CatFilter {
	c.debug = debug
	return c
}

func (c CatFilter) modifyStdout() bool {
	return c.showNumber != None || c.showEnds || c.squeezeBlanks || c.showTabs || c.showNonPrinting
}

func (c CatFilter) Run(ctx context.Context, stdio pipe.Stdio) error {
	debug := internal.Logger(c.debug, "cat", stdio.Stderr)
	var filters []pipe.Filter
	if !c.modifyStdout() {
		filters = []pipe.Filter{cat{debug: c.debug}}
	} else {
		progs, err := c.awk(debug)
		if err != nil {
			return err
		}
		filters = make([]pipe.Filter, len(progs))
		for idx, prog := range progs {
			filters[idx] = awkInternal{prog}
		}
	}
	if len(filters) == 0 {
		return fmt.Errorf("cat: nothing to do")
	}

	files := c.files
	if len(files) == 0 {
		files = []string{""}
	}

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
		err := pipe.Run(ctx, pipe.Stdio{
			Stdin:  in,
			Stdout: stdio.Stdout,
			Stderr: stdio.Stderr}, filters...)
		if err != nil {
			return err
		}

	}
	return nil
}

func (c CatFilter) awk(debug *log.Logger) ([]*parser.Program, error) {
	debug.Printf("c=%+v", c)
	var sources [][]byte
	if c.showEnds {
		src := []byte(`{sub(/$/, "$")}1`)
		sources = append(sources, src)
	}
	if c.showNumber == All {
		src := []byte(`
        BEGIN { n = 1; }
        {
            printf("%6d\t%s\n", n, $_);
            n++;
        }`)
		sources = append(sources, src)
	} else if c.showNumber == NonBlank {
		src := []byte(`
        BEGIN { n = 1; }
        {
            if (NF > 0) {
                printf("%6d\t%s\n", n, $_);
                n++;
            } else {
                print;
            }
        }
        `)
		sources = append(sources, src)
	}
	if c.squeezeBlanks {
		src := []byte(`
        BEGIN {
            squeeze = 0;
        }
        {
            if (NF == 0) {
                if (squeeze==0) {print};
                squeeze = 1;
            } else {
                squeeze = 0;
            }
            if (squeeze == 0) {
                print($_);
            }
        }
        `)
		sources = append(sources, src)
	}
	if c.showTabs {
		src := []byte(`{sub(/\t/, "^I")}1`)
		sources = append(sources, src)
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

// awkInternal - maybe this will morph to bigger awk command, but for know lets
// keep it here in order to reuse Run functionality of a multiple awk programs
type awkInternal struct {
	prog *parser.Program
}

func (c awkInternal) Run(ctx context.Context, stdio pipe.Stdio) error {
	config := &interp.Config{
		Stdin:  stdio.Stdin,
		Output: stdio.Stdout,
		Error:  stdio.Stderr,
	}
	_, err := interp.ExecProgram(c.prog, config)
	return err
}

type cat struct {
	debug bool
}

func (c cat) Run(ctx context.Context, stdio pipe.Stdio) error {
	debug := internal.Logger(c.debug, "cat", stdio.Stderr)
	const n = 8192
	for {
		wb, err := io.CopyN(stdio.Stdout, stdio.Stdin, n)
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
