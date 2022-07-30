// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Ported from https://cs.opensource.google/go/go/+/refs/tags/go1.18.4:src/time/format.go
// by Michal Vyskocil michal.vyskocil@gmail.com

package internal

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

// Unit represents the units with multiplier suffixes. Is defined as float64 to support
// things like YottaByte and so. It is generalized version of time.Duration
type Unit float64

const (
	maxUnit float64 = math.MaxFloat64 / 10
)

var (
	ErrEmpty                = errors.New("empty")
	ErrInvalidNextCharacter = errors.New("expected number or decimal separator")
	ErrOverflow             = errors.New("overflow")
	ErrNoDigits             = errors.New("no digits")
)

// parseUnit was ported from time.ParseDuration, generalized to arbitrary suffixes
// and works with float64
// https://cs.opensource.google/go/go/+/refs/tags/go1.18.4:src/time/format.go;l=1511
// [-+]?([0-9]*(\.[0-9]*)?[a-z]+)+
func parseUnit(unitMap map[string]float64, s string) (Unit, error) {
	orig := s
	var d float64
	neg := false

	// Consume [-+]?
	if s != "" {
		c := s[0]
		if c == '-' || c == '+' {
			neg = c == '-'
			s = s[1:]
		}
	}
	// Special case: if all that is left is "0", this is zero.
	if s == "0" {
		return 0, nil
	}
	if s == "" {
		return 0, fmt.Errorf("invalid size %q: %w", orig, ErrEmpty)
	}

	for s != "" {
		var (
			v, f  float64     // integers before, after decimal point
			scale float64 = 1 // value = v + f/scale
		)

		var err error
		// The next character must be [0-9.]
		if !(s[0] == '.' || '0' <= s[0] && s[0] <= '9') {
			return 0, fmt.Errorf("invalid size %q: %w", orig, ErrInvalidNextCharacter)
		}
		// Consume [0-9]*
		pl := len(s)
		v, s, err = leadingInt(s)
		if err != nil {
			return 0, fmt.Errorf("invalid size %q: %w", orig, ErrOverflow)
		}
		pre := pl != len(s) // whether we consumed anything before a period

		// Consume (\.[0-9]*)?
		post := false
		if s != "" && s[0] == '.' {
			s = s[1:]
			pl := len(s)
			f, scale, s = leadingFraction(s)
			post = pl != len(s)
		}
		if !pre && !post {
			// no digits (e.g. ".s" or "-.s")
			return 0, fmt.Errorf("invalid size %q: %w", orig, ErrNoDigits)
		}
		// Consume unit.
		i := 0
		for ; i < len(s); i++ {
			c := s[i]
			if c == '.' || '0' <= c && c <= '9' {
				break
			}
		}
		var unit float64
		if i == 0 {
			unit = 1.0
		} else {
			u := s[:i]
			s = s[i:]
			var ok bool
			unit, ok = unitMap[u]
			if !ok {
				return 0, fmt.Errorf("unknown unit %q in size %q", u, orig)
			}
		}
		if v > maxUnit/unit {
			// overflow
			return 0, fmt.Errorf("invalid size %q: %w", orig, ErrOverflow)
		}
		v *= unit
		if f > 0 {
			// float64 is needed to be nanosecond accurate for fractions of hours.
			// v >= 0 && (f*unit/scale) <= 3.6e+12 (ns/h, h is the largest unit)
			v += float64(float64(f) * (float64(unit) / scale))
			if v > maxUnit {
				// overflow
				return 0, fmt.Errorf("invalid size %q: %w", orig, ErrOverflow)
			}
		}
		d += v
		if d > maxUnit {
			return 0, fmt.Errorf("invalid size %q: %w", orig, ErrOverflow)
		}
	}

	if neg {
		return -Unit(d), nil
	}
	if d > maxUnit-1 {
		return 0, fmt.Errorf("invalid size %q: %w", orig, ErrOverflow)
	}
	return Unit(d), nil
}

var errLeadingInt = errors.New("bad [0-9]*") // never printed

