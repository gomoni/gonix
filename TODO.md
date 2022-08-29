# TODO
 
>       >128    A command was interrupted by a signal.

 * do one need to use ReadCloser for Stdio?

 * TODO: should we pass environ to builtins too?
        idea `FromArgs(env Environ, args ...string)` ???

 * race detector - should be firstNonZero variable atomic?

 * what about tasks running other commands?
    `cat /etc/passwd | xargs -L1 timeout 2s printf "%s\n"`

 * implement and a shell scripting builtins like until?

 * Add (a basic) tr
 * Add (a basic) tail
 * Add sort --version-sort
 * Add (a basic) grep
 * Add wrapper for goawk
 * https://github.com/itchyny/gojq
 * check if we can transform things from u-root

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
