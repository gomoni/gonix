// DO NOT MODIFY THIS FILE
// generated via
//
//	go generate ./...
//
// which executed
//
//	go run wasm/internal/gen/gen.go -i tr.yaml -p tr -o tr.go
package tr

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/tetratelabs/wazero"
)

// WebAssembly from suckless/sbase project
//
//go:embed tr.wasm
var wasm []byte

func Compile(ctx context.Context, r wazero.Runtime) (wazero.CompiledModule, error) {
	if len(wasm) == 0 {
		return nil, fmt.Errorf("tr.wasm is empty")
	}
	return r.CompileModule(ctx, wasm)
}

type Tr struct {
	complement bool
	_delete    bool
	squeeze    bool
	set1       string
	set2       string
}

// New returns new wasm.Configer for command tr
func New() *Tr {
	return &Tr{}
}

// Complement: Match to set1 complement
func (c *Tr) Complement(x bool) *Tr {
	c.complement = x
	return c
}

// Delete: Delete characters matching
func (c *Tr) Delete(x bool) *Tr {
	c._delete = x
	return c
}

// Squeeze: Squeeze repeated characters matching set1 or set2
func (c *Tr) Squeeze(x bool) *Tr {
	c.squeeze = x
	return c
}

// Set1: Define set1
func (c *Tr) Set1(x string) *Tr {
	c.set1 = x
	return c
}

// Set2: Define set2
func (c *Tr) Set2(x string) *Tr {
	c.set2 = x
	return c
}

// Config provides a command line arguments for an underlying command
func (c Tr) Config() wazero.ModuleConfig {
	return wazero.NewModuleConfig().WithArgs(c.args()...)
}

func (c Tr) args() []string {
	args := make([]string, 1, 6)
	args[0] = "tr"

	if c.complement {
		args = append(args, "-c")
	}

	if c._delete {
		args = append(args, "-d")
	}

	if c.squeeze {
		args = append(args, "-s")
	}

	if c.set1 != "" {
		args = append(args, c.set1)
	}

	if c.set2 != "" {
		args = append(args, c.set2)
	}

	return args
}
