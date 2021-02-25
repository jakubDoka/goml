package goml

import (
	"unicode/utf8"

	"github.com/jakubDoka/sterr"
)

// string related errors
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
func (p *Parser) string(ending byte, concatSpace bool) bool {
	p.stringBuff = p.stringBuff[:0]
	var r rune
	var fin bool
	for {
		afterSpace := r == ' '
		r, fin = p.char(ending)
		if p.failed() {
			return false
		}
		if fin {
			break
		}
		if concatSpace && afterSpace && r == ' ' {
			continue
		}
		p.stringBuff = append(p.stringBuff, r)
	}

	l := len(p.stringBuff) - 1
	if concatSpace && l != -1 {
		if p.stringBuff[l] == ' ' {
			p.stringBuff = p.stringBuff[:l]
		}
	}
	return true
}

// char turns a go string syntax to its data representation
func (p *Parser) char(ending byte) (r rune, end bool) {
	if !p.advance() {
		p.error(ErrStringNotTerminated)
		return
	}

	if p.ch >= utf8.RuneSelf {
		var size int
		r, size = utf8.DecodeRune(p.source[p.i:])
		if r == utf8.RuneError {
			p.error(ErrInvalidRune)
			return
		}
		p.i += size - 1
		return
	}

	switch p.ch {
	case '\\':
	// these runes are ignored, to add actual ones syntax has to be used
	case '\n', '\t', '\r':
		return ' ', false
	case '{':
		return p.stringTemplate(), false
	case ending:
		p.advance()
		return 0, true
	default:
		return rune(p.ch), false
	}

	if !p.advance() {
		p.error(ErrEscape.Incomplete)
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
	case '\\', ending:
		return rune(p.ch), false
	}

	if p.ch >= '0' && p.ch <= '7' {
		return p.octal(), false
	}

	return p.hex(), false
}

// hex parses all of three possible hex rune syntaxes (\x00 \u0000 \U00000000)
func (p *Parser) hex() (r rune) {
	var n int
	switch p.ch {
	case 'x':
		n = 2
	case 'u':
		n = 4
	case 'U':
		n = 8
	default:
		p.error(ErrEscape.InvalidIdent)
		return
	}

	var v int
	for j := 0; j < n; j++ {
		if p.advanceOr(ErrEscape.Incomplete) {
			return
		}

		x, ok := unHex(p.ch)
		if !ok {
			p.error(ErrEscape.Illegal.Args("hex bytes"))
			return
		}
		v = v<<4 | int(x)
	}

	if v > utf8.MaxRune {
		p.error(ErrEscape.Overflow.Args(utf8.MaxRune))
		return
	}

	return rune(v)
}

// classic hex to byte
func unHex(c byte) (v byte, ok bool) {
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

// octal parses octal byte syntax assuming that first byte is between '0' - '7' (\000)
func (p *Parser) octal() (v rune) {
	v = rune(p.ch) - '0'
	for j := 0; j < 2; j++ {
		if !p.advance() {
			p.error(ErrEscape.Incomplete)
			return
		}
		x := rune(p.ch) - '0'
		if x < 0 || x > 7 {
			p.error(ErrEscape.Illegal.Args("bytes from '0' to '7'"))
			return
		}
		v = (v << 3) | x
	}

	if v > 255 {
		p.error(ErrEscape.Overflow.Args(225))
		return
	}

	return v
}

// stringTemplate registers string template if there is just one '{'
func (p *Parser) stringTemplate() (r rune) {
	if p.advanceOr(ErrStringNotTerminated) {
		return
	}
	if p.ch == '{' {
		return '{'
	}
	p.degrade() // we advanced to get whats behind '{' so wh have to step back for template to read whole ident
	start := p.i
	if !p.template(&p.parsed, stringTemplate) {
		return
	}
	p.degrade() // degrade again or we will end up with '}}'
	for i := start; i < p.i; {
		r, size := utf8.DecodeRune(p.source[i:])
		p.stringBuff = append(p.stringBuff, r)
		i += size
	}
	return '}'
}
