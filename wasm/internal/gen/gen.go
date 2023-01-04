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
	CLI  string `yaml:"cli"`
}

type Render struct {
	CommandLine string
	Package     string
	Struct      string
	Options     []Option
	Arguments   []Option //XXX: reuse Option type
}

func (r Render) Capacity() int {
	return len(r.Options) + len(r.Arguments) + 1
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
    {{- range .Arguments}}
        {{goprivate .Name}} {{.Type}}
    {{- end}}
}

// New returns new wasm.Configer for command {{.Package}}
func New() *{{.Struct}} {
    return &{{.Struct}}{}
}
{{range .Options}}
// {{.Name}}: {{.Text}}
func (c *{{$.Struct}}) {{.Name}}(x {{.Type}}) *{{$.Struct}} {
    c.{{goprivate .Name}} = x
    return c
}
{{end}}
{{range .Arguments}}
// {{.Name}}: {{.Text}}
func (c *{{$.Struct}}) {{.Name}}(x {{.Type}}) *{{$.Struct}} {
    c.{{goprivate .Name}} = x
    return c
}
{{end}}

// Config provides a command line arguments for an underlying command
func (c {{.Struct}}) Config() wazero.ModuleConfig {
	return wazero.NewModuleConfig().WithArgs(c.args()...)
}

func (c Tr) args() []string {
	args := make([]string, 1, {{.Capacity}})
	args[0] = "{{.Package}}"

    {{range .Options}}
        {{if eq .Type "bool"}}
        if c.{{goprivate .Name}} {
            args = append(args, "{{.CLI}}")
        }
        {{else}}
        panic("Option {{.Name}} have unsupported type {{.Type}}")
        {{end}}
    {{end}}

    {{range .Arguments}}
        {{if eq .Type "string"}}
        if c.{{goprivate .Name}} != "" {
            args = append(args, c.{{goprivate .Name}})
        }
        {{else}}
        panic("Argument {{.Name}} have unsupported type {{.Type}}")
        {{end}}
    {{end}}
    return args
}
`

func main() {

	inputp := flag.String("i", "", "input yaml description")
	packagep := flag.String("p", "", "go package name")
	outputp := flag.String("o", "", "output go file")
	flag.Parse()

	if *inputp == "" {
		log.Fatal("input yaml -i is empty")
	}

	if *packagep == "" {
		log.Fatal("name -n is empty")
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

	var description map[string][]Option
	err = yaml.NewDecoder(f).Decode(&description)
	if err != nil {
		log.Fatal(err)
	}

	render := Render{
		CommandLine: strings.Join(os.Args[1:], " "),
		Package:     *packagep,
		Struct:      gopublic(packagep),
		Options:     description["options"],
		Arguments:   description["arguments"],
	}

	funcs := map[string]any{
		"goprivate": func(s string) string { return goprivate(s) },
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

func gopublic(sp *string) string {
	rn, _ := utf8.DecodeRuneInString(*sp)
	ret := strings.Replace(*sp, string(rn), string(unicode.ToUpper(rn)), 1)
	return ret
}
