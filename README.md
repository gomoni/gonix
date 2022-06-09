# gonix

Unix textutils implemented in pure Go, with native pipe and shell parsing
support. Joins shell with a real programming language.

## builtins

 * cat - the basic command - uses goawk under the hood
 * wc - word count


## As a Go library

```go
import "gonix/pipe"

// run builtin cat and wc exported as Go native structs with custom stdio
var out bytes.Buffer
stdio := pipe.Stdio {
    Stdin: os.Stdin,
    Stdout: out,
    Stderr: os.Stderr,
}
err = pipe.Run(ctx, stdio, gonix.Cat{}, gonix.Wc{}.Lines())
```

## Parse and run shell colon

Provided the name to Filter mapping and a custom split function, code
parses the command line, builds a slice of filters and executes them.

```go
ctx := context.Background()
sh := pipe.NewSh(builtins, splitfn)
err := sh.Run(ctx, stdio, `cat | wc -l`)
```

## Execute unknown programs

`AllowExec(true)` execute external command instead of returning an error
about not found builtin.

```go
ctx := context.Background()
sh := pipe.NewSh(builtins, splitfn).AllowExec(true)
err := sh.Run(ctx, stdio, `go version | wc -l`)
```

## As a command line

Can be built and run like busybox or a toybox.

```sh
./gonix cat /etc/passwd /etc/resolv.conf | md5sum
```
