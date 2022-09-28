package tr

import (
	"bufio"
	"context"
	"io"
	"unicode"
	"unicode/utf8"

	"github.com/gomoni/gonix/internal"
	"github.com/gomoni/gonix/pipe"
)

/*
   tr - translate or replace runes

   Working on runes makes it backward compatible with POSIX tr and supports
   utf-8 well. Ignores unicode combining characters though, user is expected to use NFC
   forms of input.
*/

/*
Notes for an implementation:
 1. translate array1 -> array2
      if len(array1) > len(array2) - make array2 repeat last rune
      ix -> X becomes {'i': 'X', 'x': 'X'}
 2. translate a complement
 3. delete
 4. squeeze
 5. \NNN + \\ \a et \v
 6. CHAR1-CHAR2
 7. [CHAR*]
 8. [CHAR*REPEAT]
 9. [:alnum:] to [:xdigit:]
 10. equivalence classes[=CHAR=]
	Although equivalence classes are intended to support non-English alphabets, there seems to be no standard way to define them or determine their contents. Therefore, they are not fully implemented in GNU tr; each character’s equivalence class consists only of that character, which is of no particular use.

   can be implemented in terms of https://en.wikipedia.org/wiki/Unicode_equivalence

 tr: when translating, the only character classes that may appear in
  string2 are 'upper' and 'lower'
 so [:upper:] -> [:lower:] or vice versa is all what is enabled
 tr '1[:upper:]' [:lower:] -> misaligned
 tr '1[:upper:]' 2[:lower:] is fine
 tr 'h[:upper:]' X[:lower:] is fine
 tr '[:lower:]e' [:upper:]x -> first replace lower case to upper case except e, which is replaced by x        -> HxLLO WORLD

 combining leads to weird results
 "2 Hello world" | tr [:alpha:][:digit:] XY
  Y YYYYY YYYYY
*/

type Tr struct {
	debug      bool
	array1     string
	array2     string
	complement bool // use complement of ARRAY1
	del        bool // delete characters in ARRAY1
	truncate   bool
	files      []string
}

func New() *Tr {
	return &Tr{}
}

func (c *Tr) Array1(in string) *Tr {
	c.array1 = in
	return c
}

func (c *Tr) Complement(b bool) *Tr {
	c.complement = b
	return c
}

func (c *Tr) Delete(b bool) *Tr {
	c.del = b
	return c
}

func (c Tr) Run(ctx context.Context, stdio pipe.Stdio) error {
	//c.debug = true
	//debug := dbg.Logger(c.debug, "tr", stdio.Stderr)
	var chain chain
	if c.del {
		if !c.complement {
			chain = newChain(newDeleteTr(c.array1))
		} else {
			chain = newChain(newDeleteComplementTr(c.array1))
		}
	} else {
		panic("tr without -d/--delete is not yet implemented")
	}

	tr := func(ctx context.Context, stdio pipe.Stdio, _ int, _ string) error {
		scanner := bufio.NewScanner(stdio.Stdin)
		stdout := bufio.NewWriterSize(stdio.Stdout, 4096)
		defer stdout.Flush()
		scanner.Split(bufio.ScanRunes)
		for scanner.Scan() {
			if scanner.Err() != nil {
				return scanner.Err()
			}
			in, _ := utf8.DecodeRuneInString(scanner.Text())
			rn, _ := chain.Tr(in)
			if rn == -1 {
				continue
			}
			_, err := writeRune(stdout, rn)
			if err != nil {
				return err
			}
		}
		return nil
	}
	runFiles := internal.NewRunFiles(c.files, stdio, tr)
	return runFiles.Do(ctx)
}

type tr interface {
	// Tr translate rune to other rune and returns true if it was done
	Tr(rune) (rune, bool)
}

// mapTr maps rune->other-rune
//
//	tr 'a-z' 'G'
type mapTr map[rune]rune

func (t mapTr) Tr(in rune) (rune, bool) {
	out, found := t[in]
	return out, found
}

