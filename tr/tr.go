package tr

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gomoni/gonix/internal"
	"github.com/gomoni/gonix/internal/dbg"
	"github.com/gomoni/gonix/pipe"
)

/*
   tr - translate or replace runes

   Working on runes makes it backward compatible with POSIX tr and supports
   utf-8 well. Ignores unicode combining characters though, user is expected to use NFC
   forms of input.

   Status:
   * DONE:   --delete and --delete --complement for all characters, character sets and escape characters
   * TODO: translate aka ARRAY2
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
	//truncate   bool       // TODO
	files []string
}

func New() *Tr {
	return &Tr{}
}

func (c *Tr) Array1(in string) *Tr {
	c.array1 = in
	return c
}

func (c *Tr) Array2(in string) *Tr {
	c.array2 = in
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
	c.debug = true
	debug := dbg.Logger(c.debug, "tr", stdio.Stderr)
	var chain chain
	if c.del {
		trs, err := c.makeDelChain(c.array1)
		if err != nil {
			return err
		}
		chain.trs = trs
		debug.Printf("trs=%#v", trs)
	} else {
		if c.complement {
			panic("--complement for transate is not implemented")
		}
		trs, err := c.makeTrChain(c.array1, c.array2)
		if err != nil {
			return err
		}
		chain.trs = trs
		debug.Printf("trs=%#v", trs)
	}

	var trFunc = chain.Tr
	if c.complement {
		trFunc = chain.Complement
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
			rn, _ := trFunc(in)
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

type trPred func(rune) bool

// trMap translates a rune to another rune
type trMap map[rune]rune

func (s trMap) in(in rune) bool {
	_, ok := s[in]
	return ok
}

// tr interface
func (s trMap) Tr(in rune) (rune, bool) {
	to, ok := s[in]
	return to, ok
}
func (s trMap) Complement(in rune) (rune, bool) {
	panic("trMap --complement is not yet supported")
}

// [:alnum:]
func alnum(in rune) bool {
	return unicode.IsLetter(in) || unicode.IsDigit(in)
}

// [:alpha:]
func alpha(in rune) bool {
	return unicode.IsLetter(in)
}

// [:blank:]
func blank(in rune) bool {
	return in == ' ' || in == '\t'
}

// [:cntrl:]
func cntrl(in rune) bool {
	return unicode.IsControl(in)
}

// [:digit:]
func digit(in rune) bool {
	return unicode.IsDigit(in)
}

// [:graph:]
func graph(in rune) bool {
	return unicode.IsPrint(in) && in != ' '
}

// [:lower:]
func lower(in rune) bool {
	return unicode.IsLower(in)
}

// [:prnt:]
func prnt(in rune) bool {
	return unicode.IsPrint(in)
}

// [:punct:]
func punct(in rune) bool {
	return unicode.IsPunct(in)
}

// [:space:]
func space(in rune) bool {
	return unicode.Is(unicode.White_Space, in)
}

// [:upper:]
func upper(in rune) bool {
	return unicode.IsUpper(in)
}

// [:xdigit:]
func xdigit(in rune) bool {
	return unicode.IsDigit(in) || (in >= 'a' && in <= 'f') || (in >= 'A' && in <= 'F')
}

// delTr implements tr interface for --delete and --delete --complement operations
type delTr struct {
	pred trPred
	name string
}

func (t delTr) Tr(in rune) (rune, bool) {
	if ok := t.pred(in); ok {
		return -1, true
	}
	return in, false
}

func (t delTr) Complement(in rune) (rune, bool) {
	if ok := t.pred(in); ok {
		return in, true
	}
	return -1, false
}

type tr interface {
	// Tr translate rune to other rune and returns true if it was done
	// -1 means rune is not going to be written
	Tr(rune) (rune, bool)
	Complement(rune) (rune, bool)
}

type chain struct {
	trs []tr
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

func (t chain) Complement(in rune) (rune, bool) {
	var dst rune
	for _, tr := range t.trs {
		var found bool
		dst, found = tr.Complement(in)
		if found {
			return dst, true
		}
	}
	// pass
	return dst, true
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

/*
   #########       parse command line      #################
*/

var trClasses = map[string]trPred{
	"alnum":  alnum,
	"alpha":  alpha,
	"blank":  blank,
	"cntrl":  cntrl,
	"digit":  digit,
	"graph":  graph,
	"lower":  lower,
	"print":  prnt,
	"punct":  punct,
	"space":  space,
	"upper":  upper,
	"xdigit": xdigit,
}

