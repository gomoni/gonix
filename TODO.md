# TODO

 * P0: exit codes - those are must for cmdlines (eg grep)
        use (int, error) or encode non zero error code in error?
        what are the interactions of a pipe and non zero errors?

 * Exec/Sh and how to deal with env variables
    eg
    // Inherit inherits given variable names
    func (sh *Sh) InheritEnv([]string) *Sh
    // ???
 * GHA + golangci-lint
 * unit tests `pipe/sh_test.go`
 * unit tests `pipe/exec_test.go`
 * Add (a basic) tr
 * Add (a basic) head/tail
 * Add sort --version-sort
 * Add (a basic) grep
 * Add wrapper for goawk
