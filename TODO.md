# TODO

Implement this?
>       >128    A command was interrupted by a signal.

 * what about tasks running other commands?
    `cat /etc/passwd | xargs -L1 timeout 2s printf "%s\n"`

 * implement and a shell scripting builtins like until?

 * Add (a basic) tr - x/tr
 * Add (a basic) tail
 * Add sort --version-sort
 * Add (a basic) grep
 * Add wrapper for goawk
 * https://github.com/itchyny/gojq
 * wc can run in a parallel

## sbase tools

sorted by a length of manual page

 *  yes
 * true/false
 * sponge
 * tee
 * seq
 * comm
 * fold
 * cmp
 * paste
 * unexpand
 * uniq
 * strings
 * env      - not implement as is, but check the options of pipe.Environ with this tool
 * tail
 * split
 * expand
 * uudecode
 * cols
 * tr
 * tsort
 * cut
 * od
 * sort
 * join
 * nl
 * grep (idea: add a gg - like ripgrep/rg first?)
 * sed
 * awk - based on goawk
 * jq - based on gojq

## GNU tools

 * fmt
 * shuf
 * numfmt
 * base32
 * base64
 * csplit
 * tac
 * timeout - do it via context(?)
 * basenc


# #bringmeback

_Following features got lost during a port on top of github.com/gomoni/gio.
Bring them back at least in a different projects_

Most unix colons exists in shell compatible format. `gonix` provides helpers which can split the shell
syntax into equivalent Go code. Code can

* ✔ control which names will be mapped into native Go code
* ✔ supports extra split function ([github.com/desertbit/go-shlex](https://github.com/desertbit/go-shlex) is probably the best)
* ✔ control what to do if command name is not found
* ✔ support  `PATH` lookups and binaries execution like shell does, but disabled by default
* ✔ control environment variables

```go
	builtins := map[string]func([]string) (pipe.Filter, error){
		"wc": func(a []string) (pipe.Filter, error) { return wc.New().FromArgs(a) },
	}
	// use real shlex code like github.com/desertbit/go-shlex
	// splitfn := func(s string) ([]string, error) { return shlex.Split(s, true) }
	splitfn := func(s string) ([]string, error) { return []string{"go", "version", "|", "wc", "-l"}, nil }
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
./gonix cat /etc/passwd /etc/resolv.conf | ./gonix cksum --algorithm md5 --untagged md5sum
```

