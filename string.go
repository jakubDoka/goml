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
		if p.Failed() {
			return false
		}
		if fin {
			break
		}
		if concatSpace && afterSpace && r == ' ' && p.Source[p.I-1] != '\\' {
			continue
		}
		p.stringBuff = append(p.stringBuff, r)
	}

	if concatSpace { // cutting off the invisible characters
		l := len(p.stringBuff) - 1
	o:
		for l >= 0 {
			switch p.stringBuff[l] {
			case '\n', ' ', '\t', '\r':
				l--
			default:

				break o
			}
		}
		if l != -1 {
			p.stringBuff = p.stringBuff[:l+1]
		}
	}
	return true
}

// char turns a go string syntax to its data representation
func (p *Parser) char(ending byte) (r rune, end bool) {
	if !p.Advance() {
		if ending == '<' {
			return 0, true
		}
		p.Error(ErrStringNotTerminated)
		return
	}

	if p.Ch >= utf8.RuneSelf {
		var size int
		r, size = utf8.DecodeRune(p.Source[p.I:])
		if r == utf8.RuneError {
			p.Error(ErrInvalidRune)
			return
		}
		p.I += size - 1
		return
	}

	switch p.Ch {
	case '\\':
	// these runes are ignored, to add actual ones syntax has to be used
	case '\n', '\t', '\r':
		return ' ', false
	case '{':
		return p.stringTemplate(), false
	case ending:
		p.Advance()
		return 0, true
	default:
		return rune(p.Ch), false
	}

	if !p.Advance() {
		p.Error(ErrEscape.Incomplete)
		return
	}

	switch p.Ch {
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
	case '\\', ' ', ending:
		return rune(p.Ch), false
	}

	if p.Ch >= '0' && p.Ch <= '7' {
		return p.octal(), false
	}

	return p.hex(), false
}

// hex parses all of three possible hex rune syntaxes (\x00 \u0000 \U00000000)
func (p *Parser) hex() (r rune) {
	var n int
	switch p.Ch {
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
		if p.AdvanceOr(ErrEscape.Incomplete) {
			return
		}

		x, ok := unHex(p.Ch)
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
	v = rune(p.Ch) - '0'
	for j := 0; j < 2; j++ {
		if !p.Advance() {
			p.Error(ErrEscape.Incomplete)
			return
		}
		x := rune(p.Ch) - '0'
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

	return v
}

// stringTemplate registers string template if there is just one '{'
func (p *Parser) stringTemplate() (r rune) {
	if p.AdvanceOr(ErrStringNotTerminated) {
		return
	}
	if p.Ch == '{' {
		return '{'
	}
	//p.Degrade() // we advanced to get whats behind '{' so wh have to step back for template to read whole ident
	start := p.I - 1
	if !p.template(&p.parsed, stringTemplate) {
		return
	}
	p.Degrade() // Degrade again or we will end up with '}}'
	for i := start; i < p.I; {
		r, size := utf8.DecodeRune(p.Source[i:])
		p.stringBuff = append(p.stringBuff, r)
		i += size
	}
	return '}'
}
