package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/gomoni/gonix/cat"
	"github.com/gomoni/gonix/pipe"
	"github.com/gomoni/gonix/wc"
)

var tools map[string]func([]string) (pipe.Filter, error)

func init() {
	tools = map[string]func([]string) (pipe.Filter, error){
		"cat": func(a []string) (pipe.Filter, error) { return cat.New().FromArgs(a) },
		"wc":  func(a []string) (pipe.Filter, error) { return wc.New().FromArgs(a) },
	}
}

func main() {

	args := os.Args
	if len(args) == 0 {
		panic("what to do here")
	}
	// strip gonix
	if filepath.Base(args[0]) == "gonix" {
		args = args[1:]
	}

	fromArgs, found := tools[args[0]]
	if !found {
		panic("exec not yet implemented")
	}
	filter, err := fromArgs(args[1:])
	if err != nil {
		log.Fatal(err)
	}
	// implement ctrl+c support
	err = filter.Run(
		context.Background(),
		pipe.Stdio{
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		},
	)

	if err != nil {
		log.Fatal(err)
	}

}