// complementMapTr maps !rune(s)->other-rune
//
//	tr --complement 'a-z' 'G'
//
// FIXME: in theory you can map complement to set, not a single rune!!
type mapComplementTr struct {
	tr   map[rune]struct{}
	dest rune
}

func newMapComplementTr(in []rune, dest rune) mapComplementTr {
	tr := make(map[rune]struct{}, len(in))
	for _, r := range in {
		tr[r] = struct{}{}
	}
	return mapComplementTr{
		tr:   tr,
		dest: dest,
	}
}

func (t mapComplementTr) Tr(in rune) (rune, bool) {
	_, found := t.tr[in]
	if found {
		return in, false
	}
	return t.dest, true
}

// trFunc calls a specific function for each step
type trFunc func(rune) (rune, bool)

func (t trFunc) Tr(in rune) (rune, bool) {
	return t(in)
}

type tr2 struct {
	predicate func(rune) bool
	transform func(rune) rune
}

func (t tr2) Tr(in rune) (rune, bool) {
	if t.predicate(in) {
		return t.transform(in), true
	}
	return in, false
}

var _ltu = tr2{
	predicate: unicode.IsLower,
	transform: unicode.ToUpper,
}

// lowerToUpper is tr '[:lower:]' '[:upper:]'
func lowerToUpper(in rune) (rune, bool) {
	return _ltu.Tr(in)
}

var _utl = tr2{
	predicate: unicode.IsUpper,
	transform: unicode.ToLower,
}

// upperToLower is tr '[:upper:]' '[:lower:]'
func upperToLower(in rune) (rune, bool) {
	return _utl.Tr(in)
}

// makeSet tr '[:alpha:'] X
func makeSetTr(predicate func(rune) bool, dest rune) trFunc {
	var tr = tr2{
		predicate: predicate,
		transform: func(in rune) rune { return dest },
	}
	return tr.Tr
}

// delete
type deleteTr struct {
	tr map[rune]struct{}
}

func newDeleteTr(array1 string) deleteTr {
	return deleteTr{
		tr: strToRunes(array1),
	}
}

func (t deleteTr) Tr(in rune) (rune, bool) {
	if _, ok := t.tr[in]; ok {
		return -1, true
	}
	return in, true
}

// delete complement
type deleteComplementTr struct {
	tr map[rune]struct{}
}

func newDeleteComplementTr(array1 string) deleteComplementTr {
	return deleteComplementTr{
		tr: strToRunes(array1),
	}
}

func (t deleteComplementTr) Tr(in rune) (rune, bool) {
	if _, ok := t.tr[in]; !ok {
		return -1, true
	}
	return in, true
}

// squeeze \NNN \\ \a et all

type chain struct {
	trs []tr
}

func newChain(in ...tr) chain {
	return chain{
		trs: in,
	}
}

func (t chain) Tr(in rune) (rune, bool) {
	for _, tr := range t.trs {
		dst, found := tr.Tr(in)
		if !found {
			continue
		}
		return dst, true
	}
	// pass
	return in, true
}

func trString(in string, chain chain, out io.Writer) error {
	var err error

	for _, rn := range in {
		dst, _ := chain.Tr(rn)
		if dst == -1 {
			continue
		}
		_, err = writeRune(out, dst)
		if err != nil {
			return err
		}
	}
	return nil
}

// https://cs.opensource.google/go/go/+/refs/tags/go1.19.1:src/strings/builder.go;l=104
// WriteRune appends the UTF-8 encoding of Unicode code point r to b's buffer.
// It returns the length of r and a nil error.
func writeRune(w io.Writer, r rune) (int, error) {
	// Compare as uint32 to correctly handle negative runes.
	if uint32(r) < utf8.RuneSelf {
		return w.Write([]byte{byte(r)})
	}

	var buf [utf8.UTFMax]byte
	n := utf8.EncodeRune(buf[:], r)
	return w.Write(buf[:n])
}

func strToRunes(in string) map[rune]struct{} {
	ret := make(map[rune]struct{}, len(in))
	for _, rn := range in {
		ret[rn] = struct{}{}
	}
	return ret
}
