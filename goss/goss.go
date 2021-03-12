package goss

import (
	"strconv"
	"strings"

	"github.com/jakubDoka/goml/core"
	"github.com/jakubDoka/sterr"
)

// Errors
var (
	ErrIdent         = sterr.New("expected identifier")
	ErrExpectedStart = sterr.New("expected start of field(':') after ident")
	ErrNumber        = sterr.New("failed to parse number(%s)")
	ErrExpectedByte  = sterr.New("expected '%c' but found '%c'")
	ErrNoValues      = sterr.New("field '%s' in '%s' has no values")
	ErrIncomplete    = sterr.New("style is incomplete, it has to be terminated with ';'")
	ErrExpectedValue = sterr.New("expected value after ' '")
)

// Parser parses the goss "language"
type Parser struct {
	cField, cStyle string

	parsed  Style
	val     interface{}
	valBuff []interface{}
	parser
}

// Parse expects file full of styles that have declared names
func (p *Parser) Parse(source []byte) (Styles, error) {
	p.Restart(source)
	stl := Styles{}
	for p.SkipSpace() {
		if !p.ident(&p.cStyle) {
			break
		}
		if !p.style(false) {
			break
		}
		stl[p.cStyle] = p.parsed
	}

	return stl, p.Err
}

// Style parses standalone ambiguous style
func (p *Parser) Style(source []byte) (Style, error) {
	p.Restart(source)
	p.cStyle = "inline"
	p.style(true)

	return p.parsed, p.Err
}

// Style
func (p *Parser) style(standalone bool) bool {
	p.parsed = Style{}
	for p.SkipSpace() {
		if p.Ch == ';' {
			return true
		}
		if !p.ident(&p.cField) {
			return false
		}

		if p.AdvanceOr(ErrIncomplete) {
			return false
		}

		p.valBuff = p.valBuff[:0]
	o:
		for {
			switch p.Ch {
			case ';':
				break o
			case ' ':
			default:
				p.Error(ErrExpectedByte.Args(' ', p.Ch))
				return false
			}

			if p.AdvanceOr(ErrIncomplete) {
				return false
			}
			if !p.value() {
				return false
			}

			p.valBuff = append(p.valBuff, p.val)
		}

		if len(p.valBuff) == 0 {
			p.Error(ErrNoValues.Args(p.cField, p.cStyle))
			return false
		}

		vals := make([]interface{}, len(p.valBuff))
		copy(vals, p.valBuff)
		p.parsed[p.cField] = vals
	}

	if !standalone {
		p.Error(ErrIncomplete)
	}
	return standalone
}

func (p *Parser) value() bool {
	start := p.I
	is, err := p.number()
	if is {
		if err != nil {
			p.Error(ErrNumber.Args(err))
			return false
		}
		return true
	}
	p.Set(start)
	ident := p.Ident()
	if len(ident) == 0 {
		p.Error(ErrExpectedValue)
		return false
	}

	switch v := string(ident); v {
	case "true":
		p.val = true
	case "false":
		p.val = false
	default:
		p.val = v
	}

	return true
}

func (p *Parser) number() (bool, error) {
	num := string(p.Number())
	if num == "" {
		return false, nil
	}

	var err error
	switch p.Ch {
	case 'f':
		p.val, err = strconv.ParseFloat(num, 64)
	case 'i':
		p.val, err = strconv.Atoi(num)
	case 'u':
		p.val, err = strconv.ParseUint(num, 10, 64)
	default:
		if strings.Contains(num, ".") {
			p.val, err = strconv.ParseFloat(num, 64)
		} else {
			p.val, err = strconv.Atoi(num)
		}
		return true, err
	}

	p.Advance()

	return true, err
}

func (p *Parser) ident(tgt *string) bool {
	ident := p.Ident()
	if len(ident) == 0 {
		p.Error(ErrIdent)
		return false
	}
	if p.Ch != ':' {
		p.Error(ErrExpectedByte.Args(':', p.Ch))
		return false
	}
	*tgt = string(ident)
	return true
}

// Styles is a collection of Styles
type Styles map[string]Style

// Add adds styles and owewrite the present ones
func (s Styles) Add(o Styles) {
	for k, v := range o {
		if val, ok := s[k]; ok {
			v.Overwrite(val)
		} else {
			s[k] = v
		}
	}
}

// Style is a parsed form of goss syntax
type Style map[string][]interface{}

// Ident returns first string under the property
func (s Style) Ident(key string) (string, bool) {
	val, ok := s[key]
	if !ok {
		return "", false
	}
	v, ok := val[0].(string)
	return v, ok
}

// Int returns first integer under the property
func (s Style) Int(key string) (int, bool) {
	val, ok := s[key]
	if !ok {
		return 0, false
	}
	v, ok := val[0].(int)
	return v, ok
}

// Float returns first float under the property
func (s Style) Float(key string) (float64, bool) {
	val, ok := s[key]
	if !ok {
		return 0, false
	}
	v, ok := val[0].(float64)
	return v, ok
}

// Uint returns first unsigned integer under the property
func (s Style) Uint(key string) (uint64, bool) {
	val, ok := s[key]
	if !ok {
		return 0, false
	}
	v, ok := val[0].(uint64)
	return v, ok
}

// Overwrite overwrites o by s, props can be overwritten and also added
func (s Style) Overwrite(o Style) {
	for k, v := range s {
		nv := make([]interface{}, len(v))
		copy(nv, v)
		o[k] = nv
	}
}

// Inherit makes as inherit all props that are at the same position, if
// s kay contains only one element == "inherit" the whole property of o is inherited
func (s Style) Inherit(o Style) {
	for k, v := range s {
		ov, ok := o[k]
		if !ok {
			continue
		}
		min := min(len(v), len(ov))
		for i := 0; i < min; i++ {
			val, ok := v[i].(string)
			if ok && val == "inherit" {
				if min == 1 && len(v) == 1 {
					cp := make([]interface{}, len(ov))
					copy(cp, ov)
					s[k] = cp
					break
				}
				v[i] = ov[i]
			}
		}
	}
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

type parser struct {
	core.Parser
}
