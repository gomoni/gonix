# TODO

 * exit codes

 man 1p exit

>       As explained in other sections, certain exit status values have been reserved for special uses and should be used by applications only for those purposes:
>
>        126    A file to be executed was found, but it was not an executable utility.
>
>        127    A utility to be executed was not found.
>
>       >128    A command was interrupted by a signal.
>
>       The  behavior  of  exit  when given an invalid argument or unknown option is unspecified, because of differing practices in the various historical implementations. A value larger
>       than 255 might be truncated by the shell, and be unavailable even to a parent process that uses waitid() to get the full exit value. It is recommended that  implementations  that
>       detect  any usage error should cause a non-zero exit status (or, if the shell is interactive and the error does not cause the shell to abort, store a non-zero value in "$?"), but
>       even this was not done historically in all shells.


go doc os.Exit
> package os // import "os"
>
> func Exit(code int)
>    ...
>
>   For portability, the status code should be in the range [0, 125].


 * do one need to use ReadCloser for Stdio?

 * Exec/Sh and how to deal with env variables
    eg
    // Inherit inherits given variable names
    func (sh *Sh) InheritEnv([]string) *Sh
    // ???

 * GHA + golangci-lint
 * Add (a basic) tr
 * Add (a basic) head/tail
 * Add sort --version-sort
 * Add (a basic) grep
 * Add wrapper for goawk
