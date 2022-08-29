# gonix: unix as a Go library

Unix text utilities implemented in pure Go and an excellent [github.com/benhoyt/goawk](https://github.com/benhoyt/goawk)

 * ✔ Go library
 * ✔ Native pipes in Go
 * ✔ Flexible shell colon parsing and execution
 * ✔ Busybox-like command line tool
 * ⚠ not yet guaranteed to be stable, API may change

# Go library

Each tool can be called from Go code.

```go
	// printf "three\nsmall\npigs\n" | head --lines 2
	head := head.New().Lines(2)
	err := head.Run(context.TODO(), pipe.Stdio{
		Stdin:  io.NopCloser(bytes.NewBufferString("three\nsmall\npigs\n")),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// three
	// small
```

# Native pipes in Go

Unix is unix because of a `pipe(2)` allowing seamless combination of all unix filters into longer colons.
`gonix` has `pipe.Run` allowing to connect and arbitrary number of filters. It connects stdint/stdout
automatically like unix `sh(1)` do.



```go
	// printf "three\nsmall\npigs\n" | cat | wc -l
	stdio := pipe.Stdio{
		Stdin:  io.NopCloser(bytes.NewBufferString("three\nsmall\npigs\n")),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	err := pipe.Run(context.TODO(), stdio, cat.New(), wc.New().Lines(true))
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// 3
```

## Flexible shell colon parsing and execution

Most unix colons exists in shell compatible format. `gonix` provides helpers which can split the shell
syntax into equivalent Go code. Code can

* ✔ control which names will be mapped into native Go code
* ✔ supports own split function (while [github.com/desertbit/go-shlex](https://github.com/desertbit/go-shlex) is probably the best)
* ✔ control what to do if command name is not found
* ✔ support  `PATH` lookups and binaries execution like shell does, disabled by default
* ✔ control environment variables

```go
	builtins := map[string]func([]string) (pipe.Filter, error){
		"wc": func(a []string) (pipe.Filter, error) { return wc.New().FromArgs(a) },
	}
	splitfn := func(s string) ([]string, error) { return shlex.Split(s, true) }
	stdio := pipe.Stdio{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	ctx := context.Background()

	env := pipe.DuplicateEnviron()
	sh := pipe.NewSh(builtins, splitfn).NotFoundFunc(env.NotFoundFunc)
	err := sh.Run(ctx, stdio, `go version | wc -l`)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// 1
```

## Busybox-like command line tool

Can be built and executed like busybox or a toybox.

```sh
./gonix cat /etc/passwd /etc/resolv.conf | md5sum
```

# Builtins

 * cat -uses [goawk](https://github.com/benhoyt/goawk)
 * cksum - POSIX ctx, md5 and sha check sums, runs concurrently (`-j/--threads`) by default
 * wc - word count
 * head -n/--lines - uses [goawk](https://github.com/gomoni/gonix/blob/main/head/head_negative.awk)
 
# Other interesting projects
 * ♥ [github.com/benhoyt/goawk](https://github.com/benhoyt/goawk) an excellent awk implementation for Go
 * [github.com/desertbit/go-shlex](https://github.com/desertbit/go-shlex) probably the best sh lexing library for Go
 * [github.com/u-root/u-root](https://github.com/u-root/u-root) full Go userland for bootloaders, similar idea, not providing a library
