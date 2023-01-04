// package wasm: a support for filters implemented as a web assembly
package wasm

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/gomoni/gonix/pipe"
	"github.com/tetratelabs/wazero"
)

type Configer interface {
	Config() wazero.ModuleConfig
}

type Filter struct {
	runtime  wazero.Runtime
	code     wazero.CompiledModule
	configer Configer
}

// New binds the runtime, compiled module and a config to Filter
// implementing the Filter interface
func New(runtime wazero.Runtime, code wazero.CompiledModule, configer Configer) *Filter {
	return &Filter{
		runtime:  runtime,
		code:     code,
		configer: configer,
	}
}

func (f Filter) Run(ctx context.Context, stdio pipe.Stdio) error {
	m, err := f.runtime.InstantiateModule(
		ctx,
		f.code,
		f.configer.Config().
			WithStdin(stdio.Stdin).
			WithStderr(stdio.Stderr).
			WithStdout(stdio.Stdout).
			WithName(fmt.Sprintf("wasm-%d", rand.Int63())),
	)
	if err != nil {
		return err
	}
	defer m.Close(ctx)
	return nil
}
