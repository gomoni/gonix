package tr

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/tetratelabs/wazero"
)

//go:embed tr.wasm
var wasm []byte

func Compile(ctx context.Context, r wazero.Runtime) (wazero.CompiledModule, error) {
	if len(wasm) == 0 {
		return nil, fmt.Errorf("tr.wasm is empty")
	}
	return r.CompileModule(ctx, wasm)
}

type Tr struct {
	set1       string
	set2       string
	del        bool
	complement bool
	squeeze    bool
}

func New() *Tr {
	return &Tr{}
}
func (c *Tr) Set1(set1 string) *Tr {
	c.set1 = set1
	return c
}
func (c *Tr) Set2(set2 string) *Tr {
	c.set2 = set2
	return c
}
func (c *Tr) Delete(b bool) *Tr {
	c.del = b
	return c
}
func (c *Tr) Complement(b bool) *Tr {
	c.complement = b
	return c
}
func (c *Tr) Squeeze(b bool) *Tr {
	c.squeeze = b
	return c
}

func (t Tr) Config() wazero.ModuleConfig {
	return wazero.NewModuleConfig().WithArgs(t.args()...)
}

func (c Tr) args() []string {
	args := make([]string, 1, 6)
	args[0] = "tr"
	if c.complement {
		args = append(args, "-c")
	}
	if c.del {
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
