# gonix

Unix userspace written in Go with pipes (and redirection in the future)

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

## As a command line

```sh
./gonix cat /etc/passwd /etc/resolv.conf | md5sum
```
