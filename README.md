# gonix

Unix textutils implemented in pure Go including pipe support, execution of
external programs and highly configurable "shell" environment. Compiles to
busybox-like CLI program as well as usable as a Go library.

> Use shell colons from real programming language

# How to install

Beware: it is in alpha status - some aspects can change.

```
git clone github.com/gomoni/gonix
```

## Run unix shell colon inside Go

Provided the name to Filter mapping and a custom split function, code parses
the command line, builds a slice of filters and executes them. Both `cat` and
`wc` are implemented in Go compatible with GNU counterparts.

```go
// run builtin cat and wc exported as Go native structs with custom stdio
var out bytes.Buffer
stdio := pipe.Stdio {
    Stdin: os.Stdin,
    Stdout: out,
    Stderr: os.Stderr,
}

ctx := context.Background()
sh := pipe.NewSh(builtins, splitfn)
err := sh.Run(ctx, stdio, `cat | wc -l`)
```

## Without string parsing

This is an equivalent without string parsing.

```go
err = pipe.Run(ctx, stdio, cat.New(), wc.New().Lines(true))
```

## Execute programs + environment

Gonix does not execute unknown programs by default. `pipe.Environ` helper allows
a detailed specification of an environment and passing it to executable.

```go
ctx := context.Background()
env := pipe.DuplicateEnviron()
sh := pipe.NewSh(builtins, splitfn).NotFoundFunc(env.NotFoundFunc)
err := sh.Run(ctx, stdio, `go version | wc -l`)
```

## As a command line

Can be built and run like busybox or a toybox.

```sh
./gonix cat /etc/passwd /etc/resolv.conf | md5sum
```

## builtins

 * cat - the basic command - uses goawk under the hood
 * wc - word count