// makeDelChain parse ARRAY1 to generate a proper tr chain for --delete
func (c Tr) makeDelChain(array1 string) ([]tr, error) {
	sprintf := func(string, ...any) string { return "" }
	if c.debug {
		if c.complement {
			sprintf = func(f string, a ...any) string { return fmt.Sprintf("! "+f, a) }
		} else {
			sprintf = func(f string, a ...any) string { return fmt.Sprintf(f, a) }
		}
	}

	ret := make([]tr, 0, 10)
	globalSet := make(trMap)

	in := newRunes(array1)

	for idx := 0; idx < len(in); idx++ {

		if in.at(idx) == '\\' {
			goto singleChar
		}

		if klass, next := in.klass(idx); klass != "" {
			in, ok := trClasses[klass]
			if !ok {
				return nil, fmt.Errorf("invalid character class %q", klass)
			}
			ret = append(ret, delTr{
				pred: in,
				name: sprintf("[:%s:]", klass)},
			)
			idx = next
			continue
		}

		if equiv, next := in.equiv(idx); equiv != -1 {
			idx = next
			globalSet[equiv] = -1
			continue
		}

		if from, to, next := in.set(idx); next != idx {
			set := make(trMap, int(to-from))
			for rn := from; rn > to; rn++ {
				set[rn] = -1
			}
			ret = append(ret, delTr{pred: set.in, name: sprintf("%c-%c", to, from)})
			idx = next
			continue
		}

	singleChar:
		rn, next, err := in.charAt(idx)
		if err != nil {
			return nil, err
		}
		globalSet[rn] = -1
		idx = next
	}

	if len(globalSet) != 0 {
		name := ""
		if c.debug {
			var sb strings.Builder
			sb.Grow(len(globalSet))
			for rn := range globalSet {
				_, _ = writeRune(&sb, rn)
			}
			name = fmt.Sprintf("%+v", sb.String())
		}
		ret = append(ret, delTr{pred: globalSet.in, name: name})
	}

	return ret, nil
}

// makeTrChain parse ARRAY1 and ARRAY2 to generate a proper tr chain for translation
func (c Tr) makeTrChain(array1, array2 string) ([]tr, error) {
	if len(array1) == 0 {
		return nil, fmt.Errorf("array1 is empty")
	}
	if len(array2) == 0 {
		return nil, fmt.Errorf("array2 is empty")
	}

	if c.complement {
		panic("tr --complement is not yet implemented")
	}
	sprintf := func(string, ...any) string { return "" }
	if c.debug {
		if c.complement {
			panic("tr --complement is not yet implemented")
			//sprintf = func(f string, a ...any) string { return fmt.Sprintf("! "+f, a) }
		} else {
			sprintf = func(f string, a ...any) string { return fmt.Sprintf(f, a) }
		}
	}

	ret := make([]tr, 0, 10)
	globalSet := make(trMap)

	in1 := newRunes(array1)
	in2 := newRunes(array2)
	var lastIn2 rune

	idx2 := 0
	for idx1 := 0; idx1 < len(in1); idx1++ {
		if in1.at(idx1) == '\\' {
			goto singleChar
		}

		if klass, _ := in1.klass(idx1); klass != "" {
			sprintf(klass)
			panic("character classes for tr are not yet implemented")
		}

		if equiv, _ := in1.equiv(idx1); equiv != -1 {
			panic("equivalence classes for tr are not yet implemented")
		}

		if _, _, next := in1.set(idx1); next != idx1 {
			panic("ranges/sets for tr are not yet implemented")
		}

	singleChar:
		from, next, err := in1.charAt(idx1)
		if err != nil {
			return nil, err
		}
		idx1 = next

		switch in2.typ(idx2) {
		case NONE:
			// pass
		case CHAR:
			to, next, err := in2.charAt(idx2)
			if err != nil {
				return nil, err
			}
			lastIn2 = to
			idx2 = next + 1
		default:
			panic("translate to anything than a single char is not yet implemented")
		}
		globalSet[from] = lastIn2
		continue
	}

	if len(globalSet) != 0 {
		/*
			        name := ""
					if c.debug {
						var sb strings.Builder
						sb.Grow(len(globalSet))
						for rn := range globalSet {
							_, _ = writeRune(&sb, rn)
						}
						name = fmt.Sprintf("%+v", sb.String())
					}
		*/
		ret = append(ret, globalSet)
	}

	return ret, nil
}

