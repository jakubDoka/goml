package goml

import (
	"unicode/utf8"

	"github.com/jakubDoka/sterr"
)

// escape errors
var (
	ErrStringNotTerminated = sterr.New("string is not terminated")
	ErrInvalidRune         = sterr.New("rune is not terminated or cannot be decoded by utf8")
)

// ErrEscape contains escape related errors
var ErrEscape = struct {
	Incomplete, Illegal, Overflow, InvalidIdent sterr.Err
}{
	sterr.New("escape sequence is not terminated"),
	sterr.New("illegal character in escape, only %s are allowed"),
	sterr.New("escape value overflow, max is %d"),
	sterr.New("invalid escape identifier"),
}

// String parses started string into p.stringBuff
func (p *Parser) String() {
	p.stringBuff = p.stringBuff[:0]
	for r, finished := p.UnquoteChar(); !finished; r, finished = p.UnquoteChar() {
		if p.Failed() {
			return
		}
		p.stringBuff = append(p.stringBuff, r)
	}
}

// UnquoteChar turns a go string syntax to its data representation
func (p *Parser) UnquoteChar() (r rune, end bool) {
	if !p.Advance() {
		p.Error(ErrStringNotTerminated)
		return
	}

	if p.ch >= utf8.RuneSelf {
		var size int
		r, size = utf8.DecodeRune(p.source[p.i:])
		if r == utf8.RuneError {
			p.Error(ErrInvalidRune)
			return
		}
		p.i += size - 1
		return
	}

	switch p.ch {
	case '\\':
	case '"':
		p.Advance()
		return 0, true
	default:
		return rune(p.ch), false
	}

	if !p.Advance() {
		p.Error(ErrEscape.Incomplete)
		return
	}

	switch p.ch {
	case 'a':
		return '\a', false
	case 'b':
		return '\b', false
	case 'f':
		return '\f', false
	case 'n':
		return '\n', false
	case 'r':
		return '\r', false
	case 't':
		return '\t', false
	case 'v':
		return '\v', false
	case '\\', '"':
		return rune(p.ch), false
	}

	if p.ch >= '0' && p.ch <= '7' {
		v := rune(p.ch) - '0'
		for j := 0; j < 2; j++ {
			if !p.Advance() {
				p.Error(ErrEscape.Incomplete)
				return
			}
			x := rune(p.ch) - '0'
			if x < 0 || x > 7 {
				p.Error(ErrEscape.Illegal.Args("bytes from '0' to '7'"))
				return
			}
			v = (v << 3) | x
		}
		if v > 255 {
			p.Error(ErrEscape.Overflow.Args(225))
			return
		}
		return v, false
	}

	var n int
	switch p.ch {
	case 'x':
		n = 2
	case 'u':
		n = 4
	case 'U':
		n = 8
	default:
		p.Error(ErrEscape.InvalidIdent)
		return
	}

	var v int
	for j := 0; j < n; j++ {
		if !p.Advance() {
			p.Error(ErrEscape.Incomplete)
		}
		x, ok := unHex(p.ch)
		if !ok {
			p.Error(ErrEscape.Illegal.Args("hex bytes"))
			return
		}
		v = v<<4 | int(x)
	}

	if v > utf8.MaxRune {
		p.Error(ErrEscape.Overflow.Args(utf8.MaxRune))
		return
	}

	return rune(v), false
}

func unHex(b byte) (v rune, ok bool) {
	c := rune(b)
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return
}
