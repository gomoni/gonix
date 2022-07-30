# TODO
 
>       >128    A command was interrupted by a signal.

 * do one need to use ReadCloser for Stdio?

 * TODO: should we pass environ to builtins too?
        idea `FromArgs(env Environ, args ...string)` ???

 * race detector - should be firstNonZero variable atomic?
 * split common things from `head/head.go` to `internal`

 * what about tasks running other commands?
    `cat /etc/passwd | xargs -L1 timeout 2s printf "%s\n"`

 * implement and a shell scripting builtins like until?

 * Add (a basic) tr
 * Add (a basic) tail
 * Add sort --version-sort
 * Add (a basic) grep
 * Add wrapper for goawk
 * check if we can transform things from u-root