// safeRunes is a helper for []rune, gracefully handle out of bound access
// and provides various helper parsers
type safeRunes []rune

func newRunes(s string) safeRunes {
	ret := make([]rune, utf8.RuneCountInString(s))
	for idx, rn := range s {
		ret[idx] = rn
	}
	return ret
}

func (s safeRunes) at(idx int) rune {
	if idx < 0 || idx >= len(s) {
		return -1
	}
	return s[idx]
}

type typ uint8

const (
	NONE  typ = 0
	CHAR      = 1
	KLASS     = 2
	EQUIV     = 3
)

func (s safeRunes) typ(idx int) typ {
	if idx < 0 || idx >= len(s) {
		return NONE
	}
	if klass, _ := s.klass(idx); klass != "" {
		return KLASS
	}
	if equiv, _ := s.equiv(idx); equiv != -1 {
		return EQUIV
	}
	if s.at(idx) == '\\' {
		if _, _, err := s.sequence(idx); err != nil {
			return CHAR
		}
	}
	return CHAR
}

func (s safeRunes) lookAhead(from int, needle rune) int {
	if from < 0 || from > len(s) {
		return -1
	}
	for idx := from; idx != len(s); idx++ {
		if s[idx] == needle {
			return idx
		}
	}
	return -1
}

// [:class:] support, returns a string as `class` and index of ] in slice
func (s safeRunes) klass(from int) (string, int) {
	if s.at(from) == '[' && s.at(from+1) == ':' {
		if colIdx := s.lookAhead(from+2, ':'); colIdx != -1 {
			if s.at(colIdx+1) == ']' {
				return s.substr(from+2, colIdx), colIdx + 1
			}
		}
	}
	return "", from
}

// [=C=] support, returns a string as `C` and a idex of ] in slice
func (s safeRunes) equiv(from int) (rune, int) {
	if s.at(from) == '[' && s.at(from+1) == '=' && s.at(from+3) == '=' && s.at(from+4) == ']' {
		return s[from+2], from + 4
	}
	return -1, from
}

func (s safeRunes) charAt(from int) (rune, int, error) {
	if s.at(from) == '\\' {
		return s.sequence(from)
	}
	rn := s.at(from)
	if rn == -1 {
		return rn, from, fmt.Errorf("charAt index %d our of range <0;%d>", from, len(s)-1)
	}
	return rn, from, nil
}

func (s safeRunes) sequence(from int) (rune, int, error) {
	if s.at(from) != '\\' {
		return -1, from, fmt.Errorf("can't interpret as a sequence: missing \\ at the start")
	}
	if octal(s.at(from+1)) && octal(s.at(from+2)) && octal(s.at(from+3)) {
		n, err := strconv.ParseInt(s.substr(from+1, from+4), 8, 32)
		if err != nil {
			return -1, from, fmt.Errorf("can't parse octal sequence: %w", err)
		}
		return rune(n), from + 3, nil
	}
	switch s.at(from + 1) {
	case '\\':
		return '\\', from + 1, nil
	case 'a':
		return '\a', from + 1, nil
	case 'b':
		return '\b', from + 1, nil
	case 'f':
		return '\f', from + 1, nil
	case 'n':
		return '\n', from + 1, nil
	case 'r':
		return '\r', from + 1, nil
	case 't':
		return '\t', from + 1, nil
	case 'v':
		return '\v', from + 1, nil
	default:
		return -1, from, fmt.Errorf("can't interpret sequence \\%c", s.at(from+1))
	}
}

func octal(rn rune) bool {
	return rn >= '0' && rn <= '7'
}

// character set - returns rune from, rune to and index of last processed rune in slice
// characters must be in ascending order, so first rune is always smaller than second
func (s safeRunes) set(from int) (rune, rune, int) {
	if s.at(from+1) != '-' {
		return -1, -1, from
	}
	start := s.at(from)
	stop := s.at(from + 2)
	if start < stop {
		return start, stop, from + 2
	}
	return -1, -1, from
}

func (s safeRunes) substr(from, to int) string {
	if from >= to || from <= 0 || to > len(s) {
		return ""
	}
	var sb strings.Builder
	sb.Grow(to - from)
	for idx := from; idx != to; idx++ {
		_, err := sb.WriteRune(s[idx])
		if err != nil {
			return ""
		}
	}
	return sb.String()
}
