# TODO
 
>       >128    A command was interrupted by a signal.

 * do one need to use ReadCloser for Stdio?

 * Exec: environment handling dome - helper pipe.Environ
 * Sh propagate it to pipe.Exec
 * TODO: should NotFoundFunc belong to Environ struct?
 * TODO: should we pass environ to builtins too?
        idea `FromArgs(env Environ, args ...string)` ???

 * race detector - should be firstNonZero variable atomic?
 * split common things from `head/head.go` to `internal`

 * what about tasks running other commands?
    `cat /etc/passwd | xargs -L1 timeout 2s printf "%s\n"`

 * implement and a shell scripting builtins like until?

 * GHA + golangci-lint
 * Add (a basic) tr
 * Add (a basic) head/tail
 * Add sort --version-sort
 * Add (a basic) grep
 * Add wrapper for goawk
 * check if we can transform things from u-root