// leadingInt consumes the leading [0-9]* from s.
func leadingInt(s string) (x float64, rem string, err error) {
	i := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
	}
	if i == 0 {
		return 0, s[0:], nil
	}
	x, err = strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return 0, "", errLeadingInt
	}
	return x, s[i:], nil
}

// leadingFraction consumes the leading [0-9]* from s.
// It is used only for fractions, so does not return an error on overflow,
// it just stops accumulating precision.
func leadingFraction(s string) (x float64, scale float64, rem string) {
	i := 0
	scale = 1
	overflow := false
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		if overflow {
			continue
		}
		if x > (maxUnit-1)/10 {
			// It's possible for overflow to give a positive number, so take care.
			overflow = true
			continue
		}
		y := x*10 + float64(c) - '0'
		if y > maxUnit {
			overflow = true
			continue
		}
		x = y
		scale *= 10
	}
	return x, scale, s[i:]
}

// Byte is a size of disk/memory/buffer capacities
type Byte float64

// Byte may have a multiplier suffix: b 512, kB 1000, K 1024, MB 1000*1000, M
// 1024*1024, GB 1000*1000*1000, G 1024*1024*1024, and so on for T, P, E, Z, Y.
// Binary prefixes can be used, too: KiB=K, MiB=M, and so on.
// see https://man7.org/linux/man-pages/man1/head.1.html
const (
	Block     Byte = 512              // b
	KiloByte       = 1000             // kB
	KibiByte       = 1024             // K/KiB
	MegaByte       = 1000 * KiloByte  // MB
	MebiByte       = 1024 * KibiByte  // M/MiB
	GigaByte       = 1000 * MegaByte  // GB
	GibiByte       = 1024 * MebiByte  // G/GiB
	TeraByte       = 1000 * GigaByte  // TB
	TebiByte       = 1024 * GibiByte  // T/TiB
	PetaByte       = 1000 * TeraByte  // PB
	PebiByte       = 1024 * TebiByte  // P/PiB
	ExaByte        = 1000 * PetaByte  // ZB
	ExbiByte       = 1024 * PebiByte  // Z/ZiB
	ZettaByte      = 1000 * ExaByte   // ZB
	ZebiByte       = 1024 * ExbiByte  // Z/ZiB
	YottaByte      = 1000 * ZettaByte // YB
	YobiByte       = 1024 * ZebiByte  // Y/YiB
)

var ByteSuffixes = map[string]float64{
	"b":   float64(Block),
	"kB":  float64(KiloByte),
	"K":   float64(KibiByte),
	"KiB": float64(KibiByte),
	"MB":  float64(MegaByte),
	"M":   float64(MebiByte),
	"MiB": float64(MebiByte),
	"GB":  float64(GigaByte),
	"G":   float64(GibiByte),
	"GiB": float64(GibiByte),
	"TB":  float64(TeraByte),
	"T":   float64(TebiByte),
	"TiB": float64(TebiByte),
	"PB":  float64(PetaByte),
	"P":   float64(PebiByte),
	"PiB": float64(PebiByte),
	"EB":  float64(ExaByte),
	"E":   float64(ExbiByte),
	"EiB": float64(ExbiByte),
	"ZB":  float64(ZettaByte),
	"Z":   float64(ZebiByte),
	"ZiB": float64(ZebiByte),
	"YB":  float64(YottaByte),
	"Y":   float64(YobiByte),
	"YiB": float64(YobiByte),
}

// ParseByte parses a byte definition.
// A byte string is a possibly signed sequence of
// decimal numbers, each with optional fraction and a unit suffix,
// such as "300K", "-1.5MiB" or "1TB45GB".
// Valid time units are "b" block 512, "kB" kilobyte 1000, "K", "KiB" kibibyte 1024
// and so on for M, G, T, P, E, Z, Y
func ParseByte(s string) (Byte, error) {
	u, err := parseUnit(ByteSuffixes, s)
	return Byte(u), err
}

func (b Byte) String() string {
	// TODO:
	panic("not yet implemented")
}
