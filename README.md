# gonix: unix as a Go library

Unix text utilities implemented in pure Go, using
[github.com/gomoni/gio/unix](https://github.com/gomoni/gio/blob/main/unix/unix.go)
and an excellent [github.com/benhoyt/goawk](https://github.com/benhoyt/goawk)

 * ⚠ not yet guaranteed to be stable, API and a project layout MAY change
 * ✔ Go library
 * ✔ Native pipes in Go

# Native filters

 * awk - a thin wrapper for [goawk](https://github.com/benhoyt/goawk)
 * cat -uses [goawk](https://github.com/benhoyt/goawk)
 * cksum - POSIX ctx, md5 and sha check sums, runs concurrently (`-j/--threads`) by default
 * head -n/--lines - uses [goawk](https://github.com/gomoni/gonix/blob/main/head/head_negative.awk)
 * wc - word count

# Work in progress

 * x/tr - translate characters

# Go library

Each filter can be called from Go code.

```go
	head := head.New().Lines(2)
	err := head.Run(context.TODO(), unix.NewStdio(
		bytes.NewBufferString("three\nsmall\npigs\n"),
		os.Stdout,
		os.Stderr,
	))
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// three
	// small
```

# Native pipes in Go

Unix is unix because of a `pipe(2)` allowing a seamless combination of all unix filters into longer colons.
`gonix` has `pipe.Run` allowing to connect and arbitrary number of filters. It connects stdin/stdout
automatically like unix `sh(1)` do.



```go
	// printf "three\nsmall\npigs\n" | cat | wc -l
	err := unix.NewLine().Run(ctx, stdio, cat.New(), wc.New().Lines(true))
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// 3
```

# Architecture of a filter

1. Each command is represented as Go struct
2. New() returns a pointer to zero structure, no default values are passed in
3. Optional `FromArgs([]string)(*Struct, error)` provides cli parsing and implements defaults
4. It does defer most of runtime errors to `Run` method
5. `Run(context.Context, pipe.Stdio) error` method gets a _value receiver_ so it never changes the configuration

```go
// wc does nothing, as it has all zeroes - an equivalent of &wc.Wc{} or new(Wc)
wc := wc.New()
// wc gets Lines(true) Chars(true) Bytes(true)
wc, err := wc.FromArgs(nil)
// wc gets chars(false)
wc = wc.Chars(false)
// wc is a value receiver, so never changes the configuration
err = wc.Run(...)
```

## Internal helpers

`internal.RunFiles` abstracts running a command over stdin (and) or list of
files. Takes a care about opening and proper closing the files, does errors
gracefully, so they do not cancel the code to run, but are propagated to caller
properly. Supports a parallel execution of tasks via `internal.PMap` so `cksum`
run in a parallel by default.

`internal.PMap` is a parallel map algorithm. Executes MapFunc, which converts
input slices to output slice and each execution is capped by maximum number of
threads. It maintains the order.

`internal.Unit` and `internal.Byte` is a fork of time.Duration of stdlib, which
supports bigger ranges (based on float64). New units can be easily defined on
top of Unit type.

## Testing

The typical testing is very repetitive, so there is a common structure for build of
table tests. It uses generics to improve a type safety.

```
import "github.com/gomoni/gonix/internal/test"

	testCases := []test.Case[Wc]{
		{
			Name:     "wc -l",
			Filter:   New().Lines(true),
			FromArgs: fromArgs(t, []string{"-l"}),
			Input:    "three\nsmall\npigs\n",
			Expected: "3\n",
		},
    }
	test.RunAll(t, testCases)
```

Where the struct fields are

* Name is name of test case to be printed by go test
* Input is a string input for a particular command
* Expected is what command is supposed to generate
* Filter is a definition of a filter
* FromArgs is an alternative definition obtained by `FromArgs` helper. It
  ensures CLI parsing is tested as a part of regular functional testing

## Testing with real files

WIP atm, there is `test.TestData` helper and a bunch of code in
`cksum/cksum_test.go` to run tests using real files.
 
# Other interesting projects
 * [github.com/benhoyt/goawk](https://github.com/benhoyt/goawk) an excellent awk implementation for Go
 * [https://github.com/mvdan/sh](https://github.com/mvdan/sh) shell parser formater and interpreter
 * [github.com/desertbit/go-shlex](https://github.com/desertbit/go-shlex) probably the best sh lexing library for Go
 * [github.com/u-root/u-root](https://github.com/u-root/u-root) full Go userland for bootloaders, similar idea, not providing a library

