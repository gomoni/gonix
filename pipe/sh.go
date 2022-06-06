package pipe

import (
	"context"
	"fmt"
)

type SplitFn func(string) ([]string, error)
type Tools = map[string]func([]string) (Filter, error)

type Sh struct {
	tools   Tools
	splitfn SplitFn
	debug   bool
}

func NewSh(tools Tools, splitfn SplitFn) Sh {
	return Sh{
		tools:   tools,
		splitfn: splitfn,
	}
}

func (s Sh) Parse(cmdline string) ([]Filter, error) {
	args, err := s.splitfn(cmdline)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("nothing to do")
	}

	filters := make([]Filter, 0, 16)
	start := 0
	for {
		if start > len(args) {
			break
		}
		arg0 := args[start]
		fromArgs, ok := s.tools[arg0]
		if !ok {
			return nil, fmt.Errorf("exec not yet implemented")
		}

		var argn []string
		if start+1 == len(args) {
			argn = []string{}
		} else {
			stop := start + nextPipe(args[start:])
			argn = args[start+1 : stop]
			start = stop + 1
		}
		filter, err := fromArgs(argn)
		if err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}
	return filters, nil
}

func (s Sh) Run(ctx context.Context, stdio Stdio, cmdline string) error {
	filters, err := s.Parse(cmdline)
	if err != nil {
		return err
	}
	return Run(ctx, stdio, filters...)
}

func nextPipe(args []string) int {
	for idx, a := range args {
		if a == "|" {
			return idx
		}
	}
	return len(args)
}
