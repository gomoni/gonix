// Copyright Michal Vyskocil <michal.Vyskocil@gmail.com>.
// All rights reserved. Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// gen.go generates a wasm.Configer for a given unix tool - it creates a well formed Go code
// implementing the configuration for a specific tool
package main

import (
	"bytes"
	"flag"
	"go/format"
	"go/token"
	"log"
	"os"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

type Option struct {
	Text string `yaml:"text"`
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	CLI  string `yaml:"cli,omitempty"`
}

type Render struct {
	CommandLine string
	Package     string   `yaml:"package"`
	Struct      string   `yaml:"struct"`
	Options     []Option `yaml:"options"`
}

func (r Render) Capacity() int {
	return len(r.Options) + 1
}

const plate = `// DO NOT MODIFY THIS FILE
// generated via
//      go generate ./...
// which executed
//      go run wasm/internal/gen/gen.go {{.CommandLine}}
package {{.Package}}

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/tetratelabs/wazero"
)

//WebAssembly from suckless/sbase project
//go:embed {{.Package}}.wasm
var wasm []byte

func Compile(ctx context.Context, r wazero.Runtime) (wazero.CompiledModule, error) {
	if len(wasm) == 0 {
		return nil, fmt.Errorf("{{.Package}}.wasm is empty")
	}
	return r.CompileModule(ctx, wasm)
}

type {{.Struct}} struct {
    {{- range .Options}}
        {{goprivate .Name}} {{.Type}}
    {{- end}}
}

// New returns new wasm.Configer for command {{.Package}}
func New() *{{.Struct}} {
    return &{{.Struct}}{}
}
{{range .Options}}
// {{.Name}}: set {{.Text}}(s)
func (c *{{$.Struct}}) {{.Name}}(x {{.Type}}) *{{$.Struct}} {
    c.{{goprivate .Name}} = x
    return c
}
    {{if eq .Type "[]string"}}
// Append{{.Name}}: append {{.Text}}(s)
func (c *{{$.Struct}}) Append{{.Name}}(x ...{{unslice .Type}}) *{{$.Struct}} {
    c.{{goprivate .Name}} = append(c.{{goprivate .Name}}, x...)
    return c
}
    {{end}}
{{end}}

// Config provides a command line arguments for an underlying command
func (c {{.Struct}}) Config() wazero.ModuleConfig {
	return wazero.NewModuleConfig().WithArgs(c.args()...)
}

func (c {{.Struct}}) args() []string {
	args := make([]string, 1, {{.Capacity}})
	args[0] = "{{.Package}}"

    {{range .Options}}
        {{if eq .Type "bool"}}
        if c.{{goprivate .Name}} {
            args = append(args, "{{.CLI}}")
        }
        {{else if eq .Type "string"}}
        if c.{{goprivate .Name}} != "" {
            {{- if ne .CLI ""}}
            args = append(args, "{{.CLI}}")
            {{- end}}
            args = append(args, c.{{goprivate .Name}})
        }
        {{else if eq .Type "[]string"}}
        for _, x := range c.{{goprivate .Name}} {
            {{- if ne .CLI ""}}
            args = append(args, "{{.CLI}}")
            {{- end}}
            args = append(args, x)
        }
        {{else}}
        panic("Option {{.Name}} have unsupported type {{.Type}}")
        {{end}}
    {{end}}
    return args
}
`

func main() {

	inputp := flag.String("i", "", "input yaml description")
	outputp := flag.String("o", "", "output go file")
	flag.Parse()

	if *inputp == "" {
		log.Fatal("input yaml -i is empty")
	}

	f, err := os.Open(*inputp)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	out := os.Stdout
	if *outputp != "" {
		out, err = os.Create(*outputp)
		if err != nil {
			log.Fatal(err)
		}
	}
	defer out.Close()

	var render Render
	err = yaml.NewDecoder(f).Decode(&render)
	if err != nil {
		log.Fatal(err)
	}

	render.CommandLine = strings.Join(os.Args[1:], " ")
	funcs := map[string]any{
		"goprivate": func(s string) string { return goprivate(s) },
		"singular":  func(s string) string { return singular(s) },
		"unslice":   func(s string) string { return unslice(s) },
	}

	t, err := template.New("config.go").Funcs(funcs).Parse(plate)
	if err != nil {
		log.Fatal(err)
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, render)
	if err != nil {
		log.Fatal(err)
	}

	b, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	_, err = out.Write(b)
	if err != nil {
		log.Fatal(err)
	}
}

type builtins []string

var b = builtins{
	"append", "new", "map", "delete",
}

func (b builtins) Is(s string) bool {
	for _, x := range b {
		if s == x {
			return true
		}
	}
	return false
}

func goprivate(s string) string {
	rn, _ := utf8.DecodeRuneInString(s)
	ret := strings.Replace(s, string(rn), string(unicode.ToLower(rn)), 1)
	if token.IsKeyword(ret) || b.Is(ret) {
		ret = "_" + ret
	}
	return ret
}

func singular(s string) string {
	return strings.TrimRight(s, "s")
}

func unslice(s string) string {
	return strings.TrimLeft(s, "[]")
}
